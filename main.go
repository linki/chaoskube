package main

import (
	"os"
	"time"

	"github.com/go-kit/kit/log"
	"gopkg.in/alecthomas/kingpin.v2"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/linki/chaoskube/chaoskube"
	"github.com/linki/chaoskube/util"
)

const (
	appName = "chaoskube"
	image   = "quay.io/linki/chaoskube"
	version = "v0.5.0"
)

var (
	labelString string
	annString   string
	nsString    string
	kubeconfig  string
	interval    time.Duration
	inCluster   bool
	deploy      bool
	dryRun      bool
	debug       bool
)

func init() {
	kingpin.Flag("labels", "A set of labels to restrict the list of affected pods. Defaults to everything.").Default(labels.Everything().String()).StringVar(&labelString)
	kingpin.Flag("annotations", "A set of annotations to restrict the list of affected pods. Defaults to everything.").Default(labels.Everything().String()).StringVar(&annString)
	kingpin.Flag("namespaces", "A set of namespaces to restrict the list of affected pods. Defaults to everything.").Default(v1.NamespaceAll).StringVar(&nsString)
	kingpin.Flag("kubeconfig", "Path to a kubeconfig file").Default(clientcmd.RecommendedHomeFile).StringVar(&kubeconfig)
	kingpin.Flag("interval", "Interval between Pod terminations").Short('i').Default("10m").DurationVar(&interval)
	kingpin.Flag("in-cluster", "If true, finds the Kubernetes cluster from the environment").Short('c').BoolVar(&inCluster)
	kingpin.Flag("deploy", "If true, deploys chaoskube in the current cluster with the provided configuration").Short('d').BoolVar(&deploy)
	kingpin.Flag("dry-run", "If true, don't actually do anything.").Default("true").BoolVar(&dryRun)
	kingpin.Flag("debug", "Enable debug logging.").BoolVar(&debug)
}

func main() {
	kingpin.Version(version)
	kingpin.Parse()

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "time", log.Valuer(func() interface{} {
		return time.Now().UTC().Format(time.RFC3339)
	}), "caller", log.DefaultCaller)

	if dryRun {
		logger.Log("msg", "Dry run enabled. I won't kill anything. Use --no-dry-run when you're ready.")
	}

	client, err := newClient(logger)
	if err != nil {
		logger.Log("error", err.Error())
		os.Exit(1)
	}

	if deploy {
		logger.Log("msg", "deploying container", "image", image, "version", version)

		manifest := generateManifest()

		deployment := client.Extensions().Deployments(manifest.Namespace)

		_, err := deployment.Get(manifest.Name)
		if err != nil {
			_, err = deployment.Create(manifest)
		} else {
			_, err = deployment.Update(manifest)
		}
		if err != nil {
			logger.Log("error", err.Error())
			os.Exit(1)
		}

		logger.Log("msg", "deployed container", "image", image, "version", version)
		os.Exit(0)
	}

	labelSelector, err := labels.Parse(labelString)
	if err != nil {
		logger.Log("error", err.Error())
		os.Exit(1)
	}

	annotations, err := labels.Parse(annString)
	if err != nil {
		logger.Log("error", err.Error())
		os.Exit(1)
	}

	namespaces, err := labels.Parse(nsString)
	if err != nil {
		logger.Log("error", err.Error())
		os.Exit(1)
	}

	logger.Log("msg", "filter pods", "labels", labelSelector.String(), "annotations", annotations.String(), "namespaces", namespaces.String())

	chaoskube := chaoskube.New(client, labelSelector, annotations, namespaces, logger, dryRun, time.Now().UTC().UnixNano())

	for {
		if err := chaoskube.TerminateVictim(); err != nil {
			logger.Log("error", err.Error())
			os.Exit(1)
		}

		logger.Log("msg", "sleeping", "interval", interval)
		time.Sleep(interval)
	}
}

func newClient(logger chaoskube.Logger) (*kubernetes.Clientset, error) {
	var (
		config *rest.Config
		err    error
	)

	if inCluster {
		config, err = rest.InClusterConfig()
		logger.Log("msg", "Using in-cluster config.")
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		logger.Log("kubeconfig", kubeconfig)
	}
	if err != nil {
		return nil, err
	}
	logger.Log("cluster", config.Host)

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func generateManifest() *v1beta1.Deployment {
	// modifies flags for deployment
	args := append(os.Args[1:], "--in-cluster")
	args = util.StripElements(args, "--kubeconfig", "--deploy")

	return &v1beta1.Deployment{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Deployment",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      appName,
			Namespace: v1.NamespaceDefault,
			Labels: map[string]string{
				"app":      appName,
				"heritage": appName,
			},
		},
		Spec: v1beta1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"app": appName,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Name:  appName,
							Image: image + ":" + version,
							Args:  args,
						},
					},
				},
			},
		},
	}
}

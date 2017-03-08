package main

import (
	"net/http"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/prometheus/client_golang/prometheus/promhttp"

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

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	if dryRun {
		log.Infof("Dry run enabled. I won't kill anything. Use --no-dry-run when you're ready.")
	}

	client, err := newClient()
	if err != nil {
		log.Fatal(err)
	}

	if deploy {
		log.Debugf("Deploying %s:%s", image, version)

		manifest := generateManifest()

		deployment := client.Extensions().Deployments(manifest.Namespace)

		_, err := deployment.Get(manifest.Name)
		if err != nil {
			_, err = deployment.Create(manifest)
		} else {
			_, err = deployment.Update(manifest)
		}
		if err != nil {
			log.Fatal(err)
		}

		log.Infof("Deployed %s:%s", image, version)
		os.Exit(0)
	}

	go serveMetrics()

	labelSelector, err := labels.Parse(labelString)
	if err != nil {
		log.Fatal(err)
	}

	if !labelSelector.Empty() {
		log.Infof("Filtering pods by labels: %s", labelSelector.String())
	}

	annotations, err := labels.Parse(annString)
	if err != nil {
		log.Fatal(err)
	}

	if !annotations.Empty() {
		log.Infof("Filtering pods by annotations: %s", annotations.String())
	}

	namespaces, err := labels.Parse(nsString)
	if err != nil {
		log.Fatal(err)
	}

	if !namespaces.Empty() {
		log.Infof("Filtering pods by namespaces: %s", namespaces.String())
	}

	chaoskube := chaoskube.New(client, labelSelector, annotations, namespaces, log.StandardLogger(), dryRun, time.Now().UTC().UnixNano())

	for {
		if err := chaoskube.TerminateVictim(); err != nil {
			log.Error(err)
		}

		log.Debugf("Sleeping for %s...", interval)
		time.Sleep(interval)
	}
}

func newClient() (*kubernetes.Clientset, error) {
	var (
		config *rest.Config
		err    error
	)

	if inCluster {
		config, err = rest.InClusterConfig()
		log.Debug("Using in-cluster config.")
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		log.Debugf("Using current context from kubeconfig at %s.", kubeconfig)
	}
	if err != nil {
		return nil, err
	}
	log.Infof("Targeting cluster at %s", config.Host)

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

func serveMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9099", nil))
}

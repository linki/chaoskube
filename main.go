package main

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/linki/chaoskube/chaoskube"
	"github.com/linki/chaoskube/util"
)

var (
	annString       string
	debug           bool
	dryRun          bool
	excludeWeekends bool
	inCluster       bool
	interval        time.Duration
	kubeconfig      string
	labelString     string
	master          string
	nsString        string
	percentage      float64
	runFrom         string
	runUntil        string
	version         string
)

func init() {
	kingpin.Flag("annotations", "A set of annotations to restrict the list of affected pods. Defaults to everything.").StringVar(&annString)
	kingpin.Flag("debug", "Enable debug logging.").BoolVar(&debug)
	kingpin.Flag("dry-run", "If true, don't actually do anything.").Default("true").BoolVar(&dryRun)
	kingpin.Flag("excludeWeekends", "Do not run on weekends").BoolVar(&excludeWeekends)
	kingpin.Flag("interval", "Interval between Pod terminations").Default("1m").DurationVar(&interval)
	kingpin.Flag("kubeconfig", "Path to a kubeconfig file").StringVar(&kubeconfig)
	kingpin.Flag("labels", "A set of labels to restrict the list of affected pods. Defaults to everything.").StringVar(&labelString)
	kingpin.Flag("master", "The address of the Kubernetes cluster to target").StringVar(&master)
	kingpin.Flag("namespaces", "A set of namespaces to restrict the list of affected pods. Defaults to everything.").StringVar(&nsString)
	kingpin.Flag("percentage", "How likely should a pod be killed every single run").Default("0.0").Float64Var(&percentage)
	kingpin.Flag("run-from", "Start chaoskube daily at hours:minutes, e.g. 9:00").Default("0:00").StringVar(&runFrom)
	kingpin.Flag("run-until", "Stop chaoskube daily at hours:minutes, e.g. 17:00").Default("0:00").StringVar(&runUntil)
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

	labelSelector, err := labels.Parse(labelString)
	if err != nil {
		log.Fatal(err)
	}

	annotations, err := labels.Parse(annString)
	if err != nil {
		log.Fatal(err)
	}

	namespaces, err := labels.Parse(nsString)
	if err != nil {
		log.Fatal(err)
	}

	if !labelSelector.Empty() {
		log.Infof("Filtering pods by labels: %s", labelSelector.String())
	}

	if !annotations.Empty() {
		log.Infof("Filtering pods by annotations: %s", annotations.String())
	}

	if !namespaces.Empty() {
		log.Infof("Filtering pods by namespaces: %s", namespaces.String())
	}

	chaoskube := chaoskube.New(
		client,
		labelSelector,
		annotations,
		namespaces,
		log.StandardLogger(),
		dryRun,
		time.Now().UTC().UnixNano(),
	)

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			if util.ShouldRunNow(excludeWeekends, runFrom, runUntil) {
				candidates, err := chaoskube.Candidates()
				if err != nil {
					log.Fatal(err)
				}
				for _, candidate := range candidates {
					if util.PodShouldDie(candidate, interval, percentage) {
						chaoskube.DeletePod(candidate)
					}
				}
			}
		}
	}
}

func newClient() (*kubernetes.Clientset, error) {
	if kubeconfig == "" {
		if _, err := os.Stat(clientcmd.RecommendedHomeFile); err == nil {
			kubeconfig = clientcmd.RecommendedHomeFile
		}
	}

	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
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

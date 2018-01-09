package main

import (
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/linki/chaoskube/chaoskube"
)

var (
	labelString      string
	annString        string
	nsString         string
	excludedWeekdays string
	master           string
	kubeconfig       string
	interval         time.Duration
	inCluster        bool
	dryRun           bool
	debug            bool
	version          string
)

func init() {
	kingpin.Flag("labels", "A set of labels to restrict the list of affected pods. Defaults to everything.").StringVar(&labelString)
	kingpin.Flag("annotations", "A set of annotations to restrict the list of affected pods. Defaults to everything.").StringVar(&annString)
	kingpin.Flag("namespaces", "A set of namespaces to restrict the list of affected pods. Defaults to everything.").StringVar(&nsString)
	kingpin.Flag("excluded-weekdays", "A list of weekdays when termination is suspended, e.g. sat,sun").StringVar(&excludedWeekdays)
	kingpin.Flag("master", "The address of the Kubernetes cluster to target").StringVar(&master)
	kingpin.Flag("kubeconfig", "Path to a kubeconfig file").StringVar(&kubeconfig)
	kingpin.Flag("interval", "Interval between Pod terminations").Default("10m").DurationVar(&interval)
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

	parsedWeekdays := parseWeekdays(excludedWeekdays)
	if len(parsedWeekdays) > 0 {
		log.Infof("Excluding weekdays: %s", parsedWeekdays)
	}

	chaoskube := chaoskube.New(
		client,
		labelSelector,
		annotations,
		namespaces,
		parsedWeekdays,
		log.StandardLogger(),
		dryRun,
		time.Now().UTC().UnixNano(),
	)

	for {
		if err := chaoskube.TerminateVictim(); err != nil {
			log.Fatal(err)
		}

		log.Debugf("Sleeping for %s...", interval)
		time.Sleep(interval)
	}
}

func parseWeekdays(weekdays string) []time.Weekday {
	var days = map[string]time.Weekday{
		"sun": time.Sunday,
		"mon": time.Monday,
		"tue": time.Tuesday,
		"wed": time.Wednesday,
		"thu": time.Thursday,
		"fri": time.Friday,
		"sat": time.Saturday,
	}

	parsedWeekdays := []time.Weekday{}
	for _, wd := range strings.Split(weekdays, ",") {
		if day, ok := days[strings.TrimSpace(strings.ToLower(wd))]; ok {
			parsedWeekdays = append(parsedWeekdays, day)
		}
	}
	return parsedWeekdays
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

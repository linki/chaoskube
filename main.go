package main

import (
	"math/rand"
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
	labelString        string
	annString          string
	nsString           string
	excludedWeekdays   string
	excludedTimesOfDay string
	timezone           string
	master             string
	kubeconfig         string
	interval           time.Duration
	dryRun             bool
	debug              bool
	version            string
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	kingpin.Flag("labels", "A set of labels to restrict the list of affected pods. Defaults to everything.").StringVar(&labelString)
	kingpin.Flag("annotations", "A set of annotations to restrict the list of affected pods. Defaults to everything.").StringVar(&annString)
	kingpin.Flag("namespaces", "A set of namespaces to restrict the list of affected pods. Defaults to everything.").StringVar(&nsString)
	kingpin.Flag("excluded-weekdays", "A list of weekdays when termination is suspended, e.g. sat,sun").StringVar(&excludedWeekdays)
	kingpin.Flag("excluded-times-of-day", "A list of time periods of a day when termination is suspended, e.g. 22:00-08:00").StringVar(&excludedTimesOfDay)
	kingpin.Flag("timezone", "The timezone by which to interpret the excluded weekdays and times of day, e.g. UTC, Local, Europe/Berlin. Defaults to UTC.").Default("UTC").StringVar(&timezone)
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

	log.WithFields(log.Fields{
		"labels":             labelString,
		"annotations":        annString,
		"namespaces":         nsString,
		"excludedWeekdays":   excludedWeekdays,
		"excludedTimesOfDay": excludedTimesOfDay,
		"timezone":           timezone,
		"master":             master,
		"kubeconfig":         kubeconfig,
		"interval":           interval,
		"dryRun":             dryRun,
		"debug":              debug,
		"version":            version,
	}).Debugf("reading config")

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

	log.WithFields(log.Fields{
		"labels":      labelSelector,
		"annotations": annotations,
		"namespaces":  namespaces,
	}).Infof("filtering candidates")

	parsedWeekdays := util.ParseWeekdays(excludedWeekdays)
	parsedTimesOfDay, err := util.ParseTimePeriods(excludedTimesOfDay)
	if err != nil {
		log.Fatal(err)
	}

	log.WithFields(log.Fields{
		"weekdays":   parsedWeekdays,
		"timesOfDay": parsedTimesOfDay,
	}).Infof("setting quiet times")

	parsedTimezone, err := time.LoadLocation(timezone)
	if err != nil {
		log.Fatal(err)
	}
	timezoneName, offset := time.Now().In(parsedTimezone).Zone()

	log.WithFields(log.Fields{
		"name":     timezoneName,
		"location": parsedTimezone,
		"offset":   offset / int(time.Hour/time.Second),
	}).Infof("setting timezone")

	chaoskube := chaoskube.New(
		client,
		labelSelector,
		annotations,
		namespaces,
		parsedWeekdays,
		parsedTimesOfDay,
		parsedTimezone,
		log.StandardLogger(),
		dryRun,
	)

	for {
		if err := chaoskube.TerminateVictim(); err != nil {
			log.WithField("err", err).Error("failed to terminate victim")
		}

		log.WithField("duration", interval).Debug("sleeping")
		time.Sleep(interval)
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

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	serverVersion, err := client.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"url":           config.Host,
		"serverVersion": serverVersion,
	}).Info("connecting to cluster")

	return client, nil
}

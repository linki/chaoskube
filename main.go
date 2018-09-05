package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	restclient "k8s.io/client-go/rest"

	"github.com/linki/chaoskube/chaoskube"
	"github.com/linki/chaoskube/util"
	"strings"
)

var (
	version = "undefined"
)

var (
	labelString        string
	annString          string
	nsString           string
	excludedWeekdays   string
	excludedTimesOfDay string
	excludedDaysOfYear string
	timezone           string
	minimumAge         time.Duration
	master             string
	kubeconfig         string
	interval           time.Duration
	dryRun             bool
	debug              bool
	metricsAddress     string
	exec               string
	execContainer      string
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	kingpin.Flag("labels", "A set of labels to restrict the list of affected pods. Defaults to everything.").StringVar(&labelString)
	kingpin.Flag("annotations", "A set of annotations to restrict the list of affected pods. Defaults to everything.").StringVar(&annString)
	kingpin.Flag("namespaces", "A set of namespaces to restrict the list of affected pods. Defaults to everything.").StringVar(&nsString)
	kingpin.Flag("excluded-weekdays", "A list of weekdays when termination is suspended, e.g. Sat,Sun").StringVar(&excludedWeekdays)
	kingpin.Flag("excluded-times-of-day", "A list of time periods of a day when termination is suspended, e.g. 22:00-08:00").StringVar(&excludedTimesOfDay)
	kingpin.Flag("excluded-days-of-year", "A list of days of a year when termination is suspended, e.g. Apr1,Dec24").StringVar(&excludedDaysOfYear)
	kingpin.Flag("timezone", "The timezone by which to interpret the excluded weekdays and times of day, e.g. UTC, Local, Europe/Berlin. Defaults to UTC.").Default("UTC").StringVar(&timezone)
	kingpin.Flag("minimum-age", "Minimum age of pods to consider for termination").Default("0s").DurationVar(&minimumAge)
	kingpin.Flag("master", "The address of the Kubernetes cluster to target").StringVar(&master)
	kingpin.Flag("kubeconfig", "Path to a kubeconfig file").StringVar(&kubeconfig)
	kingpin.Flag("interval", "Interval between Pod terminations").Default("10m").DurationVar(&interval)
	kingpin.Flag("dry-run", "If true, don't actually do anything.").Default("true").BoolVar(&dryRun)
	kingpin.Flag("exec", "Execute the given terminal command on victim pods, rather than deleting pods, eg killall -9 bash").StringVar(&exec)
	kingpin.Flag("exec-container", "Name of container to run --exec command in, defaults to first container in spec").Default("").StringVar(&execContainer)
	kingpin.Flag("debug", "Enable debug logging.").BoolVar(&debug)
	kingpin.Flag("metrics-address", "Listening address for metrics handler").Default(":8080").StringVar(&metricsAddress)
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
		"excludedDaysOfYear": excludedDaysOfYear,
		"timezone":           timezone,
		"minimumAge":         minimumAge,
		"master":             master,
		"kubeconfig":         kubeconfig,
		"interval":           interval,
		"dryRun":             dryRun,
		"exec":               exec,
		"execContainer":      execContainer,
		"debug":              debug,
		"metricsAddress":     metricsAddress,
	}).Info("reading config")

	log.WithFields(log.Fields{
		"version":  version,
		"dryRun":   dryRun,
		"interval": interval,
	}).Info("starting up")

	config, err := newConfig()
	if err != nil {
		log.WithField("err", err).Fatal("failed to determine k8s client config")
	}

	client, err := newClient(config)
	if err != nil {
		log.WithField("err", err).Fatal("failed to connect to cluster")
	}

	var (
		labelSelector = parseSelector(labelString)
		annotations   = parseSelector(annString)
		namespaces    = parseSelector(nsString)
	)

	log.WithFields(log.Fields{
		"labels":      labelSelector,
		"annotations": annotations,
		"namespaces":  namespaces,
		"minimumAge":  minimumAge,
	}).Info("setting pod filter")

	parsedWeekdays := util.ParseWeekdays(excludedWeekdays)
	parsedTimesOfDay, err := util.ParseTimePeriods(excludedTimesOfDay)
	if err != nil {
		log.WithFields(log.Fields{
			"timesOfDay": excludedTimesOfDay,
			"err":        err,
		}).Fatal("failed to parse times of day")
	}
	parsedDaysOfYear, err := util.ParseDays(excludedDaysOfYear)
	if err != nil {
		log.WithFields(log.Fields{
			"daysOfYear": excludedDaysOfYear,
			"err":        err,
		}).Fatal("failed to parse days of year")
	}

	log.WithFields(log.Fields{
		"weekdays":   parsedWeekdays,
		"timesOfDay": parsedTimesOfDay,
		"daysOfYear": formatDays(parsedDaysOfYear),
	}).Info("setting quiet times")

	parsedTimezone, err := time.LoadLocation(timezone)
	if err != nil {
		log.WithFields(log.Fields{
			"timeZone": timezone,
			"err":      err,
		}).Fatal("failed to detect time zone")
	}
	timezoneName, offset := time.Now().In(parsedTimezone).Zone()

	log.WithFields(log.Fields{
		"name":     timezoneName,
		"location": parsedTimezone,
		"offset":   offset / int(time.Hour/time.Second),
	}).Info("setting timezone")

	var action chaoskube.ChaosAction
	if dryRun {
		action = chaoskube.NewDryRunAction()
	} else if len(exec) > 0 {
		action = chaoskube.NewExecAction(client.CoreV1().RESTClient(), config, execContainer, strings.Split(exec, " "))
	} else {
		action = chaoskube.NewDeletePodAction(client)
	}

	chaoskube := chaoskube.New(
		client,
		labelSelector,
		annotations,
		namespaces,
		parsedWeekdays,
		parsedTimesOfDay,
		parsedDaysOfYear,
		parsedTimezone,
		minimumAge,
		log.StandardLogger(),
		action,
	)

	if metricsAddress != "" {
		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/healthz",
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, "OK")
			})
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html>
					<head><title>chaoskube</title></head>
					<body>
					<h1>chaoskube</h1>
					<p><a href="/metrics">Metrics</a></p>
					<p><a href="/healthz">Health Check</a></p>
					</body>
					</html>`))
		})
		go func() {
			if err := http.ListenAndServe(metricsAddress, nil); err != nil {
				log.WithFields(log.Fields{
					"err": err,
				}).Fatal("failed to start HTTP server")
			}
		}()
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-done
		cancel()
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	chaoskube.Run(ctx, ticker.C)
}

func newConfig() (*restclient.Config, error) {
	if kubeconfig == "" {
		if _, err := os.Stat(clientcmd.RecommendedHomeFile); err == nil {
			kubeconfig = clientcmd.RecommendedHomeFile
		}
	}

	log.WithFields(log.Fields{
		"kubeconfig": kubeconfig,
		"master":     master,
	}).Debug("using cluster config")

	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func newClient(config *restclient.Config) (*kubernetes.Clientset, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	serverVersion, err := client.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"master":        config.Host,
		"serverVersion": serverVersion,
	}).Info("connected to cluster")

	return client, nil
}

func parseSelector(str string) labels.Selector {
	selector, err := labels.Parse(str)
	if err != nil {
		log.WithFields(log.Fields{
			"selector": str,
			"err":      err,
		}).Fatal("failed to parse selector")
	}
	return selector
}

func formatDays(days []time.Time) []string {
	formattedDays := make([]string, 0, len(days))
	for _, d := range days {
		formattedDays = append(formattedDays, d.Format(util.YearDay))
	}
	return formattedDays
}

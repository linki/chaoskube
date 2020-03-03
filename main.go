package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path"
	"regexp"
	"runtime"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/linki/chaoskube/chaoskube"
	"github.com/linki/chaoskube/notifier"
	"github.com/linki/chaoskube/terminator"
	"github.com/linki/chaoskube/util"
)

var (
	version = "undefined"
)

var (
	labelString        string
	annString          string
	nsString           string
	nsLabelString      string
	includedPodNames   *regexp.Regexp
	excludedPodNames   *regexp.Regexp
	excludedWeekdays   string
	excludedTimesOfDay string
	excludedDaysOfYear string
	timezone           string
	minimumAge         time.Duration
	maxKill            int
	master             string
	kubeconfig         string
	interval           time.Duration
	maxJitter          time.Duration
	dryRun             bool
	debug              bool
	metricsAddress     string
	gracePeriod        time.Duration
	logFormat          string
	logCaller          bool
	slackWebhook       string
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	klog.SetOutput(ioutil.Discard)

	kingpin.Flag("labels", "A set of labels to restrict the list of affected pods. Defaults to everything.").StringVar(&labelString)
	kingpin.Flag("annotations", "A set of annotations to restrict the list of affected pods. Defaults to everything.").StringVar(&annString)
	kingpin.Flag("namespaces", "A set of namespaces to restrict the list of affected pods. Defaults to everything.").StringVar(&nsString)
	kingpin.Flag("namespace-labels", "A set of labels to restrict the list of affected namespaces. Defaults to everything.").StringVar(&nsLabelString)
	kingpin.Flag("included-pod-names", "Regular expression that defines which pods to include. All included by default.").RegexpVar(&includedPodNames)
	kingpin.Flag("excluded-pod-names", "Regular expression that defines which pods to exclude. None excluded by default.").RegexpVar(&excludedPodNames)
	kingpin.Flag("excluded-weekdays", "A list of weekdays when termination is suspended, e.g. Sat,Sun").StringVar(&excludedWeekdays)
	kingpin.Flag("excluded-times-of-day", "A list of time periods of a day when termination is suspended, e.g. 22:00-08:00").StringVar(&excludedTimesOfDay)
	kingpin.Flag("excluded-days-of-year", "A list of days of a year when termination is suspended, e.g. Apr1,Dec24").StringVar(&excludedDaysOfYear)
	kingpin.Flag("timezone", "The timezone by which to interpret the excluded weekdays and times of day, e.g. UTC, Local, Europe/Berlin. Defaults to UTC.").Default("UTC").StringVar(&timezone)
	kingpin.Flag("minimum-age", "Minimum age of pods to consider for termination").Default("0s").DurationVar(&minimumAge)
	kingpin.Flag("max-kill", "Specifies the maximum number of pods to be terminated per interval.").Default("1").IntVar(&maxKill)
	kingpin.Flag("master", "The address of the Kubernetes cluster to target").StringVar(&master)
	kingpin.Flag("kubeconfig", "Path to a kubeconfig file").StringVar(&kubeconfig)
	kingpin.Flag("interval", "Interval between Pod terminations").Default("10m").DurationVar(&interval)
	kingpin.Flag("max-jitter", "The max duration of jitter to add to the interval").Default("0s").DurationVar(&maxJitter)
	kingpin.Flag("dry-run", "Don't actually kill any pod. Turned on by default. Turn off with `--no-dry-run`.").Default("true").BoolVar(&dryRun)
	kingpin.Flag("debug", "Enable debug logging.").BoolVar(&debug)
	kingpin.Flag("metrics-address", "Listening address for metrics handler").Default(":8080").StringVar(&metricsAddress)
	kingpin.Flag("grace-period", "Grace period to terminate Pods. Negative values will use the Pod's grace period.").Default("-1s").DurationVar(&gracePeriod)
	kingpin.Flag("log-format", "Specify the format of the log messages. Options are text and json. Defaults to text.").Default("text").EnumVar(&logFormat, "text", "json")
	kingpin.Flag("log-caller", "Include the calling function name and location in the log messages.").BoolVar(&logCaller)
	kingpin.Flag("slack-webhook", "The address of the slack webhook for notifications").StringVar(&slackWebhook)
}

func main() {
	kingpin.Version(version)
	kingpin.Parse()

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	switch logFormat {
	case "json":
		log.SetFormatter(&log.JSONFormatter{CallerPrettyfier: prettifyCaller})
	default:
		log.SetFormatter(&log.TextFormatter{CallerPrettyfier: prettifyCaller})
	}

	log.SetReportCaller(logCaller)

	log.WithFields(log.Fields{
		"labels":             labelString,
		"annotations":        annString,
		"namespaces":         nsString,
		"namespaceLabels":    nsLabelString,
		"includedPodNames":   includedPodNames,
		"excludedPodNames":   excludedPodNames,
		"excludedWeekdays":   excludedWeekdays,
		"excludedTimesOfDay": excludedTimesOfDay,
		"excludedDaysOfYear": excludedDaysOfYear,
		"timezone":           timezone,
		"minimumAge":         minimumAge,
		"maxKill":            maxKill,
		"master":             master,
		"kubeconfig":         kubeconfig,
		"interval":           interval,
		"maxJitter":          maxJitter,
		"dryRun":             dryRun,
		"debug":              debug,
		"metricsAddress":     metricsAddress,
		"gracePeriod":        gracePeriod,
		"logFormat":          logFormat,
		"slackWebhook":       slackWebhook,
	}).Debug("reading config")

	log.WithFields(log.Fields{
		"version":   version,
		"dryRun":    dryRun,
		"interval":  interval,
		"maxJitter": maxJitter,
	}).Info("starting up")

	client, err := newClient()
	if err != nil {
		log.WithField("err", err).Fatal("failed to connect to cluster")
	}

	var (
		labelSelector   = parseSelector(labelString)
		annotations     = parseSelector(annString)
		namespaces      = parseSelector(nsString)
		namespaceLabels = parseSelector(nsLabelString)
	)

	log.WithFields(log.Fields{
		"labels":           labelSelector,
		"annotations":      annotations,
		"namespaces":       namespaces,
		"namespaceLabels":  namespaceLabels,
		"includedPodNames": includedPodNames,
		"excludedPodNames": excludedPodNames,
		"minimumAge":       minimumAge,
		"maxKill":          maxKill,
	}).Info("setting pod filter")

	if interval <= maxJitter {
		log.WithFields(log.Fields{
			"interval":  interval,
			"maxJitter": maxJitter,
		}).Fatal("maxJitter must be less than interval")
	}

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
		"daysOfYear": util.FormatDays(parsedDaysOfYear),
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

	notifiers := createNotifier()

	chaoskube := chaoskube.New(
		client,
		labelSelector,
		annotations,
		namespaces,
		namespaceLabels,
		includedPodNames,
		excludedPodNames,
		parsedWeekdays,
		parsedTimesOfDay,
		parsedDaysOfYear,
		parsedTimezone,
		minimumAge,
		log.StandardLogger(),
		dryRun,
		terminator.NewDeletePodTerminator(client, log.StandardLogger(), gracePeriod),
		maxKill,
		notifiers,
	)

	if metricsAddress != "" {
		go serveMetrics()
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

	chaoskube.Run(ctx, maxJitter, ticker.C)
}

func newClient() (*kubernetes.Clientset, error) {
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

func createNotifier() notifier.Notifier {
	notifiers := notifier.New()
	if slackWebhook != "" {
		notifiers.Add(notifier.NewSlackNotifier(slackWebhook))
	}

	return notifiers
}

func serveMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "OK")
	})
	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, adminPage)
	})
	if err := http.ListenAndServe(metricsAddress, nil); err != nil {
		log.WithField("err", err).Fatal("failed to start HTTP server")
	}
}

func prettifyCaller(f *runtime.Frame) (string, string) {
	_, filename := path.Split(f.File)
	return "", fmt.Sprintf("%s:%d", filename, f.Line)
}

var adminPage = `<html>
	<head>
		<title>chaoskube</title>
	</head>
	<body>
		<h1>chaoskube</h1>
		<p><a href="/metrics">Metrics</a></p>
		<p><a href="/healthz">Health Check</a></p>
		<p><a href="/debug/pprof">pprof</a></p>
	</body>
</html>`

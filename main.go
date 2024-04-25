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

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
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

const envVarPrefix = "CHAOSKUBE_"

var version = "undefined"

var (
	labelString          string
	annString            string
	kindsString          string
	nsString             string
	nsLabelString        string
	includedPodNames     *regexp.Regexp
	excludedPodNames     *regexp.Regexp
	excludedWeekdays     string
	excludedTimesOfDay   string
	excludedDaysOfYear   string
	timezone             string
	minimumAge           time.Duration
	maxRuntime           time.Duration
	maxKill              int
	master               string
	kubeconfig           string
	interval             time.Duration
	dryRun               bool
	debug                bool
	metricsAddress       string
	gracePeriod          time.Duration
	logFormat            string
	logCaller            bool
	slackWebhook         string
	clientNamespaceScope string
	ignoreSingleReplicas bool
)

func cliEnvVar(name string) string {
	return envVarPrefix + name
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	klog.SetOutput(ioutil.Discard)

	kingpin.Flag("labels", "A set of labels to restrict the list of affected pods. Defaults to everything.").Envar(cliEnvVar("LABELS")).StringVar(&labelString)
	kingpin.Flag("annotations", "A set of annotations to restrict the list of affected pods. Defaults to everything.").Envar(cliEnvVar("ANNOTATIONS")).StringVar(&annString)
	kingpin.Flag("kinds", "A set of kinds to restrict the list of affected pods. Defaults to everything.").Envar(cliEnvVar("KINDS")).StringVar(&kindsString)
	kingpin.Flag("namespaces", "A set of namespaces to restrict the list of affected pods. Defaults to everything.").Envar(cliEnvVar("NAMESPACES")).StringVar(&nsString)
	kingpin.Flag("namespace-labels", "A set of labels to restrict the list of affected namespaces. Defaults to everything.").Envar(cliEnvVar("NAMESPACE_LABELS")).StringVar(&nsLabelString)
	kingpin.Flag("included-pod-names", "Regular expression that defines which pods to include. All included by default.").Envar(cliEnvVar("INCLUDED_POD_NAMES")).RegexpVar(&includedPodNames)
	kingpin.Flag("excluded-pod-names", "Regular expression that defines which pods to exclude. None excluded by default.").Envar(cliEnvVar("EXCLUDED_POD_NAMES")).RegexpVar(&excludedPodNames)
	kingpin.Flag("excluded-weekdays", "A list of weekdays when termination is suspended, e.g. Sat,Sun").Envar(cliEnvVar("EXCLUDED_WEEKDAYS")).StringVar(&excludedWeekdays)
	kingpin.Flag("excluded-times-of-day", "A list of time periods of a day when termination is suspended, e.g. 22:00-08:00").Envar(cliEnvVar("EXCLUDED_TIMES_OF_DAY")).StringVar(&excludedTimesOfDay)
	kingpin.Flag("excluded-days-of-year", "A list of days of a year when termination is suspended, e.g. Apr1,Dec24").Envar(cliEnvVar("EXCLUDED_DAYS_OF_YEAR")).StringVar(&excludedDaysOfYear)
	kingpin.Flag("timezone", "The timezone by which to interpret the excluded weekdays and times of day, e.g. UTC, Local, Europe/Berlin. Defaults to UTC.").Envar(cliEnvVar("TIMEZONE")).Default("UTC").StringVar(&timezone)
	kingpin.Flag("minimum-age", "Minimum age of pods to consider for termination").Envar(cliEnvVar("MINIMUM_AGE")).Default("0s").DurationVar(&minimumAge)
	kingpin.Flag("max-runtime", "Maximum runtime before chaoskube exits").Envar(cliEnvVar("MAX_RUNTIME")).Default("-1s").DurationVar(&maxRuntime)
	kingpin.Flag("max-kill", "Specifies the maximum number of pods to be terminated per interval.").Envar(cliEnvVar("MAX_KILL")).Default("1").IntVar(&maxKill)
	kingpin.Flag("master", "The address of the Kubernetes cluster to target").Envar(cliEnvVar("MASTER")).StringVar(&master)
	kingpin.Flag("kubeconfig", "Path to a kubeconfig file").Envar(cliEnvVar("KUBECONFIG")).StringVar(&kubeconfig)
	kingpin.Flag("interval", "Interval between Pod terminations").Envar(cliEnvVar("INTERVAL")).Default("10m").DurationVar(&interval)
	kingpin.Flag("dry-run", "Don't actually kill any pod. Turned on by default. Turn off with `--no-dry-run`.").Envar(cliEnvVar("DRY_RUN")).Default("true").BoolVar(&dryRun)
	kingpin.Flag("debug", "Enable debug logging.").Envar(cliEnvVar("DEBUG")).BoolVar(&debug)
	kingpin.Flag("metrics-address", "Listening address for metrics handler").Envar(cliEnvVar("METRICS_ADDRESS")).Default(":8080").StringVar(&metricsAddress)
	kingpin.Flag("grace-period", "Grace period to terminate Pods. Negative values will use the Pod's grace period.").Envar(cliEnvVar("GRACE_PERIOD")).Default("-1s").DurationVar(&gracePeriod)
	kingpin.Flag("log-format", "Specify the format of the log messages. Options are text and json. Defaults to text.").Envar(cliEnvVar("LOG_FORMAT")).Default("text").EnumVar(&logFormat, "text", "json")
	kingpin.Flag("log-caller", "Include the calling function name and location in the log messages.").Envar(cliEnvVar("LOG_CALLER")).BoolVar(&logCaller)
	kingpin.Flag("slack-webhook", "The address of the slack webhook for notifications").Envar(cliEnvVar("SLACK_WEBHOOK")).StringVar(&slackWebhook)
	kingpin.Flag("client-namespace-scope", "Scope Kubernetes API calls to the given namespace. Defaults to v1.NamespaceAll which requires global read permission.").Envar(cliEnvVar("CLIENT_NAMESPACE_SCOPE")).Default(v1.NamespaceAll).StringVar(&clientNamespaceScope)
	kingpin.Flag("ignore-single-replicas", "Ignore services with one replica when needed.").Envar(cliEnvVar("IGNORE_SINGLE_REPLICAS")).BoolVar(&ignoreSingleReplicas)
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
		"labels":               labelString,
		"annotations":          annString,
		"kinds":                kindsString,
		"namespaces":           nsString,
		"namespaceLabels":      nsLabelString,
		"includedPodNames":     includedPodNames,
		"excludedPodNames":     excludedPodNames,
		"excludedWeekdays":     excludedWeekdays,
		"excludedTimesOfDay":   excludedTimesOfDay,
		"excludedDaysOfYear":   excludedDaysOfYear,
		"timezone":             timezone,
		"minimumAge":           minimumAge,
		"maxRuntime":           maxRuntime,
		"maxKill":              maxKill,
		"master":               master,
		"kubeconfig":           kubeconfig,
		"interval":             interval,
		"dryRun":               dryRun,
		"debug":                debug,
		"metricsAddress":       metricsAddress,
		"gracePeriod":          gracePeriod,
		"logFormat":            logFormat,
		"slackWebhook":         slackWebhook,
		"clientNamespaceScope": clientNamespaceScope,
		"ignoreSingleReplicas": ignoreSingleReplicas,
	}).Debug("reading config")

	log.WithFields(log.Fields{
		"version":    version,
		"dryRun":     dryRun,
		"interval":   interval,
		"maxRuntime": maxRuntime,
	}).Info("starting up")

	client, err := newClient()
	if err != nil {
		log.WithField("err", err).Fatal("failed to connect to cluster")
	}

	var (
		labelSelector   = parseSelector(labelString)
		annotations     = parseSelector(annString)
		kinds           = parseSelector(kindsString)
		namespaces      = parseSelector(nsString)
		namespaceLabels = parseSelector(nsLabelString)
	)

	log.WithFields(log.Fields{
		"labels":           labelSelector,
		"annotations":      annotations,
		"kinds":            kinds,
		"namespaces":       namespaces,
		"namespaceLabels":  namespaceLabels,
		"includedPodNames": includedPodNames,
		"excludedPodNames": excludedPodNames,
		"minimumAge":       minimumAge,
		"maxKill":          maxKill,
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
		kinds,
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
		clientNamespaceScope,
		ignoreSingleReplicas,
	)

	if metricsAddress != "" {
		go serveMetrics()
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	if maxRuntime > -1 {
		ctx, cancel = context.WithTimeout(ctx, maxRuntime)
	}

	defer cancel()

	go func() {
		<-done
		cancel()
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	chaoskube.Run(ctx, ticker.C)
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

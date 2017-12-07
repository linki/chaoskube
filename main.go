package main

import (
	"fmt"
	"os"
	"strconv"
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

const (
	limitChaosOpt = "limit-chaos"
	locationOpt   = "location"
	offDaysOpt    = "off-days"
	chaosHrsOpt   = "chaos-hours"
	holidaysOpt   = "holidays"

	defaultStartHr  = 9
	defaultStartMin = 30
	defaultEndHr    = 16
	defaultEndMin   = 30

	iso8601 = "2006-01-02"
)

var (
	labelString    string
	annString      string
	nsString       string
	master         string
	kubeconfig     string
	interval       time.Duration
	inCluster      bool
	dryRun         bool
	debug          bool
	version        string
	limitChaos     bool
	locationString string
	offDaysString  string
	chaosHrsString string
	holidaysString string
)

// offtimeCfg holds configuration information related to when to suspend the chaos.
type offtimeCfg struct {
	// Whether chaos limiting is enabled
	enabled bool
	// timezone in which the worktimes are expressed
	location *time.Location
	// Days on which chaos is suspended
	offDays []time.Weekday
	// Chaos start and end hours and minutes
	chaosStartHr  int
	chaosStartMin int
	chaosEndHr    int
	chaosEndMin   int
	// holidays, assumed to be expressed in UTC, regardless of Location
	holidays []time.Time
}

func init() {
	kingpin.Flag("labels", "A set of labels to restrict the list of affected pods. Defaults to everything.").StringVar(&labelString)
	kingpin.Flag("annotations", "A set of annotations to restrict the list of affected pods. Defaults to everything.").StringVar(&annString)
	kingpin.Flag("namespaces", "A set of namespaces to restrict the list of affected pods. Defaults to everything.").StringVar(&nsString)
	kingpin.Flag("master", "The address of the Kubernetes cluster to target").StringVar(&master)
	kingpin.Flag("kubeconfig", "Path to a kubeconfig file").StringVar(&kubeconfig)
	kingpin.Flag("interval", "Interval between Pod terminations").Default("10m").DurationVar(&interval)
	kingpin.Flag("dry-run", "If true, don't actually do anything.").Default("true").BoolVar(&dryRun)
	kingpin.Flag("debug", "Enable debug logging.").BoolVar(&debug)
	kingpin.Flag(limitChaosOpt, "Whether to limit chaos according to configuration. Defaults to false.").Default("false").BoolVar(&limitChaos)
	kingpin.Flag(locationOpt, `Timezone location from the "tz database" (e.g. "America/Los_Angeles", not "PDT") `+
		`for interpreting chaos-period start and stop times. No default.`).StringVar(&locationString)
	help := fmt.Sprintf(`Daily start and end times for introducing chaos. Defaults to "start: %d:%d, end: %d:%d".`,
		defaultStartHr, defaultStartMin, defaultEndHr, defaultEndMin)
	kingpin.Flag(chaosHrsOpt, help).StringVar(&chaosHrsString)
	kingpin.Flag(offDaysOpt, `A list of days of the week when chaos is suspended. Defaults to "Saturday, Sunday". (Use "none" for no off days.)`).StringVar(&offDaysString)
	kingpin.Flag(holidaysOpt, `A list of ISO 8601 dates (YYYY-MM-DD) when chaos is suspended. Defaults to and empty list.`).StringVar(&holidaysString)
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

	offcfg, err := handleOfftimeConfig(limitChaos, locationString, offDaysString, chaosHrsString, holidaysString)
	if err != nil {
		log.Fatal(err)
	}
	if offcfg.enabled {
		log.Infof("Limiting chaos. %s: %s, %s: %s, %s: %s, %s: %s",
			locationOpt, locationString,
			offDaysOpt, offDaysString,
			chaosHrsOpt, chaosHrsString,
			holidaysOpt, holidaysString)
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

	for {
		if timeToSuspend(time.Now(), *offcfg) {
			log.Debugf("Chaos currently suspended")
		} else {
			if err := chaoskube.TerminateVictim(); err != nil {
				log.Fatal(err)
			}
		}

		log.Debugf("Sleeping for %s...", interval)
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

	log.Infof("Targeting cluster at %s", config.Host)

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func setLocation(offcfg *offtimeCfg, locationStr string) error {
	var err error
	if len(locationStr) == 0 {
		err = fmt.Errorf("timezone location is required if %s is enabled", limitChaosOpt)
		return err
	}
	offcfg.location, err = time.LoadLocation(locationStr)
	if err != nil {
		err = fmt.Errorf(err.Error()+`- %s must one of: a timezone from the "tz database" (IANA), "UTC" or "Local"`, locationOpt)
		return err
	}
	return err
}

func setOffDays(offcfg *offtimeCfg, offDaysStr string) error {
	var err error
	offcfg.offDays = make([]time.Weekday, 0, 2)
	if offDaysStr == "none" {
		return err
	} else if len(offDaysStr) == 0 {
		offcfg.offDays = append(offcfg.offDays, time.Saturday, time.Sunday)
	} else {
		days := strings.Split(offDaysStr, ",")
		for _, day := range days {
			switch strings.TrimSpace(day) {
			case time.Sunday.String():
				offcfg.offDays = append(offcfg.offDays, time.Sunday)
			case time.Monday.String():
				offcfg.offDays = append(offcfg.offDays, time.Monday)
			case time.Tuesday.String():
				offcfg.offDays = append(offcfg.offDays, time.Tuesday)
			case time.Wednesday.String():
				offcfg.offDays = append(offcfg.offDays, time.Wednesday)
			case time.Thursday.String():
				offcfg.offDays = append(offcfg.offDays, time.Thursday)
			case time.Friday.String():
				offcfg.offDays = append(offcfg.offDays, time.Friday)
			case time.Saturday.String():
				offcfg.offDays = append(offcfg.offDays, time.Saturday)
			default:
				err = fmt.Errorf("unrecognized day of week in %s: %s", offDaysOpt, day)
				return err
			}
		}
	}
	return err
}

func setChaosHours(offcfg *offtimeCfg, chaosHrsStr string) error {
	var err error
	if len(chaosHrsStr) == 0 {
		offcfg.chaosStartHr = defaultStartHr
		offcfg.chaosStartMin = defaultStartMin
		offcfg.chaosEndHr = defaultEndHr
		offcfg.chaosEndMin = defaultEndMin
	} else {
		startEnd := strings.Split(chaosHrsStr, ",")
		for _, item := range startEnd {
			switch kv := strings.SplitN(strings.TrimSpace(item), ":", 2); kv[0] {
			case "start":
				offcfg.chaosStartHr, offcfg.chaosStartMin, err = getHrMin(kv[1])
				if err != nil {
					err = fmt.Errorf(`in %s, could not parse "%s"`, chaosHrsOpt, item)
					return err
				}
			case "end":
				offcfg.chaosEndHr, offcfg.chaosEndMin, err = getHrMin(kv[1])
				if err != nil {
					err = fmt.Errorf(`in %s, could not parse "%s"`, chaosHrsOpt, item)
					return err
				}
			default:
				err = fmt.Errorf(`%s requires this format: "start: 9:30, end: 17:30". (Got key: "%s")`, chaosHrsOpt, kv[0])
				return err
			}
		}
	}
	// Validate
	v1 := offcfg.chaosStartHr*10 + offcfg.chaosStartMin
	v2 := offcfg.chaosEndHr*10 + offcfg.chaosEndMin
	if v1 > v2 {
		err = fmt.Errorf("%s may not specify a period that spans midnight, and must be expressed in 24hr time", chaosHrsOpt)
	}
	return err
}

// getHrmMin parses out the hr and min from " hr:min"
func getHrMin(hrmMinStr string) (hr, min int, err error) {
	hm := strings.Split(strings.TrimSpace(hrmMinStr), ":")
	hr, err = strconv.Atoi(hm[0])
	if err != nil {
		return hr, min, err
	}
	min, err = strconv.Atoi(hm[1])
	if err != nil {
		return hr, min, err
	}
	return hr, min, err
}

func setHolidays(offcfg *offtimeCfg, holidaysStr string) error {
	var err error
	if len(holidaysStr) == 0 {
		// Leave Holidays nil
		return err
	}
	offcfg.holidays = make([]time.Time, 0)
	for _, hStr := range strings.Split(holidaysStr, ",") {
		layout := iso8601
		var holiday time.Time
		holiday, err = time.ParseInLocation(layout, strings.TrimSpace(hStr), offcfg.location)
		if err != nil {
			err = fmt.Errorf(`in %s, invalid date format. "YYYY-MM-DD" required. (Got "%s")`, holidaysOpt, hStr)
			return err
		}
		offcfg.holidays = append(offcfg.holidays, holiday)
	}
	return err
}

func handleOfftimeConfig(enabled bool, locationStr, offDaysStr, chaosHrsStr, holidaysStr string) (*offtimeCfg, error) {
	var err error
	offcfg := &offtimeCfg{}

	offcfg.enabled = enabled
	if !enabled {
		// Not enabled, no need to set other values
		return offcfg, err
	}

	if err = setLocation(offcfg, locationStr); err != nil {
		return offcfg, err
	}

	if err = setOffDays(offcfg, offDaysStr); err != nil {
		return offcfg, err
	}

	if err = setChaosHours(offcfg, chaosHrsStr); err != nil {
		return offcfg, err
	}

	if err = setHolidays(offcfg, holidaysStr); err != nil {
		return offcfg, err
	}

	return offcfg, err
}

// timeToSuspend examines the supplied time and offtimeCfg and determines whether it is time to suspend chaos.
func timeToSuspend(currTime time.Time, offcfg offtimeCfg) bool {
	if !offcfg.enabled {
		// If limiting not enabled, it's never time to suspend
		return false
	}

	// Localize the currTime
	locTime := currTime.In(offcfg.location)

	// Check offDays
	currDay := locTime.Weekday()
	for _, od := range offcfg.offDays {
		if currDay == od {
			return true
		}
	}

	// Check holidays
	ty, tm, td := locTime.Date()
	for _, holiday := range offcfg.holidays {
		hy, hm, hd := holiday.Date()
		if ty == hy && tm == hm && td == hd {
			return true
		}
	}

	// Check time of day. Start by getting today's chaos start/end times
	chaosStart := time.Date(ty, tm, td, offcfg.chaosStartHr, offcfg.chaosStartMin, 0, 0, offcfg.location)
	chaosEnd := time.Date(ty, tm, td, offcfg.chaosEndHr, offcfg.chaosEndMin, 0, 0, offcfg.location)
	if !((chaosStart.Before(locTime) || chaosStart.Equal(locTime)) && locTime.Before(chaosEnd)) {
		return true
	}

	return false
}

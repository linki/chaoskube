package chaoskube

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"

	multierror "github.com/hashicorp/go-multierror"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"

	"github.com/linki/chaoskube/metrics"
	"github.com/linki/chaoskube/notifier"
	"github.com/linki/chaoskube/terminator"
	"github.com/linki/chaoskube/util"
)

// Chaoskube represents an instance of chaoskube
type Chaoskube struct {
	// a kubernetes client object
	Client kubernetes.Interface
	// a label selector which restricts the pods to choose from
	Labels labels.Selector
	// an annotation selector which restricts the pods to choose from
	Annotations labels.Selector
	// a kind label selector which restricts the kinds to choose from
	Kinds labels.Selector
	// a namespace selector which restricts the pods to choose from
	Namespaces labels.Selector
	// a namespace label selector which restricts the namespaces to choose from
	NamespaceLabels labels.Selector
	// a regular expression for pod names to include
	IncludedPodNames *regexp.Regexp
	// a regular expression for pod names to exclude
	ExcludedPodNames *regexp.Regexp
	// a list of weekdays when termination is suspended
	ExcludedWeekdays []time.Weekday
	// a list of time periods of a day when termination is suspended
	ExcludedTimesOfDay []util.TimePeriod
	// a list of days of a year when termination is suspended
	ExcludedDaysOfYear []time.Time
	// the timezone to apply when detecting the current weekday
	Timezone *time.Location
	// minimum age of pods to consider
	MinimumAge time.Duration
	// an instance of logrus.StdLogger to write log messages to
	Logger log.FieldLogger
	// a terminator that terminates victim pods
	Terminator terminator.Terminator
	// dry run will not allow any pod terminations
	DryRun bool
	// grace period to terminate the pods
	GracePeriod time.Duration
	// event recorder allows to publish events to Kubernetes
	EventRecorder record.EventRecorder
	// a function to retrieve the current time
	Now func() time.Time

	MaxKill int
	// chaos events notifier
	Notifier notifier.Notifier
	// namespace scope for the Kubernetes client
	ClientNamespaceScope string

	// Dynamic interval configuration
	DynamicInterval       bool
	DynamicIntervalFactor float64
	BaseInterval          time.Duration
}

var (
	// errPodNotFound is returned when no victim could be found
	errPodNotFound = errors.New("pod not found")
	// msgVictimNotFound is the log message when no victim was found
	msgVictimNotFound = "no victim found"
	// msgWeekdayExcluded is the log message when termination is suspended due to the weekday filter
	msgWeekdayExcluded = "weekday excluded"
	// msgTimeOfDayExcluded is the log message when termination is suspended due to the time of day filter
	msgTimeOfDayExcluded = "time of day excluded"
	// msgDayOfYearExcluded is the log message when termination is suspended due to the day of year filter
	msgDayOfYearExcluded = "day of year excluded"
)

// New returns a new instance of Chaoskube. It expects:
// * a Kubernetes client to connect to a Kubernetes API
// * label, annotation and/or namespace selectors to reduce the amount of possible target pods
// * a list of weekdays, times of day and/or days of a year when chaos mode is disabled
// * a time zone to apply to the aforementioned time-based filters
// * a logger implementing logrus.FieldLogger to send log output to
// * what specific terminator to use to imbue chaos on victim pods
// * whether to enable/disable dry-run mode
func New(client kubernetes.Interface, labels, annotations, kinds, namespaces, namespaceLabels labels.Selector, includedPodNames, excludedPodNames *regexp.Regexp, excludedWeekdays []time.Weekday, excludedTimesOfDay []util.TimePeriod, excludedDaysOfYear []time.Time, timezone *time.Location, minimumAge time.Duration, logger log.FieldLogger, dryRun bool, terminator terminator.Terminator, maxKill int, notifier notifier.Notifier, clientNamespaceScope string, dynamicInterval bool, dynamicIntervalFactor float64, baseInterval time.Duration) *Chaoskube {
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events(clientNamespaceScope)})
	recorder := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "chaoskube"})

	return &Chaoskube{
		Client:                client,
		Labels:                labels,
		Annotations:           annotations,
		Kinds:                 kinds,
		Namespaces:            namespaces,
		NamespaceLabels:       namespaceLabels,
		IncludedPodNames:      includedPodNames,
		ExcludedPodNames:      excludedPodNames,
		ExcludedWeekdays:      excludedWeekdays,
		ExcludedTimesOfDay:    excludedTimesOfDay,
		ExcludedDaysOfYear:    excludedDaysOfYear,
		Timezone:              timezone,
		MinimumAge:            minimumAge,
		Logger:                logger,
		DryRun:                dryRun,
		Terminator:            terminator,
		EventRecorder:         recorder,
		Now:                   time.Now,
		MaxKill:               maxKill,
		Notifier:              notifier,
		ClientNamespaceScope:  clientNamespaceScope,
		DynamicInterval:       dynamicInterval,
		DynamicIntervalFactor: dynamicIntervalFactor,
		BaseInterval:          baseInterval,
	}
}

// CalculateDynamicInterval calculates a dynamic interval based on current pod count
func (c *Chaoskube) CalculateDynamicInterval(ctx context.Context) time.Duration {
	// If dynamic interval is disabled, return the base interval
	if !c.DynamicInterval {
		return c.BaseInterval
	}

	// Get candidate pods count
	pods, err := c.Candidates(ctx)
	if err != nil {
		c.Logger.WithField("err", err).Error("failed to get candidates, using base interval")
		return c.BaseInterval
	}

	podCount := len(pods)

	// Add debug logging for pod details
	if c.Logger.(*log.Entry).Logger.Level >= log.DebugLevel {
		c.Logger.Debug("Listing candidate pods for dynamic interval calculation:")
		for i, pod := range pods {
			c.Logger.WithFields(log.Fields{
				"index":     i,
				"name":      pod.Name,
				"namespace": pod.Namespace,
				"labels":    pod.Labels,
				"phase":     pod.Status.Phase,
			}).Debug("candidate pod")
		}
	}

	// Guard against division by zero
	if podCount == 0 {
		c.Logger.WithField("podCount", 0).Info("no pods found, using base interval")
		return c.BaseInterval
	}
	// As a simple reference, we asume that every pod should be killed during 10 working days (9-17h)
	totalWorkingMinutes := 10 * 8 * 60

	// Calculate raw interval in minutes
	// Higher pod counts = shorter intervals, lower pod counts = longer intervals
	rawIntervalMinutes := float64(totalWorkingMinutes) / (float64(podCount) * c.DynamicIntervalFactor)

	// Round to nearest minute and ensure minimum of 1 minute
	minutes := int(math.Max(1, math.Round(rawIntervalMinutes)))
	roundedInterval := time.Duration(minutes) * time.Minute

	// Provide detailed logging about the calculation
	c.Logger.WithFields(log.Fields{
		"podCount":         podCount,
		"totalWorkMinutes": totalWorkingMinutes,
		"factor":           c.DynamicIntervalFactor,
		"rawIntervalMins":  rawIntervalMinutes,
		"roundedInterval":  roundedInterval,
	}).Info("calculated dynamic interval")

	return roundedInterval
}

// Run continuously picks and terminates a victim pod at a given interval
// described by channel next. It returns when the given context is canceled.
func (c *Chaoskube) Run(ctx context.Context, next <-chan time.Time) {
	for {
		// If dynamic interval is enabled, calculate new interval before terminating victims
		var waitDuration time.Duration
		if c.DynamicInterval {
			waitDuration = c.CalculateDynamicInterval(ctx)
			metrics.CurrentIntervalSeconds.Set(float64(waitDuration.Seconds()))
		}

		if err := c.TerminateVictims(ctx); err != nil {
			c.Logger.WithField("err", err).Error("failed to terminate victim")
			metrics.ErrorsTotal.Inc()
		}

		c.Logger.Debug("sleeping...")
		metrics.IntervalsTotal.Inc()

		// Use the appropriate waiting mechanism
		if c.DynamicInterval {
			select {
			case <-time.After(waitDuration):
				// Continue to next iteration
			case <-ctx.Done():
				return
			}
		} else {
			// Use original fixed interval from ticker
			select {
			case <-next:
			case <-ctx.Done():
				return
			}
		}
	}
}

// TerminateVictims picks and deletes a victim.
// It respects the configured excluded weekdays, times of day and days of a year filters.
func (c *Chaoskube) TerminateVictims(ctx context.Context) error {
	now := c.Now().In(c.Timezone)

	for _, wd := range c.ExcludedWeekdays {
		if wd == now.Weekday() {
			c.Logger.WithField("weekday", now.Weekday()).Debug(msgWeekdayExcluded)
			return nil
		}
	}

	for _, tp := range c.ExcludedTimesOfDay {
		if tp.Includes(now) {
			c.Logger.WithField("timeOfDay", now.Format(util.Kitchen24)).Debug(msgTimeOfDayExcluded)
			return nil
		}
	}

	for _, d := range c.ExcludedDaysOfYear {
		if d.Day() == now.Day() && d.Month() == now.Month() {
			c.Logger.WithField("dayOfYear", now.Format(util.YearDay)).Debug(msgDayOfYearExcluded)
			return nil
		}
	}

	victims, err := c.Victims(ctx)
	if err == errPodNotFound {
		c.Logger.Debug(msgVictimNotFound)
		return nil
	}
	if err != nil {
		return err
	}

	var result *multierror.Error
	for _, victim := range victims {
		err = c.DeletePod(ctx, victim)
		result = multierror.Append(result, err)
	}

	return result.ErrorOrNil()
}

// Victims returns up to N pods as configured by MaxKill flag
func (c *Chaoskube) Victims(ctx context.Context) ([]v1.Pod, error) {
	pods, err := c.Candidates(ctx)
	if err != nil {
		return []v1.Pod{}, err
	}

	c.Logger.WithField("count", len(pods)).Debug("found candidates")

	if len(pods) == 0 {
		return []v1.Pod{}, errPodNotFound
	}

	pods = util.RandomPodSubSlice(pods, c.MaxKill)

	c.Logger.WithField("count", len(pods)).Debug("found victims")
	return pods, nil
}

// Candidates returns the list of pods that are available for termination.
// It returns all pods that match the configured label, annotation and namespace selectors.
func (c *Chaoskube) Candidates(ctx context.Context) ([]v1.Pod, error) {
	listOptions := metav1.ListOptions{LabelSelector: c.Labels.String()}

	podList, err := c.Client.CoreV1().Pods(c.ClientNamespaceScope).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	pods, err := filterByNamespaces(podList.Items, c.Namespaces)
	if err != nil {
		return nil, err
	}

	pods, err = filterPodsByNamespaceLabels(ctx, pods, c.NamespaceLabels, c.Client)
	if err != nil {
		return nil, err
	}

	pods, err = filterByKinds(pods, c.Kinds)
	if err != nil {
		return nil, err
	}

	pods = filterByAnnotations(pods, c.Annotations)
	pods = filterByPhase(pods, v1.PodRunning)
	pods = filterTerminatingPods(pods)
	pods = filterByMinimumAge(pods, c.MinimumAge, c.Now())
	pods = filterByPodName(pods, c.IncludedPodNames, c.ExcludedPodNames)
	pods = filterByOwnerReference(pods)

	return pods, nil
}

// DeletePod deletes the given pod with the selected terminator.
// It will not delete the pod if dry-run mode is enabled.
func (c *Chaoskube) DeletePod(ctx context.Context, victim v1.Pod) error {
	c.Logger.WithFields(log.Fields{
		"namespace": victim.Namespace,
		"name":      victim.Name,
	}).Info("terminating pod")

	// return early if we're running in dryRun mode.
	if c.DryRun {
		return nil
	}

	start := time.Now()
	err := c.Terminator.Terminate(ctx, victim)
	metrics.TerminationDurationSeconds.Observe(time.Since(start).Seconds())
	if err != nil {
		return err
	}

	metrics.PodsDeletedTotal.WithLabelValues(victim.Namespace).Inc()

	ref, err := reference.GetReference(scheme.Scheme, &victim)
	if err != nil {
		return err
	}

	c.EventRecorder.Event(ref, v1.EventTypeNormal, "Killing", "Pod was terminated by chaoskube to introduce chaos.")

	if err := c.Notifier.NotifyPodTermination(victim); err != nil {
		c.Logger.WithField("err", err).Warn("failed to notify pod termination")
	}

	return nil
}

// filterByKinds filters a list of pods by a given kind selector.
func filterByKinds(pods []v1.Pod, kinds labels.Selector) ([]v1.Pod, error) {
	// empty filter returns original list
	if kinds.Empty() {
		return pods, nil
	}

	// split requirements into including and excluding groups
	reqs, _ := kinds.Requirements()
	reqIncl := []labels.Requirement{}
	reqExcl := []labels.Requirement{}

	for _, req := range reqs {
		switch req.Operator() {
		case selection.Exists:
			reqIncl = append(reqIncl, req)
		case selection.DoesNotExist:
			reqExcl = append(reqExcl, req)
		default:
			return nil, fmt.Errorf("unsupported operator: %s", req.Operator())
		}
	}

	filteredList := []v1.Pod{}

	for _, pod := range pods {
		// if there aren't any including requirements, we're in by default
		included := len(reqIncl) == 0

		// Check owner reference
		for _, ref := range pod.GetOwnerReferences() {
			// convert the pod's owner kind to an equivalent label selector
			selector := labels.Set{ref.Kind: ""}

			// include pod if one including requirement matches
			for _, req := range reqIncl {
				if req.Matches(selector) {
					included = true
					break
				}
			}

			// exclude pod if it is filtered out by at least one excluding requirement
			for _, req := range reqExcl {
				if !req.Matches(selector) {
					included = false
					break
				}
			}
		}

		if included {
			filteredList = append(filteredList, pod)
		}
	}

	return filteredList, nil
}

// filterByNamespaces filters a list of pods by a given namespace selector.
func filterByNamespaces(pods []v1.Pod, namespaces labels.Selector) ([]v1.Pod, error) {
	// empty filter returns original list
	if namespaces.Empty() {
		return pods, nil
	}

	// split requirements into including and excluding groups
	reqs, _ := namespaces.Requirements()
	reqIncl := []labels.Requirement{}
	reqExcl := []labels.Requirement{}

	for _, req := range reqs {
		switch req.Operator() {
		case selection.Exists:
			reqIncl = append(reqIncl, req)
		case selection.DoesNotExist:
			reqExcl = append(reqExcl, req)
		default:
			return nil, fmt.Errorf("unsupported operator: %s", req.Operator())
		}
	}

	filteredList := []v1.Pod{}

	for _, pod := range pods {
		// if there aren't any including requirements, we're in by default
		included := len(reqIncl) == 0

		// convert the pod's namespace to an equivalent label selector
		selector := labels.Set{pod.Namespace: ""}

		// include pod if one including requirement matches
		for _, req := range reqIncl {
			if req.Matches(selector) {
				included = true
				break
			}
		}

		// exclude pod if it is filtered out by at least one excluding requirement
		for _, req := range reqExcl {
			if !req.Matches(selector) {
				included = false
				break
			}
		}

		if included {
			filteredList = append(filteredList, pod)
		}
	}

	return filteredList, nil
}

// filterPodsByNamespaceLabels filters a list of pods by a given label selector on their namespace.
func filterPodsByNamespaceLabels(ctx context.Context, pods []v1.Pod, labels labels.Selector, client kubernetes.Interface) ([]v1.Pod, error) {
	// empty filter returns original list
	if labels.Empty() {
		return pods, nil
	}

	// find all namespaces matching the label selector
	listOptions := metav1.ListOptions{LabelSelector: labels.String()}

	namespaces, err := client.CoreV1().Namespaces().List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	filteredList := []v1.Pod{}

	for _, pod := range pods {
		for _, namespace := range namespaces.Items {
			// include pod if its in one of the matched namespaces
			if pod.Namespace == namespace.Name {
				filteredList = append(filteredList, pod)
			}
		}
	}

	return filteredList, nil
}

// filterByAnnotations filters a list of pods by a given annotation selector.
func filterByAnnotations(pods []v1.Pod, annotations labels.Selector) []v1.Pod {
	// empty filter returns original list
	if annotations.Empty() {
		return pods
	}

	filteredList := []v1.Pod{}

	for _, pod := range pods {
		// convert the pod's annotations to an equivalent label selector
		selector := labels.Set(pod.Annotations)

		// include pod if its annotations match the selector
		if annotations.Matches(selector) {
			filteredList = append(filteredList, pod)
		}
	}

	return filteredList
}

// filterByPhase filters a list of pods by a given PodPhase, e.g. Running.
func filterByPhase(pods []v1.Pod, phase v1.PodPhase) []v1.Pod {
	filteredList := []v1.Pod{}

	for _, pod := range pods {
		if pod.Status.Phase == phase {
			filteredList = append(filteredList, pod)
		}
	}

	return filteredList
}

// filterTerminatingPods removes pod which have a non nil DeletionTimestamp
func filterTerminatingPods(pods []v1.Pod) []v1.Pod {
	filteredList := []v1.Pod{}
	for _, pod := range pods {
		if pod.DeletionTimestamp != nil {
			continue
		}
		filteredList = append(filteredList, pod)
	}
	return filteredList
}

// filterByMinimumAge filters pods by creation time. Only pods
// older than minimumAge are returned
func filterByMinimumAge(pods []v1.Pod, minimumAge time.Duration, now time.Time) []v1.Pod {
	if minimumAge <= time.Duration(0) {
		return pods
	}

	creationTime := now.Add(-minimumAge)

	filteredList := []v1.Pod{}

	for _, pod := range pods {
		if pod.ObjectMeta.CreationTimestamp.Time.Before(creationTime) {
			filteredList = append(filteredList, pod)
		}
	}

	return filteredList
}

// filterByPodName filters pods by name.  Only pods matching the includedPodNames and not
// matching the excludedPodNames are returned
func filterByPodName(pods []v1.Pod, includedPodNames, excludedPodNames *regexp.Regexp) []v1.Pod {
	// return early if neither included nor excluded regular expressions are given
	if includedPodNames == nil && excludedPodNames == nil {
		return pods
	}

	filteredList := []v1.Pod{}

	for _, pod := range pods {
		include := includedPodNames == nil || includedPodNames.String() == "" || includedPodNames.MatchString(pod.Name)
		exclude := excludedPodNames != nil && excludedPodNames.String() != "" && excludedPodNames.MatchString(pod.Name)

		if include && !exclude {
			filteredList = append(filteredList, pod)
		}
	}

	return filteredList
}

func filterByOwnerReference(pods []v1.Pod) []v1.Pod {
	owners := make(map[types.UID][]v1.Pod)
	filteredList := []v1.Pod{}
	for _, pod := range pods {
		// Don't filter out pods with no owner reference
		if len(pod.GetOwnerReferences()) == 0 {
			filteredList = append(filteredList, pod)
			continue
		}

		// Group remaining pods by their owner reference
		for _, ref := range pod.GetOwnerReferences() {
			owners[ref.UID] = append(owners[ref.UID], pod)
		}
	}

	// For each owner reference select a random pod from its group
	for _, pods := range owners {
		filteredList = append(filteredList, util.RandomPodSubSlice(pods, 1)...)
	}

	return filteredList
}

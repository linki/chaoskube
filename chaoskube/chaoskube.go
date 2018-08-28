package chaoskube

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/linki/chaoskube/metrics"
	"github.com/linki/chaoskube/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes"
)

// Chaoskube represents an instance of chaoskube
type Chaoskube struct {
	// a kubernetes client object
	Client kubernetes.Interface
	// a label selector which restricts the pods to choose from
	Labels labels.Selector
	// an annotation selector which restricts the pods to choose from
	Annotations labels.Selector
	// a namespace selector which restricts the pods to choose from
	Namespaces labels.Selector
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
	// dry run will not allow any pod terminations
	DryRun bool
	// a function to retrieve the current time
	Now func() time.Time
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
// * whether to enable/disable dry-run mode
func New(client kubernetes.Interface, labels, annotations, namespaces labels.Selector, excludedWeekdays []time.Weekday, excludedTimesOfDay []util.TimePeriod, excludedDaysOfYear []time.Time, timezone *time.Location, minimumAge time.Duration, logger log.FieldLogger, dryRun bool) *Chaoskube {
	return &Chaoskube{
		Client:             client,
		Labels:             labels,
		Annotations:        annotations,
		Namespaces:         namespaces,
		ExcludedWeekdays:   excludedWeekdays,
		ExcludedTimesOfDay: excludedTimesOfDay,
		ExcludedDaysOfYear: excludedDaysOfYear,
		Timezone:           timezone,
		MinimumAge:         minimumAge,
		Logger:             logger,
		DryRun:             dryRun,
		Now:                time.Now,
	}
}

// Run continuously picks and terminates a victim pod at a given interval
// described by channel next. It returns when the given context is canceled.
func (c *Chaoskube) Run(ctx context.Context, next <-chan time.Time) {
	for {
		metrics.RunCounter.Inc()
		if err := c.TerminateVictim(); err != nil {
			metrics.ErrorCounter.Inc()
			c.Logger.WithField("err", err).Error("failed to terminate victim")
		} else {
			metrics.PodsDeletedCounter.Inc()
		}

		c.Logger.Debug("sleeping...")
		select {
		case <-next:
		case <-ctx.Done():
			return
		}
	}
}

// TerminateVictim picks and deletes a victim.
// It respects the configured excluded weekdays, times of day and days of a year filters.
func (c *Chaoskube) TerminateVictim() error {
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

	victim, err := c.Victim()
	if err == errPodNotFound {
		c.Logger.Debug(msgVictimNotFound)
		return nil
	}
	if err != nil {
		return err
	}

	return c.DeletePod(victim)
}

// Victim returns a random pod from the list of Candidates.
// It returns an error if there are no candidates to choose from.
func (c *Chaoskube) Victim() (v1.Pod, error) {
	pods, err := c.Candidates()
	if err != nil {
		return v1.Pod{}, err
	}

	c.Logger.WithField("count", len(pods)).Debug("found candidates")

	if len(pods) == 0 {
		return v1.Pod{}, errPodNotFound
	}

	index := rand.Intn(len(pods))

	return pods[index], nil
}

// Candidates returns the list of pods that are available for termination.
// It returns all pods that match the configured label, annotation and namespace selectors.
func (c *Chaoskube) Candidates() ([]v1.Pod, error) {
	listOptions := metav1.ListOptions{LabelSelector: c.Labels.String()}

	podList, err := c.Client.CoreV1().Pods(v1.NamespaceAll).List(listOptions)
	if err != nil {
		return nil, err
	}

	pods, err := filterByNamespaces(podList.Items, c.Namespaces)
	if err != nil {
		return nil, err
	}

	pods = filterByAnnotations(pods, c.Annotations)
	pods = filterByPhase(pods, v1.PodRunning)
	pods = filterByMinimumAge(pods, c.MinimumAge, c.Now())

	return pods, nil
}

// DeletePod deletes the given pod.
// It will not delete the pod if dry-run mode is enabled.
func (c *Chaoskube) DeletePod(victim v1.Pod) error {
	c.Logger.WithFields(log.Fields{
		"namespace": victim.Namespace,
		"name":      victim.Name,
	}).Info("terminating pod")

	if c.DryRun {
		return nil
	}

	return c.Client.CoreV1().Pods(victim.Namespace).Delete(victim.Name, nil)
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

package chaoskube

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"

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
	// a namespace selector which restricts the pods to choose from
	Namespaces labels.Selector
	// a list of weekdays when termination is suspended
	ExcludedWeekdays []time.Weekday
	// a list of time periods of a day when termination is suspended
	ExcludedTimesOfDay []util.TimePeriod
	// the timezone to apply when detecting the current weekday
	Timezone *time.Location
	// an instance of logrus.StdLogger to write log messages to
	Logger log.FieldLogger
	// dry run will not allow any pod terminations
	DryRun bool
	// seed value for the randomizer
	Seed int64
	// a function to retrieve the current time
	Now func() time.Time
}

var (
	// ErrPodNotFound is returned when no victim could be found
	ErrPodNotFound = errors.New("pod not found")
	// msgVictimNotFound is the log message when no victim was found
	msgVictimNotFound = "no victim found"
	// msgWeekdayExcluded is the log message when termination is suspended due to the weekday filter
	msgWeekdayExcluded = "weekday excluded"
	// msgTimeOfDayExcluded is the log message when termination is suspended due to the time of day filter
	msgTimeOfDayExcluded = "time of day excluded"
)

// New returns a new instance of Chaoskube. It expects a kubernetes client, a
// label, annotation and/or namespace selector to reduce the amount of affected
// pods as well as whether to enable dryRun mode and a seed to seed the randomizer
// with. You can also provide a list of weekdays and corresponding time zone when
// chaoskube should be inactive.
func New(client kubernetes.Interface, labels, annotations, namespaces labels.Selector, excludedWeekdays []time.Weekday, excludedTimesOfDay []util.TimePeriod, timezone *time.Location, logger log.FieldLogger, dryRun bool, seed int64) *Chaoskube {
	c := &Chaoskube{
		Client:             client,
		Labels:             labels,
		Annotations:        annotations,
		Namespaces:         namespaces,
		ExcludedWeekdays:   excludedWeekdays,
		ExcludedTimesOfDay: excludedTimesOfDay,
		Timezone:           timezone,
		Logger:             logger,
		DryRun:             dryRun,
		Seed:               seed,
		Now:                time.Now,
	}

	rand.Seed(c.Seed)

	return c
}

// Candidates returns the list of pods that are available for termination.
// It returns all pods matching the label selector and at least one namespace.
func (c *Chaoskube) Candidates() ([]v1.Pod, error) {
	listOptions := metav1.ListOptions{LabelSelector: c.Labels.String()}

	podList, err := c.Client.Core().Pods(v1.NamespaceAll).List(listOptions)
	if err != nil {
		return nil, err
	}

	pods, err := filterByNamespaces(podList.Items, c.Namespaces)
	if err != nil {
		return nil, err
	}

	pods, err = filterByAnnotations(pods, c.Annotations)
	if err != nil {
		return nil, err
	}

	return pods, nil
}

// Victim returns a random pod from the list of Candidates.
func (c *Chaoskube) Victim() (v1.Pod, error) {
	pods, err := c.Candidates()
	if err != nil {
		return v1.Pod{}, err
	}

	c.Logger.WithField("count", len(pods)).Debugf("considering candidates")

	if len(pods) == 0 {
		return v1.Pod{}, ErrPodNotFound
	}

	index := rand.Intn(len(pods))

	return pods[index], nil
}

// DeletePod deletes the passed in pod iff dry run mode is enabled.
func (c *Chaoskube) DeletePod(victim v1.Pod) error {
	c.Logger.WithFields(log.Fields{
		"namespace": victim.Namespace,
		"name":      victim.Name,
		"dryRun":    c.DryRun,
	}).Infof("killing pod")

	if c.DryRun {
		return nil
	}

	return c.Client.Core().Pods(victim.Namespace).Delete(victim.Name, nil)
}

// TerminateVictim picks and deletes a victim if found.
func (c *Chaoskube) TerminateVictim() error {
	now := c.Now().In(c.Timezone)

	for _, wd := range c.ExcludedWeekdays {
		if wd == now.Weekday() {
			c.Logger.WithField("weekday", now.Weekday()).Infof(msgWeekdayExcluded)
			return nil
		}
	}

	for _, tp := range c.ExcludedTimesOfDay {
		if tp.Includes(now) {
			c.Logger.WithField("timeOfDay", now.Format(util.Kitchen24)).Infof(msgTimeOfDayExcluded)
			return nil
		}
	}

	victim, err := c.Victim()
	if err == ErrPodNotFound {
		c.Logger.Infof(msgVictimNotFound)
		return nil
	}
	if err != nil {
		return err
	}

	return c.DeletePod(victim)
}

// filterByNamespaces filters a list of pods by a given namespace selector.
func filterByNamespaces(pods []v1.Pod, namespaces labels.Selector) ([]v1.Pod, error) {
	// empty filter returns original list
	if namespaces.Empty() {
		return pods, nil
	}

	filteredList := []v1.Pod{}

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
			return filteredList, fmt.Errorf("unsupported operator: %s", req.Operator())
		}
	}

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
func filterByAnnotations(pods []v1.Pod, annotations labels.Selector) ([]v1.Pod, error) {
	// empty filter returns original list
	if annotations.Empty() {
		return pods, nil
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

	return filteredList, nil
}

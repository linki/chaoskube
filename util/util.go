package util

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// a short time format; like time.Kitchen but with 24-hour notation.
	Kitchen24 = "15:04"
	// a time format that just cares about the day and month.
	YearDay = "Jan_2"

	DefaultBaseAnnotation = "chaos.alpha.kubernetes.io"
)

// TimePeriod represents a time period with a single beginning and end.
type TimePeriod struct {
	From time.Time
	To   time.Time
}

// NewTimePeriod returns a normalized TimePeriod given a start and end time.
func NewTimePeriod(from, to time.Time) TimePeriod {
	return TimePeriod{From: TimeOfDay(from), To: TimeOfDay(to)}
}

// Includes returns true iff the given pointInTime's time of day is included in time period tp.
func (tp TimePeriod) Includes(pointInTime time.Time) bool {
	isAfter := TimeOfDay(pointInTime).After(tp.From)
	isBefore := TimeOfDay(pointInTime).Before(tp.To)

	if tp.From.Before(tp.To) {
		return isAfter && isBefore
	}
	if tp.From.After(tp.To) {
		return isAfter || isBefore
	}
	return TimeOfDay(pointInTime).Equal(tp.From)
}

// String returns tp as a pretty string.
func (tp TimePeriod) String() string {
	return fmt.Sprintf("%s-%s", tp.From.Format(Kitchen24), tp.To.Format(Kitchen24))
}

// ParseWeekdays takes a comma-separated list of abbreviated weekdays (e.g. sat,sun) and turns them
// into a slice of time.Weekday. It ignores any whitespace and any invalid weekdays.
func ParseWeekdays(weekdays string) []time.Weekday {
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

// ParseTimePeriods takes a comma-separated list of time periods in Kitchen24 format and turns them
// into a slice of TimePeriods. It ignores any whitespace.
func ParseTimePeriods(timePeriods string) ([]TimePeriod, error) {
	parsedTimePeriods := []TimePeriod{}

	for _, tp := range strings.Split(timePeriods, ",") {
		if strings.TrimSpace(tp) == "" {
			continue
		}

		parts := strings.Split(tp, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Invalid time range '%v': must contain exactly one '-'", tp)
		}

		begin, err := time.Parse(Kitchen24, strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, err
		}

		end, err := time.Parse(Kitchen24, strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}

		parsedTimePeriods = append(parsedTimePeriods, NewTimePeriod(begin, end))
	}

	return parsedTimePeriods, nil
}

func ParseDays(days string) ([]time.Time, error) {
	parsedDays := []time.Time{}

	for _, day := range strings.Split(days, ",") {
		if strings.TrimSpace(day) == "" {
			continue
		}

		parsedDay, err := time.Parse(YearDay, strings.TrimSpace(day))
		if err != nil {
			return nil, err
		}

		parsedDays = append(parsedDays, parsedDay)
	}

	return parsedDays, nil
}

// Parses a "frequency" annotation in the form "[number] / [period]" (eg. 1/day)
// and converts it into a chance of occurrence in any given interval (eg. ~0.007)
func ParseFrequency(text string, interval time.Duration) (float64, error) {
	parseablePeriods := map[string]time.Duration{
		"minute": 1 * time.Minute,
		"hour":   1 * time.Hour,
		"day":    24 * time.Hour,
		"week":   24 * 7 * time.Hour,
	}

	parts := strings.SplitN(text, "/", 2)
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}

	frequency, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}

	period, ok := parseablePeriods[parts[1]]
	if !ok {
		return 0, fmt.Errorf("unknown time period, %v", parts[1])
	}

	chance := (float64(interval) / float64(period)) * frequency
	return chance, nil
}

// TimeOfDay normalizes the given point in time by returning a time object that represents the same
// time of day of the given time but on the very first day (day 0).
func TimeOfDay(pointInTime time.Time) time.Time {
	return time.Date(0, 0, 0, pointInTime.Hour(), pointInTime.Minute(), pointInTime.Second(), pointInTime.Nanosecond(), time.UTC)
}

// FormatDays takes a slice of times and returns a slice of strings in YearDate format (e.g. [Apr 1, Sep 24])
func FormatDays(days []time.Time) []string {
	formattedDays := make([]string, 0, len(days))
	for _, d := range days {
		formattedDays = append(formattedDays, d.Format(YearDay))
	}
	return formattedDays
}

// NewNamespace returns a new namespace instance for testing purposes.
func NewNamespace(name string) v1.Namespace {
	return v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"env": name,
			},
		},
	}
}

// RandomPodSubSlice creates a shuffled subslice of the give pods slice
func RandomPodSubSlice(pods []v1.Pod, count int) []v1.Pod {
	maxCount := len(pods)
	if count > maxCount {
		count = maxCount
	}

	rand.Shuffle(len(pods), func(i, j int) { pods[i], pods[j] = pods[j], pods[i] })
	res := pods[0:count]
	return res
}

type PodBuilder struct {
	Name           string
	Namespace      string
	Phase          v1.PodPhase
	OwnerReference *metav1.OwnerReference
	Labels         map[string]string
	Annotations    map[string]string
}

func NewPodBuilder(namespace string, name string) PodBuilder {
	return PodBuilder{
		Name:           name,
		Namespace:      namespace,
		Phase:          v1.PodRunning,
		OwnerReference: nil,
		Annotations:    make(map[string]string),
		Labels:         make(map[string]string),
	}
}

func (b PodBuilder) Build() v1.Pod {
	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   b.Namespace,
			Name:        b.Name,
			Labels:      b.Labels,
			Annotations: b.Annotations,
			SelfLink: fmt.Sprintf(
				"/api/v1/namespaces/%s/pods/%s",
				b.Namespace,
				b.Name,
			),
		},
		Status: v1.PodStatus{
			Phase: b.Phase,
		},
	}

	if b.OwnerReference != nil {
		pod.ObjectMeta.OwnerReferences = []metav1.OwnerReference{*b.OwnerReference}
	}

	return pod
}

func (b PodBuilder) WithPhase(phase v1.PodPhase) PodBuilder {
	b.Phase = phase
	return b
}
func (b PodBuilder) WithOwnerReference(ownerReference metav1.OwnerReference) PodBuilder {
	b.OwnerReference = &ownerReference
	return b
}
func (b PodBuilder) WithOwnerUID(owner types.UID) PodBuilder {
	b.OwnerReference = &metav1.OwnerReference{UID: owner, Kind: "testkind"}
	return b
}
func (b PodBuilder) WithAnnotations(annotations map[string]string) PodBuilder {
	b.Annotations = annotations
	return b
}
func (b PodBuilder) WithLabels(labels map[string]string) PodBuilder {
	b.Labels = labels
	return b
}
func (b PodBuilder) WithFrequency(text string) PodBuilder {
	annotation := strings.Join([]string{DefaultBaseAnnotation, "frequency"}, "/")

	b.Annotations[annotation] = text
	return b
}

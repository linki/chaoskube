package chaoskube

import (
	"bytes"
	"log"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/linki/chaoskube/util"
)

var logOutput = bytes.NewBuffer([]byte{})
var logger = log.New(logOutput, "", 0)

// TestNew tests that arguments are passed to the new instance correctly
func TestNew(t *testing.T) {
	client := fake.NewSimpleClientset()
	labelSelector, _ := labels.Parse("foo=bar")
	annotations, _ := labels.Parse("baz=waldo")
	namespaces, _ := labels.Parse("qux")
	excludedWeekdays := []time.Weekday{time.Friday}
	excludedTimeOfDay := util.NewTimePeriod(ThankGodItsFriday{}.Now(), ThankGodItsFriday{}.Now())

	chaoskube := New(client, labelSelector, annotations, namespaces, excludedWeekdays, []util.TimePeriod{excludedTimeOfDay}, time.UTC, logger, false, 42)

	if chaoskube == nil {
		t.Errorf("expected Chaoskube but got nothing")
	}

	if chaoskube.Client != client {
		t.Errorf("expected %#v, got %#v", client, chaoskube.Client)
	}

	if chaoskube.Labels.String() != "foo=bar" {
		t.Errorf("expected %s, got %s", "foo=bar", chaoskube.Labels.String())
	}

	if chaoskube.Annotations.String() != "baz=waldo" {
		t.Errorf("expected %s, got %s", "baz=waldo", chaoskube.Annotations.String())
	}

	if chaoskube.Namespaces.String() != "qux" {
		t.Errorf("expected %s, got %s", "qux", chaoskube.Namespaces.String())
	}

	if len(chaoskube.ExcludedWeekdays) != 1 {
		t.Fatalf("expected %d, got %d", 1, len(chaoskube.ExcludedWeekdays))
	}

	if chaoskube.ExcludedWeekdays[0] != time.Friday {
		t.Errorf("expected %s, got %s", time.Friday.String(), chaoskube.ExcludedWeekdays[0].String())
	}

	if len(chaoskube.ExcludedTimesOfDay) != 1 {
		t.Fatalf("expected %d, got %d", 1, len(chaoskube.ExcludedTimesOfDay))
	}

	if chaoskube.ExcludedTimesOfDay[0] != excludedTimeOfDay {
		t.Errorf("expected %#v, got %#v", excludedTimeOfDay, chaoskube.ExcludedTimesOfDay[0])
	}

	if chaoskube.Timezone != time.UTC {
		t.Errorf("expected %#v, got %#v", time.UTC, chaoskube.Timezone)
	}

	if chaoskube.Logger != logger {
		t.Errorf("expected %#v, got %#v", logger, chaoskube.Logger)
	}

	if chaoskube.DryRun != false {
		t.Errorf("expected %t, got %t", false, chaoskube.DryRun)
	}

	if chaoskube.Seed != 42 {
		t.Errorf("expected %d, got %d", 42, chaoskube.Seed)
	}
}

func TestCandidates(t *testing.T) {
	foo := map[string]string{"namespace": "default", "name": "foo"}
	bar := map[string]string{"namespace": "testing", "name": "bar"}

	for _, test := range []struct {
		labelSelector      string
		annotationSelector string
		namespaceSelector  string
		pods               []map[string]string
	}{
		{"", "", "", []map[string]string{foo, bar}},
		{"app=foo", "", "", []map[string]string{foo}},
		{"app!=foo", "", "", []map[string]string{bar}},
		{"", "chaos=foo", "", []map[string]string{foo}},
		{"", "chaos!=foo", "", []map[string]string{bar}},
		{"", "", "default", []map[string]string{foo}},
		{"", "", "default,testing", []map[string]string{foo, bar}},
		{"", "", "!testing", []map[string]string{foo}},
		{"", "", "!default,!testing", []map[string]string{}},
		{"", "", "default,!testing", []map[string]string{foo}},
		{"", "", "default,!default", []map[string]string{}},
	} {
		labelSelector, err := labels.Parse(test.labelSelector)
		if err != nil {
			t.Fatal(err)
		}

		annotationSelector, err := labels.Parse(test.annotationSelector)
		if err != nil {
			t.Fatal(err)
		}

		namespaceSelector, err := labels.Parse(test.namespaceSelector)
		if err != nil {
			t.Fatal(err)
		}

		chaoskube := setup(t, labelSelector, annotationSelector, namespaceSelector, []time.Weekday{}, []util.TimePeriod{}, time.UTC, false, 0)

		validateCandidates(t, chaoskube, test.pods)
	}
}

func TestVictim(t *testing.T) {
	foo := map[string]string{"namespace": "default", "name": "foo"}
	bar := map[string]string{"namespace": "testing", "name": "bar"}

	for _, test := range []struct {
		seed          int64
		labelSelector string
		victim        map[string]string
	}{
		{2000, "", foo},
		{4000, "", bar},
		{4000, "app=foo", foo},
	} {
		labelSelector, err := labels.Parse(test.labelSelector)
		if err != nil {
			t.Fatal(err)
		}

		chaoskube := setup(t, labelSelector, labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, false, test.seed)

		validateVictim(t, chaoskube, test.victim)
	}
}

// TestNoVictimReturnsError tests that on missing victim it returns a known error
func TestNoVictimReturnsError(t *testing.T) {
	chaoskube := New(fake.NewSimpleClientset(), labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, logger, false, 0)

	if _, err := chaoskube.Victim(); err != ErrPodNotFound {
		t.Errorf("expected %#v, got %#v", ErrPodNotFound, err)
	}
}

func TestDeletePod(t *testing.T) {
	foo := map[string]string{"namespace": "default", "name": "foo"}
	bar := map[string]string{"namespace": "testing", "name": "bar"}

	for _, test := range []struct {
		dryRun        bool
		remainingPods []map[string]string
	}{
		{false, []map[string]string{bar}},
		{true, []map[string]string{foo, bar}},
	} {
		chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, test.dryRun, 0)

		victim := util.NewPod("default", "foo")

		if err := chaoskube.DeletePod(victim); err != nil {
			t.Fatal(err)
		}

		validateLog(t, "Killing pod default/foo")
		validateCandidates(t, chaoskube, test.remainingPods)
	}
}

func TestTerminateVictim(t *testing.T) {
	midnight := util.NewTimePeriod(ThankGodItsFriday{}.Now().Add(-16*time.Hour), ThankGodItsFriday{}.Now().Add(-14*time.Hour))
	morning := util.NewTimePeriod(ThankGodItsFriday{}.Now().Add(-7*time.Hour), ThankGodItsFriday{}.Now().Add(-6*time.Hour))
	afternoon := util.NewTimePeriod(ThankGodItsFriday{}.Now().Add(-1*time.Hour), ThankGodItsFriday{}.Now().Add(+1*time.Hour))

	australia, err := time.LoadLocation("Australia/Brisbane")
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		excludedWeekdays   []time.Weekday
		excludedTimesOfDay []util.TimePeriod
		now                func() time.Time
		timezone           *time.Location
		remainingPodCount  int
	}{
		// no time is excluded, one pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{},
			ThankGodItsFriday{}.Now,
			time.UTC,
			1,
		},
		// current weekday is excluded, no pod should be killed
		{
			[]time.Weekday{time.Friday},
			[]util.TimePeriod{},
			ThankGodItsFriday{}.Now,
			time.UTC,
			2,
		},
		// current time of day is excluded, no pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{afternoon},
			ThankGodItsFriday{}.Now,
			time.UTC,
			2,
		},
		// one day after an excluded weekday, one pod should be killed
		{
			[]time.Weekday{time.Friday},
			[]util.TimePeriod{},
			func() time.Time { return ThankGodItsFriday{}.Now().Add(24 * time.Hour) },
			time.UTC,
			1,
		},
		// seven days after an excluded weekday, no pod should be killed
		{
			[]time.Weekday{time.Friday},
			[]util.TimePeriod{},
			func() time.Time { return ThankGodItsFriday{}.Now().Add(7 * 24 * time.Hour) },
			time.UTC,
			2,
		},
		// one hour after an excluded time period, one pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{afternoon},
			func() time.Time { return ThankGodItsFriday{}.Now().Add(+2 * time.Hour) },
			time.UTC,
			1,
		},
		// twenty four hours after an excluded time period, no pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{afternoon},
			func() time.Time { return ThankGodItsFriday{}.Now().Add(+24 * time.Hour) },
			time.UTC,
			2,
		},
		// current weekday is excluded but we are in another time zone, one pod should be killed
		{
			[]time.Weekday{time.Friday},
			[]util.TimePeriod{},
			ThankGodItsFriday{}.Now,
			australia,
			1,
		},
		// current time period is excluded but we are in another time zone, one pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{afternoon},
			ThankGodItsFriday{}.Now,
			australia,
			1,
		},
		// one out of two excluded weeksdays match, no pod should be killed
		{
			[]time.Weekday{time.Monday, time.Friday},
			[]util.TimePeriod{},
			ThankGodItsFriday{}.Now,
			time.UTC,
			2,
		},
		// one out of two excluded time periods match, no pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{morning, afternoon},
			ThankGodItsFriday{}.Now,
			time.UTC,
			2,
		},
		// we're inside an excluded time period across days, no pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{midnight},
			func() time.Time { return ThankGodItsFriday{}.Now().Add(-15 * time.Hour) },
			time.UTC,
			2,
		},
		// we're before an excluded time period across days, one pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{midnight},
			func() time.Time { return ThankGodItsFriday{}.Now().Add(-17 * time.Hour) },
			time.UTC,
			1,
		},
		// we're after an excluded time period across days, one pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{midnight},
			func() time.Time { return ThankGodItsFriday{}.Now().Add(-13 * time.Hour) },
			time.UTC,
			1,
		},
	} {
		chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), test.excludedWeekdays, test.excludedTimesOfDay, test.timezone, false, 0)
		chaoskube.Now = test.now

		if err := chaoskube.TerminateVictim(); err != nil {
			t.Fatal(err)
		}

		validateCandidatesCount(t, chaoskube, test.remainingPodCount)
	}
}

// TestTerminateNoVictimLogsInfo tests that missing victim prints a log message
func TestTerminateNoVictimLogsInfo(t *testing.T) {
	logOutput.Reset()
	chaoskube := New(fake.NewSimpleClientset(), labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, logger, false, 0)

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateLog(t, msgVictimNotFound)
}

// helper functions

func validateCandidatesCount(t *testing.T, chaoskube *Chaoskube, expected int) {
	pods, err := chaoskube.Candidates()
	if err != nil {
		t.Fatal(err)
	}

	if len(pods) != expected {
		t.Errorf("expected %d pods, got %d pods", expected, len(pods))
	}
}

func validateCandidates(t *testing.T, chaoskube *Chaoskube, expected []map[string]string) {
	pods, err := chaoskube.Candidates()
	if err != nil {
		t.Fatal(err)
	}

	validatePods(t, pods, expected)
}

func validateVictim(t *testing.T, chaoskube *Chaoskube, expected map[string]string) {
	victim, err := chaoskube.Victim()
	if err != nil {
		t.Fatal(err)
	}

	validatePod(t, victim, expected)
}

func validatePods(t *testing.T, pods []v1.Pod, expected []map[string]string) {
	if len(pods) != len(expected) {
		t.Fatalf("expected %d pod(s), got %d", len(expected), len(pods))
	}

	for i, pod := range pods {
		validatePod(t, pod, expected[i])
	}
}

func validatePod(t *testing.T, pod v1.Pod, expected map[string]string) {
	if pod.Namespace != expected["namespace"] {
		t.Errorf("expected %s, got %s", expected["namespace"], pod.Namespace)
	}

	if pod.Name != expected["name"] {
		t.Errorf("expected %s, got %s", expected["name"], pod.Name)
	}
}

func validateLog(t *testing.T, msg string) {
	if !strings.Contains(logOutput.String(), msg) {
		t.Errorf("expected string '%s' in '%s'.", msg, logOutput.String())
	}
}

func setup(t *testing.T, labelSelector labels.Selector, annotations labels.Selector, namespaces labels.Selector, excludedWeekdays []time.Weekday, excludedTimesOfDay []util.TimePeriod, timezone *time.Location, dryRun bool, seed int64) *Chaoskube {
	pods := []v1.Pod{
		util.NewPod("default", "foo"),
		util.NewPod("testing", "bar"),
	}

	client := fake.NewSimpleClientset()

	for _, pod := range pods {
		if _, err := client.Core().Pods(pod.Namespace).Create(&pod); err != nil {
			t.Fatal(err)
		}
	}

	logOutput.Reset()

	return New(client, labelSelector, annotations, namespaces, excludedWeekdays, excludedTimesOfDay, timezone, logger, dryRun, seed)
}

// ThankGodItsFriday is a helper struct that contains a Now() function that always returns a Friday.
type ThankGodItsFriday struct{}

// Now returns a particular Friday.
func (t ThankGodItsFriday) Now() time.Time {
	blackFriday, _ := time.Parse(time.RFC1123, "Fri, 24 Sep 1869 15:04:05 UTC")
	return blackFriday
}

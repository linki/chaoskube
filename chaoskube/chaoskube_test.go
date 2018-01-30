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

// TestCandidates tests the set of pods available for termination
func TestCandidates(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, false, 0)

	validateCandidates(t, chaoskube, []map[string]string{
		{"namespace": "default", "name": "foo"},
		{"namespace": "testing", "name": "bar"},
	})
}

// TestCandidatesLabelSelector tests that the list of pods available for
// termination can be restricted by providing a label selector.
func TestCandidatesLabelSelector(t *testing.T) {
	selector, err := labels.Parse("app=foo")
	if err != nil {
		t.Fatal(err)
	}

	chaoskube := setup(t, selector, labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, false, 0)

	validateCandidates(t, chaoskube, []map[string]string{
		{"namespace": "default", "name": "foo"},
	})
}

// TestCandidatesExcludingLabelSelector tests that label selector supports exclusion
func TestCandidatesExcludingLabelSelector(t *testing.T) {
	selector, err := labels.Parse("app!=foo")
	if err != nil {
		t.Fatal(err)
	}

	chaoskube := setup(t, selector, labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, false, 0)

	validateCandidates(t, chaoskube, []map[string]string{
		{"namespace": "testing", "name": "bar"},
	})
}

// TestCandidatesAnnotationSelector tests that the list of pods available for
// termination can be restricted by providing an annotation selector.
func TestCandidatesAnnotationSelector(t *testing.T) {
	selector, err := labels.Parse("chaos=foo")
	if err != nil {
		t.Fatal(err)
	}

	chaoskube := setup(t, labels.Everything(), selector, labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, false, 0)

	validateCandidates(t, chaoskube, []map[string]string{
		{"namespace": "default", "name": "foo"},
	})
}

// TestCandidatesExcludingAnnotationSelector tests that annotation selector supports exclusion
func TestCandidatesExcludingAnnotationSelector(t *testing.T) {
	selector, err := labels.Parse("chaos!=foo")
	if err != nil {
		t.Fatal(err)
	}

	chaoskube := setup(t, labels.Everything(), selector, labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, false, 0)

	validateCandidates(t, chaoskube, []map[string]string{
		{"namespace": "testing", "name": "bar"},
	})
}

// TestCandidatesNamespaces tests that the list of pods available for
// termination can be restricted by namespaces.
func TestCandidatesNamespaces(t *testing.T) {
	foo := map[string]string{"namespace": "default", "name": "foo"}
	bar := map[string]string{"namespace": "testing", "name": "bar"}

	for _, test := range []struct {
		namespaces string
		pods       []map[string]string
	}{
		{"", []map[string]string{foo, bar}},
		{"default", []map[string]string{foo}},
		{"default,testing", []map[string]string{foo, bar}},
		{"!testing", []map[string]string{foo}},
		{"!default,!testing", []map[string]string{}},
		{"default,!testing", []map[string]string{foo}},
		{"default,!default", []map[string]string{}},
	} {
		namespaces, err := labels.Parse(test.namespaces)
		if err != nil {
			t.Fatal(err)
		}

		chaoskube := setup(t, labels.Everything(), labels.Everything(), namespaces, []time.Weekday{}, []util.TimePeriod{}, time.UTC, false, 0)

		validateCandidates(t, chaoskube, test.pods)
	}
}

// TestVictim tests that a pod is chosen from the candidates
func TestVictim(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, false, 2000)

	validateVictim(t, chaoskube, map[string]string{
		"namespace": "default", "name": "foo",
	})
}

// TestAnotherVictim tests that the chosen victim is different for another seed
func TestAnotherVictim(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, false, 4000)

	validateVictim(t, chaoskube, map[string]string{
		"namespace": "testing", "name": "bar",
	})
}

// TestAnotherVictimRespectsLabelSelector tests that a pod chosen from the
// candidates respects the provided label selector
func TestAnotherVictimRespectsLabelSelector(t *testing.T) {
	selector, err := labels.Parse("app=foo")
	if err != nil {
		t.Fatal(err)
	}

	chaoskube := setup(t, selector, labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, false, 0)

	validateVictim(t, chaoskube, map[string]string{
		"namespace": "default", "name": "foo",
	})
}

// TestNoVictimReturnsError tests that on missing victim it returns a known error
func TestNoVictimReturnsError(t *testing.T) {
	chaoskube := New(fake.NewSimpleClientset(), labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, logger, false, 0)

	if _, err := chaoskube.Victim(); err != ErrPodNotFound {
		t.Errorf("expected %#v, got %#v", ErrPodNotFound, err)
	}
}

// TestDeletePod tests deleting a particular pod
func TestDeletePod(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, false, 0)

	victim := util.NewPod("default", "foo")

	if err := chaoskube.DeletePod(victim); err != nil {
		t.Fatal(err)
	}

	validateLog(t, "Killing pod default/foo")

	validateCandidates(t, chaoskube, []map[string]string{
		{"namespace": "testing", "name": "bar"},
	})
}

// TestDeletePodDryRun tests that enabled dry run doesn't delete the pod
func TestDeletePodDryRun(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, true, 0)

	victim := util.NewPod("default", "foo")

	if err := chaoskube.DeletePod(victim); err != nil {
		t.Fatal(err)
	}

	validateCandidates(t, chaoskube, []map[string]string{
		{"namespace": "default", "name": "foo"},
		{"namespace": "testing", "name": "bar"},
	})
}

// TestTerminateVictim tests that the correct victim pod is chosen and deleted
func TestTerminateVictim(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{}, time.UTC, false, 0)

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateCandidatesCount(t, chaoskube, 1)
}

// TestTerminateVictimRespectsExcludedWeekday tests that no victim is terminated when the current weekday is excluded.
func TestTerminateVictimRespectsExcludedWeekdays(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{time.Friday}, []util.TimePeriod{}, time.UTC, false, 0)

	// simulate that it's a Friday in our test (UTC).
	chaoskube.Now = ThankGodItsFriday{}.Now

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateCandidatesCount(t, chaoskube, 2)
	validateLog(t, msgWeekdayExcluded)
}

// TestTerminateVictimRespectsExcludedTimePeriods tests that no victim is terminated when the current time of day is excluded.
func TestTerminateVictimRespectsExcludedTimePeriods(t *testing.T) {
	afternoon := util.NewTimePeriod(ThankGodItsFriday{}.Now().Add(-1*time.Hour), ThankGodItsFriday{}.Now().Add(+1*time.Hour))

	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{afternoon}, time.UTC, false, 0)

	// simulate that it's a Friday afternoon in our test.
	chaoskube.Now = ThankGodItsFriday{}.Now

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateCandidatesCount(t, chaoskube, 2)
	validateLog(t, msgTimeOfDayExcluded)
}

// TestTerminateVictimOnNonExcludedWeekdays tests that victim is terminated when weekday filter doesn't match.
func TestTerminateVictimOnNonExcludedWeekdays(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{time.Friday}, []util.TimePeriod{}, time.UTC, false, 0)

	// simulate that it's a Saturday in our test.
	chaoskube.Now = func() time.Time { return ThankGodItsFriday{}.Now().Add(24 * time.Hour) }

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateCandidatesCount(t, chaoskube, 1)
}

// TestTerminateVictimOnNonActiveTimePeriod tests that victim is terminated when time period filter doesn't match.
func TestTerminateVictimOnNonActiveTimePeriod(t *testing.T) {
	afternoon := util.NewTimePeriod(ThankGodItsFriday{}.Now().Add(-1*time.Hour), ThankGodItsFriday{}.Now().Add(+1*time.Hour))

	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{afternoon}, time.UTC, false, 0)

	// simulate that it's a late Friday afternoon in our test.
	chaoskube.Now = func() time.Time { return ThankGodItsFriday{}.Now().Add(+2 * time.Hour) }

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateCandidatesCount(t, chaoskube, 1)
}

// TestTerminateVictimRespectsExcludedWeekdaysWithinTimezone tests that victim is terminated when weekday filter doesn't match due to different timezone.
func TestTerminateVictimRespectsExcludedWeekdaysWithinTimezone(t *testing.T) {
	timezone, err := time.LoadLocation("Australia/Brisbane")
	if err != nil {
		t.Fatal(err)
	}

	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{time.Friday}, []util.TimePeriod{}, timezone, false, 0)

	// simulate that it's a Friday in our test. However, in Australia it's already Saturday.
	chaoskube.Now = ThankGodItsFriday{}.Now

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateCandidatesCount(t, chaoskube, 1)
}

// TestTerminateVictimRespectsExcludedTimePeriodsWithinTimezone tests that victim is terminated when time period filter doesn't match due to different timezone.
func TestTerminateVictimRespectsExcludedTimePeriodsWithinTimezone(t *testing.T) {
	timezone, err := time.LoadLocation("Australia/Brisbane")
	if err != nil {
		t.Fatal(err)
	}

	afternoon := util.NewTimePeriod(ThankGodItsFriday{}.Now().Add(-1*time.Hour), ThankGodItsFriday{}.Now().Add(+1*time.Hour))

	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{afternoon}, timezone, false, 0)

	// simulate that it's a Friday afternoon in our test. However, in Australia it's already Saturday morning.
	chaoskube.Now = ThankGodItsFriday{}.Now

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateCandidatesCount(t, chaoskube, 1)
}

// TestTerminateVictimRespectsExcludedMultipleTimePeriods tests that multiple time periods can be provided and are correctly matched.
func TestTerminateVictimRespectsExcludedMultipleTimePeriods(t *testing.T) {
	morning := util.NewTimePeriod(ThankGodItsFriday{}.Now().Add(-7*time.Hour), ThankGodItsFriday{}.Now().Add(-6*time.Hour))
	afternoon := util.NewTimePeriod(ThankGodItsFriday{}.Now().Add(-1*time.Hour), ThankGodItsFriday{}.Now().Add(+1*time.Hour))

	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{morning, afternoon}, time.UTC, false, 0)

	// simulate that it's a Friday afternoon in our test.
	chaoskube.Now = ThankGodItsFriday{}.Now

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateCandidatesCount(t, chaoskube, 2)
	validateLog(t, msgTimeOfDayExcluded)
}

// TestTerminateVictimRespectsExcludedTimePeriodsOnEachDay tests that a filtered time period is relatively applied to every day.
func TestTerminateVictimRespectsExcludedTimePeriodsOnEachDay(t *testing.T) {
	afternoon := util.NewTimePeriod(ThankGodItsFriday{}.Now().Add(-1*time.Hour), ThankGodItsFriday{}.Now().Add(+1*time.Hour))

	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{afternoon}, time.UTC, false, 0)

	// simulate that it's the next day during the same time period.
	chaoskube.Now = func() time.Time { return ThankGodItsFriday{}.Now().Add(+24 * time.Hour) }

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateCandidatesCount(t, chaoskube, 2)
}

// TestTerminateVictimRespectsExcludedTimePeriodsAcrossDayBoundaryMatching tests that a time period across days is matched correctly (here it should be matched).
func TestTerminateVictimRespectsExcludedTimePeriodsAcrossDayBoundaryMatching(t *testing.T) {
	midnight := util.NewTimePeriod(ThankGodItsFriday{}.Now().Add(-16*time.Hour), ThankGodItsFriday{}.Now().Add(-14*time.Hour))

	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{midnight}, time.UTC, false, 0)

	// simulate that it's right in the middle of the excluded time period.
	chaoskube.Now = func() time.Time { return ThankGodItsFriday{}.Now().Add(-15 * time.Hour) }

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateCandidatesCount(t, chaoskube, 2)
}

// TestTerminateVictimRespectsExcludedTimePeriodsAcrossDayBoundaryBefore tests that a time period across days is matched correctly (here it shouldn't be matched).
func TestTerminateVictimRespectsExcludedTimePeriodsAcrossDayBoundaryBefore(t *testing.T) {
	midnight := util.NewTimePeriod(ThankGodItsFriday{}.Now().Add(-16*time.Hour), ThankGodItsFriday{}.Now().Add(-14*time.Hour))

	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{midnight}, time.UTC, false, 0)

	// simulate a time of day right before the excluded time period.
	chaoskube.Now = func() time.Time { return ThankGodItsFriday{}.Now().Add(-17 * time.Hour) }

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateCandidatesCount(t, chaoskube, 1)
}

// TestTerminateVictimRespectsExcludedTimePeriodsAcrossDayBoundaryAfter tests that a time period across days is matched correctly (here it shouldn't be matched).
func TestTerminateVictimRespectsExcludedTimePeriodsAcrossDayBoundaryAfter(t *testing.T) {
	midnight := util.NewTimePeriod(ThankGodItsFriday{}.Now().Add(-16*time.Hour), ThankGodItsFriday{}.Now().Add(-14*time.Hour))

	chaoskube := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), []time.Weekday{}, []util.TimePeriod{midnight}, time.UTC, false, 0)

	// simulate a time of day right after the excluded time period.
	chaoskube.Now = func() time.Time { return ThankGodItsFriday{}.Now().Add(-13 * time.Hour) }

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateCandidatesCount(t, chaoskube, 1)
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

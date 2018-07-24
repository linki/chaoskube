package chaoskube

import (
	"math/rand"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/linki/chaoskube/util"

	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
}

var (
	logger, logOutput = test.NewNullLogger()
)

func (suite *Suite) SetupTest() {
	logger.SetLevel(log.DebugLevel)
	logOutput.Reset()
}

// TestNew tests that arguments are passed to the new instance correctly
func (suite *Suite) TestNew() {
	var (
		client             = fake.NewSimpleClientset()
		labelSelector, _   = labels.Parse("foo=bar")
		annotations, _     = labels.Parse("baz=waldo")
		namespaces, _      = labels.Parse("qux")
		excludedWeekdays   = []time.Weekday{time.Friday}
		excludedTimesOfDay = []util.TimePeriod{util.TimePeriod{}}
		excludedDaysOfYear = []time.Time{time.Now()}
		minimumAge         = time.Duration(42)
	)

	chaoskube := New(
		client,
		labelSelector,
		annotations,
		namespaces,
		excludedWeekdays,
		excludedTimesOfDay,
		excludedDaysOfYear,
		time.UTC,
		minimumAge,
		logger,
		false,
	)
	suite.Require().NotNil(chaoskube)

	suite.Equal(client, chaoskube.Client)
	suite.Equal("foo=bar", chaoskube.Labels.String())
	suite.Equal("baz=waldo", chaoskube.Annotations.String())
	suite.Equal("qux", chaoskube.Namespaces.String())
	suite.Equal(excludedWeekdays, chaoskube.ExcludedWeekdays)
	suite.Equal(excludedTimesOfDay, chaoskube.ExcludedTimesOfDay)
	suite.Equal(excludedDaysOfYear, chaoskube.ExcludedDaysOfYear)
	suite.Equal(time.UTC, chaoskube.Timezone)
	suite.Equal(minimumAge, chaoskube.MinimumAge)
	suite.Equal(logger, chaoskube.Logger)
	suite.Equal(false, chaoskube.DryRun)
}

func (suite *Suite) TestCandidates() {
	foo := map[string]string{"namespace": "default", "name": "foo"}
	bar := map[string]string{"namespace": "testing", "name": "bar"}

	for _, tt := range []struct {
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
		labelSelector, err := labels.Parse(tt.labelSelector)
		suite.Require().NoError(err)

		annotationSelector, err := labels.Parse(tt.annotationSelector)
		suite.Require().NoError(err)

		namespaceSelector, err := labels.Parse(tt.namespaceSelector)
		suite.Require().NoError(err)

		chaoskube := suite.setupWithPods(
			labelSelector,
			annotationSelector,
			namespaceSelector,
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{},
			time.UTC,
			time.Duration(42),
			false,
		)

		suite.assertCandidates(chaoskube, tt.pods)
	}
}

func (suite *Suite) TestVictim() {
	foo := map[string]string{"namespace": "default", "name": "foo"}
	bar := map[string]string{"namespace": "testing", "name": "bar"}

	for _, tt := range []struct {
		seed          int64
		labelSelector string
		victim        map[string]string
	}{
		{2000, "", foo},
		{4000, "", bar},
		{4000, "app=foo", foo},
	} {
		rand.Seed(tt.seed)

		labelSelector, err := labels.Parse(tt.labelSelector)
		suite.Require().NoError(err)

		chaoskube := suite.setupWithPods(
			labelSelector,
			labels.Everything(),
			labels.Everything(),
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{},
			time.UTC,
			time.Duration(42),
			false,
		)

		suite.assertVictim(chaoskube, tt.victim)
	}
}

// TestNoVictimReturnsError tests that on missing victim it returns a known error
func (suite *Suite) TestNoVictimReturnsError() {
	chaoskube := suite.setup(
		labels.Everything(),
		labels.Everything(),
		labels.Everything(),
		[]time.Weekday{},
		[]util.TimePeriod{},
		[]time.Time{},
		time.UTC,
		time.Duration(42),
		false,
	)

	_, err := chaoskube.Victim()
	suite.Equal(err, errPodNotFound)
	suite.EqualError(err, "pod not found")
}

func (suite *Suite) TestDeletePod() {
	foo := map[string]string{"namespace": "default", "name": "foo"}
	bar := map[string]string{"namespace": "testing", "name": "bar"}

	for _, tt := range []struct {
		dryRun        bool
		remainingPods []map[string]string
	}{
		{false, []map[string]string{bar}},
		{true, []map[string]string{foo, bar}},
	} {
		chaoskube := suite.setupWithPods(
			labels.Everything(),
			labels.Everything(),
			labels.Everything(),
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{},
			time.UTC,
			time.Duration(42),
			tt.dryRun,
		)

		victim := util.NewPod("default", "foo", v1.PodRunning)

		err := chaoskube.DeletePod(victim)
		suite.Require().NoError(err)

		suite.assertLog(log.InfoLevel, "terminating pod", log.Fields{"namespace": "default", "name": "foo"})
		suite.assertCandidates(chaoskube, tt.remainingPods)
	}
}

func (suite *Suite) TestTerminateVictim() {
	midnight := util.NewTimePeriod(
		ThankGodItsFriday{}.Now().Add(-16*time.Hour),
		ThankGodItsFriday{}.Now().Add(-14*time.Hour),
	)
	morning := util.NewTimePeriod(
		ThankGodItsFriday{}.Now().Add(-7*time.Hour),
		ThankGodItsFriday{}.Now().Add(-6*time.Hour),
	)
	afternoon := util.NewTimePeriod(
		ThankGodItsFriday{}.Now().Add(-1*time.Hour),
		ThankGodItsFriday{}.Now().Add(+1*time.Hour),
	)

	australia, err := time.LoadLocation("Australia/Brisbane")
	suite.Require().NoError(err)

	for _, tt := range []struct {
		excludedWeekdays   []time.Weekday
		excludedTimesOfDay []util.TimePeriod
		excludedDaysOfYear []time.Time
		now                func() time.Time
		timezone           *time.Location
		remainingPodCount  int
	}{
		// no time is excluded, one pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{},
			ThankGodItsFriday{}.Now,
			time.UTC,
			1,
		},
		// current weekday is excluded, no pod should be killed
		{
			[]time.Weekday{time.Friday},
			[]util.TimePeriod{},
			[]time.Time{},
			ThankGodItsFriday{}.Now,
			time.UTC,
			2,
		},
		// current time of day is excluded, no pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{afternoon},
			[]time.Time{},
			ThankGodItsFriday{}.Now,
			time.UTC,
			2,
		},
		// one day after an excluded weekday, one pod should be killed
		{
			[]time.Weekday{time.Friday},
			[]util.TimePeriod{},
			[]time.Time{},
			func() time.Time { return ThankGodItsFriday{}.Now().Add(24 * time.Hour) },
			time.UTC,
			1,
		},
		// seven days after an excluded weekday, no pod should be killed
		{
			[]time.Weekday{time.Friday},
			[]util.TimePeriod{},
			[]time.Time{},
			func() time.Time { return ThankGodItsFriday{}.Now().Add(7 * 24 * time.Hour) },
			time.UTC,
			2,
		},
		// one hour after an excluded time period, one pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{afternoon},
			[]time.Time{},
			func() time.Time { return ThankGodItsFriday{}.Now().Add(+2 * time.Hour) },
			time.UTC,
			1,
		},
		// twenty four hours after an excluded time period, no pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{afternoon},
			[]time.Time{},
			func() time.Time { return ThankGodItsFriday{}.Now().Add(+24 * time.Hour) },
			time.UTC,
			2,
		},
		// current weekday is excluded but we are in another time zone, one pod should be killed
		{
			[]time.Weekday{time.Friday},
			[]util.TimePeriod{},
			[]time.Time{},
			ThankGodItsFriday{}.Now,
			australia,
			1,
		},
		// current time period is excluded but we are in another time zone, one pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{afternoon},
			[]time.Time{},
			ThankGodItsFriday{}.Now,
			australia,
			1,
		},
		// one out of two excluded weeksdays match, no pod should be killed
		{
			[]time.Weekday{time.Monday, time.Friday},
			[]util.TimePeriod{},
			[]time.Time{},
			ThankGodItsFriday{}.Now,
			time.UTC,
			2,
		},
		// one out of two excluded time periods match, no pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{morning, afternoon},
			[]time.Time{},
			ThankGodItsFriday{}.Now,
			time.UTC,
			2,
		},
		// we're inside an excluded time period across days, no pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{midnight},
			[]time.Time{},
			func() time.Time { return ThankGodItsFriday{}.Now().Add(-15 * time.Hour) },
			time.UTC,
			2,
		},
		// we're before an excluded time period across days, one pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{midnight},
			[]time.Time{},
			func() time.Time { return ThankGodItsFriday{}.Now().Add(-17 * time.Hour) },
			time.UTC,
			1,
		},
		// we're after an excluded time period across days, one pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{midnight},
			[]time.Time{},
			func() time.Time { return ThankGodItsFriday{}.Now().Add(-13 * time.Hour) },
			time.UTC,
			1,
		},
		// this day of year is excluded, no pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{
				ThankGodItsFriday{}.Now(), // today
			},
			func() time.Time { return ThankGodItsFriday{}.Now() },
			time.UTC,
			2,
		},
		// this day of year in year 0 is excluded, no pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{
				time.Date(0, 9, 24, 0, 00, 00, 00, time.UTC), // same year day
			},
			func() time.Time { return ThankGodItsFriday{}.Now() },
			time.UTC,
			2,
		},
		// matching works fine even when multiple days-of-year are provided, no pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{
				time.Date(0, 9, 25, 10, 00, 00, 00, time.UTC), // different year day
				time.Date(0, 9, 24, 10, 00, 00, 00, time.UTC), // same year day
			},
			func() time.Time { return ThankGodItsFriday{}.Now() },
			time.UTC,
			2,
		},
		// there is an excluded day of year but it's not today, one pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{
				time.Date(0, 9, 25, 10, 00, 00, 00, time.UTC), // different year day
			},
			func() time.Time { return ThankGodItsFriday{}.Now() },
			time.UTC,
			1,
		},
		// there is an excluded day of year but the month is different, one pod should be killed
		{
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{
				time.Date(0, 10, 24, 10, 00, 00, 00, time.UTC), // different year day
			},
			func() time.Time { return ThankGodItsFriday{}.Now() },
			time.UTC,
			1,
		},
	} {
		chaoskube := suite.setupWithPods(
			labels.Everything(),
			labels.Everything(),
			labels.Everything(),
			tt.excludedWeekdays,
			tt.excludedTimesOfDay,
			tt.excludedDaysOfYear,
			tt.timezone,
			time.Duration(42),
			false,
		)
		chaoskube.Now = tt.now

		err := chaoskube.TerminateVictim()
		suite.Require().NoError(err)

		pods, err := chaoskube.Candidates()
		suite.Require().NoError(err)

		suite.Len(pods, tt.remainingPodCount)
	}
}

// TestTerminateNoVictimLogsInfo tests that missing victim prints a log message
func (suite *Suite) TestTerminateNoVictimLogsInfo() {
	chaoskube := suite.setup(
		labels.Everything(),
		labels.Everything(),
		labels.Everything(),
		[]time.Weekday{},
		[]util.TimePeriod{},
		[]time.Time{},
		time.UTC,
		time.Duration(42),
		false,
	)

	err := chaoskube.TerminateVictim()
	suite.Require().NoError(err)

	suite.assertLog(log.DebugLevel, msgVictimNotFound, log.Fields{})
}

// helper functions

func (suite *Suite) assertCandidates(chaoskube *Chaoskube, expected []map[string]string) {
	pods, err := chaoskube.Candidates()
	suite.Require().NoError(err)

	suite.assertPods(pods, expected)
}

func (suite *Suite) assertVictim(chaoskube *Chaoskube, expected map[string]string) {
	victim, err := chaoskube.Victim()
	suite.Require().NoError(err)

	suite.assertPod(victim, expected)
}

func (suite *Suite) assertPods(pods []v1.Pod, expected []map[string]string) {
	suite.Require().Len(pods, len(expected))

	for i, pod := range pods {
		suite.assertPod(pod, expected[i])
	}
}

func (suite *Suite) assertPod(pod v1.Pod, expected map[string]string) {
	suite.Equal(expected["namespace"], pod.Namespace)
	suite.Equal(expected["name"], pod.Name)
}

func (suite *Suite) assertLog(level log.Level, msg string, fields log.Fields) {
	suite.Require().NotEmpty(logOutput.Entries)

	lastEntry := logOutput.LastEntry()
	suite.Equal(level, lastEntry.Level)
	suite.Equal(msg, lastEntry.Message)
	for k := range fields {
		suite.Equal(fields[k], lastEntry.Data[k])
	}
}

func (suite *Suite) setupWithPods(labelSelector labels.Selector, annotations labels.Selector, namespaces labels.Selector, excludedWeekdays []time.Weekday, excludedTimesOfDay []util.TimePeriod, excludedDaysOfYear []time.Time, timezone *time.Location, minimumAge time.Duration, dryRun bool) *Chaoskube {
	chaoskube := suite.setup(
		labelSelector,
		annotations,
		namespaces,
		excludedWeekdays,
		excludedTimesOfDay,
		excludedDaysOfYear,
		timezone,
		minimumAge,
		dryRun,
	)

	pods := []v1.Pod{
		util.NewPod("default", "foo", v1.PodRunning),
		util.NewPod("testing", "bar", v1.PodRunning),
		util.NewPod("testing", "baz", v1.PodPending), // Non-running pods are ignored
	}

	for _, pod := range pods {
		_, err := chaoskube.Client.Core().Pods(pod.Namespace).Create(&pod)
		suite.Require().NoError(err)
	}

	return chaoskube
}

func (suite *Suite) setup(labelSelector labels.Selector, annotations labels.Selector, namespaces labels.Selector, excludedWeekdays []time.Weekday, excludedTimesOfDay []util.TimePeriod, excludedDaysOfYear []time.Time, timezone *time.Location, minimumAge time.Duration, dryRun bool) *Chaoskube {
	logOutput.Reset()

	return New(
		fake.NewSimpleClientset(),
		labelSelector,
		annotations,
		namespaces,
		excludedWeekdays,
		excludedTimesOfDay,
		excludedDaysOfYear,
		timezone,
		minimumAge,
		logger,
		dryRun,
	)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

// ThankGodItsFriday is a helper struct that contains a Now() function that always returns a Friday.
type ThankGodItsFriday struct{}

// Now returns a particular Friday.
func (t ThankGodItsFriday) Now() time.Time {
	blackFriday, _ := time.Parse(time.RFC1123, "Fri, 24 Sep 1869 15:04:05 UTC")
	return blackFriday
}

func (suite *Suite) TestMinimumAge() {
	type pod struct {
		name         string
		namespace    string
		creationTime time.Time
	}

	for _, tt := range []struct {
		minimumAge time.Duration
		now        func() time.Time
		pods       []pod
		candidates int
	}{
		// no minimum age set
		{
			time.Duration(0),
			func() time.Time { return time.Date(0, 10, 24, 10, 00, 00, 00, time.UTC) },
			[]pod{
				{
					name:         "test1",
					namespace:    "test",
					creationTime: time.Date(0, 10, 24, 9, 00, 00, 00, time.UTC),
				},
			},
			1,
		},
		// minimum age set, but pod is too young
		{
			time.Hour * 1,
			func() time.Time { return time.Date(0, 10, 24, 10, 00, 00, 00, time.UTC) },
			[]pod{
				{
					name:         "test1",
					namespace:    "test",
					creationTime: time.Date(0, 10, 24, 9, 30, 00, 00, time.UTC),
				},
			},
			0,
		},
		// one pod is too young, one matches
		{
			time.Hour * 1,
			func() time.Time { return time.Date(0, 10, 24, 10, 00, 00, 00, time.UTC) },
			[]pod{
				// too young
				{
					name:         "test1",
					namespace:    "test",
					creationTime: time.Date(0, 10, 24, 9, 30, 00, 00, time.UTC),
				},
				// matches
				{
					name:         "test2",
					namespace:    "test",
					creationTime: time.Date(0, 10, 23, 8, 00, 00, 00, time.UTC),
				},
			},
			1,
		},
		// exact time - should not match
		{
			time.Hour * 1,
			func() time.Time { return time.Date(0, 10, 24, 10, 00, 00, 00, time.UTC) },
			[]pod{
				{
					name:         "test1",
					namespace:    "test",
					creationTime: time.Date(0, 10, 24, 10, 00, 00, 00, time.UTC),
				},
			},
			0,
		},
	} {
		chaoskube := suite.setup(
			labels.Everything(),
			labels.Everything(),
			labels.Everything(),
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{},
			time.UTC,
			tt.minimumAge,
			false,
		)
		chaoskube.Now = tt.now

		for _, p := range tt.pods {
			pod := util.NewPod(p.namespace, p.name, v1.PodRunning)
			pod.ObjectMeta.CreationTimestamp = metav1.Time{Time: p.creationTime}
			_, err := chaoskube.Client.Core().Pods(pod.Namespace).Create(&pod)
			suite.Require().NoError(err)
		}

		pods, err := chaoskube.Candidates()
		suite.Require().NoError(err)

		suite.Len(pods, tt.candidates)
	}
}

package chaoskube

import (
	"math/rand"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"

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
	)

	chaoskube := New(
		client,
		labelSelector,
		annotations,
		namespaces,
		excludedWeekdays,
		excludedTimesOfDay,
		time.UTC,
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
	suite.Equal(time.UTC, chaoskube.Timezone)
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
			time.UTC,
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
			time.UTC,
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
		time.UTC,
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
			time.UTC,
			tt.dryRun,
		)

		victim := util.NewPod("default", "foo")

		err := chaoskube.DeletePod(victim)
		suite.Require().NoError(err)

		suite.assertLog("killing pod", log.Fields{"namespace": "default", "name": "foo"})
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
		chaoskube := suite.setupWithPods(
			labels.Everything(),
			labels.Everything(),
			labels.Everything(),
			tt.excludedWeekdays,
			tt.excludedTimesOfDay,
			tt.timezone,
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
		time.UTC,
		false,
	)

	err := chaoskube.TerminateVictim()
	suite.Require().NoError(err)

	suite.assertLog(msgVictimNotFound, log.Fields{})
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

func (suite *Suite) assertLog(msg string, fields log.Fields) {
	suite.Require().Len(logOutput.Entries, 1)

	lastEntry := logOutput.LastEntry()
	suite.Equal(log.InfoLevel, lastEntry.Level)
	suite.Equal(msg, lastEntry.Message)
	for k := range fields {
		suite.Equal(fields[k], lastEntry.Data[k])
	}
}

func (suite *Suite) setupWithPods(labelSelector labels.Selector, annotations labels.Selector, namespaces labels.Selector, excludedWeekdays []time.Weekday, excludedTimesOfDay []util.TimePeriod, timezone *time.Location, dryRun bool) *Chaoskube {
	chaoskube := suite.setup(
		labelSelector,
		annotations,
		namespaces,
		excludedWeekdays,
		excludedTimesOfDay,
		timezone,
		dryRun,
	)

	pods := []v1.Pod{
		util.NewPod("default", "foo"),
		util.NewPod("testing", "bar"),
	}

	for _, pod := range pods {
		_, err := chaoskube.Client.Core().Pods(pod.Namespace).Create(&pod)
		suite.Require().NoError(err)
	}

	return chaoskube
}

func (suite *Suite) setup(labelSelector labels.Selector, annotations labels.Selector, namespaces labels.Selector, excludedWeekdays []time.Weekday, excludedTimesOfDay []util.TimePeriod, timezone *time.Location, dryRun bool) *Chaoskube {
	logOutput.Reset()

	return New(
		fake.NewSimpleClientset(),
		labelSelector,
		annotations,
		namespaces,
		excludedWeekdays,
		excludedTimesOfDay,
		timezone,
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

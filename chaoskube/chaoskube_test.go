package chaoskube

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/linki/chaoskube/internal/testutil"
	"github.com/linki/chaoskube/notifier"
	"github.com/linki/chaoskube/terminator"
	"github.com/linki/chaoskube/util"

	"github.com/stretchr/testify/suite"
)

type Suite struct {
	testutil.TestSuite
}

var (
	logger, logOutput = test.NewNullLogger()
	testNotifier      = &notifier.Noop{}
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
		namespaceLabels, _ = labels.Parse("taz=wubble")
		includedPodNames   = regexp.MustCompile("foo")
		excludedPodNames   = regexp.MustCompile("bar")
		excludedWeekdays   = []time.Weekday{time.Friday}
		excludedTimesOfDay = []util.TimePeriod{util.TimePeriod{}}
		excludedDaysOfYear = []time.Time{time.Now()}
		minimumAge         = time.Duration(42)
		dryRun             = true
		terminator         = terminator.NewDeletePodTerminator(client, logger, 10*time.Second)
		maxKill            = 1
		notifier           = testNotifier
	)

	chaoskube := New(
		client,
		labelSelector,
		annotations,
		namespaces,
		namespaceLabels,
		includedPodNames,
		excludedPodNames,
		excludedWeekdays,
		excludedTimesOfDay,
		excludedDaysOfYear,
		time.UTC,
		minimumAge,
		logger,
		dryRun,
		terminator,
		maxKill,
		notifier,
	)
	suite.Require().NotNil(chaoskube)

	suite.Equal(client, chaoskube._Client)
	suite.Equal("foo=bar", chaoskube.Labels.String())
	suite.Equal("baz=waldo", chaoskube.Annotations.String())
	suite.Equal("qux", chaoskube.Namespaces.String())
	suite.Equal("taz=wubble", chaoskube.NamespaceLabels.String())
	suite.Equal("foo", chaoskube.IncludedPodNames.String())
	suite.Equal("bar", chaoskube.ExcludedPodNames.String())
	suite.Equal(excludedWeekdays, chaoskube.ExcludedWeekdays)
	suite.Equal(excludedTimesOfDay, chaoskube.ExcludedTimesOfDay)
	suite.Equal(excludedDaysOfYear, chaoskube.ExcludedDaysOfYear)
	suite.Equal(time.UTC, chaoskube.Timezone)
	suite.Equal(minimumAge, chaoskube.MinimumAge)
	suite.Equal(logger, chaoskube.Logger)
	suite.Equal(dryRun, chaoskube.DryRun)
	suite.Equal(terminator, chaoskube.Terminator)
}

// TestRunContextCanceled tests that a canceled context will exit the Run function.
func (suite *Suite) TestRunContextCanceled() {
	chaoskube := suite.setup(
		labels.Everything(),
		labels.Everything(),
		labels.Everything(),
		labels.Everything(),
		&regexp.Regexp{},
		&regexp.Regexp{},
		[]time.Weekday{},
		[]util.TimePeriod{},
		[]time.Time{},
		time.UTC,
		time.Duration(0),
		false,
		10,
		1,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	chaoskube.Run(ctx, nil)
}

// TestCandidates tests that the various pod filters are applied correctly.
func (suite *Suite) TestCandidates() {
	foo := map[string]string{"namespace": "default", "name": "foo"}
	bar := map[string]string{"namespace": "testing", "name": "bar"}

	_ = foo
	_ = bar

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
			labels.Everything(),
			nil,
			nil,
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{},
			time.UTC,
			time.Duration(0),
			false,
			10,
		)

		suite.assertCandidates(chaoskube, tt.pods)
	}
}

// TestCandidatesNamespaceLabels tests that the label selector for namespaces works correctly.
func (suite *Suite) TestCandidatesNamespaceLabels() {
	foo := map[string]string{"namespace": "default", "name": "foo"}
	bar := map[string]string{"namespace": "testing", "name": "bar"}

	for _, tt := range []struct {
		labels string
		pods   []map[string]string
	}{
		{"", []map[string]string{foo, bar}},
		{"env", []map[string]string{foo, bar}},
		{"!env", []map[string]string{}},
		{"env=default", []map[string]string{foo}},
		{"env=testing", []map[string]string{bar}},
		{"env!=default", []map[string]string{bar}},
		{"env!=testing", []map[string]string{foo}},
		{"env!=default,env!=testing", []map[string]string{}},
		{"env=default,env!=testing", []map[string]string{foo}},
		{"env=default,env!=default", []map[string]string{}},
		{"nomatch", []map[string]string{}},
	} {
		namespaceLabels, err := labels.Parse(tt.labels)
		suite.Require().NoError(err)

		chaoskube := suite.setupWithPods(
			labels.Everything(),
			labels.Everything(),
			labels.Everything(),
			namespaceLabels,
			nil,
			nil,
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{},
			time.UTC,
			time.Duration(0),
			false,
			10,
		)

		suite.assertCandidates(chaoskube, tt.pods)
	}
}

// TestCandidatesPodNameRegexp tests that the included and excluded pod name regular expressions
// are applied correctly.
func (suite *Suite) TestCandidatesPodNameRegexp() {
	foo := map[string]string{"namespace": "default", "name": "foo"}
	bar := map[string]string{"namespace": "testing", "name": "bar"}

	for _, tt := range []struct {
		includedPodNames *regexp.Regexp
		excludedPodNames *regexp.Regexp
		pods             []map[string]string
	}{
		// no included nor excluded regular expressions given
		{nil, nil, []map[string]string{foo, bar}},
		// either included or excluded regular expression given
		{regexp.MustCompile("fo.*"), nil, []map[string]string{foo}},
		{nil, regexp.MustCompile("fo.*"), []map[string]string{bar}},
		// either included or excluded regular expression is empty
		{regexp.MustCompile("fo.*"), regexp.MustCompile(""), []map[string]string{foo}},
		{regexp.MustCompile(""), regexp.MustCompile("fo.*"), []map[string]string{bar}},
		// both included and excluded regular expressions are considered
		{regexp.MustCompile("fo.*"), regexp.MustCompile("f.*"), []map[string]string{}},
	} {
		chaoskube := suite.setupWithPods(
			labels.Everything(),
			labels.Everything(),
			labels.Everything(),
			labels.Everything(),
			tt.includedPodNames,
			tt.excludedPodNames,
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{},
			time.UTC,
			time.Duration(0),
			false,
			10,
		)

		suite.assertCandidates(chaoskube, tt.pods)
	}
}

// TestVictim tests that a random victim is chosen from selected candidates.
func (suite *Suite) TestVictim() {
	foo := map[string]string{"namespace": "default", "name": "foo"}
	bar := map[string]string{"namespace": "testing", "name": "bar"}

	for _, tt := range []struct {
		labelSelector string
		victim        map[string]string
	}{
		// {"", fooOrBar},
		{"app=foo", foo},
		{"app=bar", bar},
	} {
		labelSelector, err := labels.Parse(tt.labelSelector)
		suite.Require().NoError(err)

		chaoskube := suite.setupWithPods(
			labelSelector,
			labels.Everything(),
			labels.Everything(),
			labels.Everything(),
			&regexp.Regexp{},
			&regexp.Regexp{},
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{},
			time.UTC,
			time.Duration(0),
			false,
			10,
		)

		suite.assertVictim(chaoskube, tt.victim)
	}
}

// TestVictims tests that a random subset of pods is chosen from selected candidates
func (suite *Suite) TestVictims() {
	foo := map[string]string{"namespace": "default", "name": "foo"}
	bar := map[string]string{"namespace": "testing", "name": "bar"}
	baz := map[string]string{"namespace": "test", "name": "baz"}

	for _, tt := range []struct {
		labelSelector string
		victims       []map[string]string
		maxKill       int
		killed        int
	}{
		{"", []map[string]string{foo, bar, baz}, 1, 1},
		{"", []map[string]string{foo, bar}, 2, 2},
		{"", []map[string]string{foo}, 3, 3},
		{"app=foo", []map[string]string{foo}, 2, 1},
		{"app=bar", []map[string]string{bar}, 2, 1},
	} {

		labelSelector, err := labels.Parse(tt.labelSelector)
		suite.Require().NoError(err)

		chaoskube := suite.setup(
			labelSelector,
			labels.Everything(),
			labels.Everything(),
			labels.Everything(),
			&regexp.Regexp{},
			&regexp.Regexp{},
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{},
			time.UTC,
			time.Duration(0),
			false,
			10,
			tt.maxKill,
		)
		suite.createPods(chaoskube._Client, []map[string]string{foo, bar, baz})

		_ = chaoskube._Informer.InformerFor(&v1.Pod{}, nil)

		stopChan := make(chan struct{})
		chaoskube._Informer.WaitForCacheSync(stopChan)

		// spew.Dump(inf.HasSynced())

		// time.Sleep(time.Second * 2)

		// vic, err := chaoskube.Victims()

		// suite.Require().Equal(len(vic), tt.killed)

		// spew.Dump(chaoskube.Victims())
		// spew.Dump(tt.victims)

		// it's just the order
		suite.assertVictims(chaoskube, tt.killed, tt.victims)
	}
}

// TestNoVictimReturnsError tests that on missing victim it returns a known error
func (suite *Suite) TestNoVictimReturnsError() {
	chaoskube := suite.setup(
		labels.Everything(),
		labels.Everything(),
		labels.Everything(),
		labels.Everything(),
		&regexp.Regexp{},
		&regexp.Regexp{},
		[]time.Weekday{},
		[]util.TimePeriod{},
		[]time.Time{},
		time.UTC,
		time.Duration(0),
		false,
		10,
		1,
	)

	_, err := chaoskube.Victims()
	suite.Equal(err, errPodNotFound)
	suite.EqualError(err, "pod not found")
}

// TestDeletePod tests that a given pod is deleted and dryRun is respected.
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
			labels.Everything(),
			&regexp.Regexp{},
			&regexp.Regexp{},
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{},
			time.UTC,
			time.Duration(0),
			tt.dryRun,
			10,
		)

		victim := util.NewPod("default", "foo", v1.PodRunning)

		err := chaoskube.DeletePod(victim)
		suite.Require().NoError(err)

		time.Sleep(time.Second)

		suite.AssertLog(logOutput, log.InfoLevel, "terminating pod", log.Fields{"namespace": "default", "name": "foo"})
		suite.assertCandidates(chaoskube, tt.remainingPods)
	}
}

// TestDeletePodNotFound tests missing target pod will return an error.
func (suite *Suite) TestDeletePodNotFound() {
	chaoskube := suite.setup(
		labels.Everything(),
		labels.Everything(),
		labels.Everything(),
		labels.Everything(),
		&regexp.Regexp{},
		&regexp.Regexp{},
		[]time.Weekday{},
		[]util.TimePeriod{},
		[]time.Time{},
		time.UTC,
		time.Duration(0),
		false,
		10,
		1,
	)

	victim := util.NewPod("default", "foo", v1.PodRunning)

	err := chaoskube.DeletePod(victim)
	suite.EqualError(err, `pods "foo" not found`)
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
			labels.Everything(),
			&regexp.Regexp{},
			&regexp.Regexp{},
			tt.excludedWeekdays,
			tt.excludedTimesOfDay,
			tt.excludedDaysOfYear,
			tt.timezone,
			time.Duration(0),
			false,
			10,
		)
		chaoskube.Now = tt.now

		time.Sleep(time.Second * 2)

		err := chaoskube.TerminateVictims()
		suite.Require().NoError(err)

		time.Sleep(time.Second * 2)

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
		labels.Everything(),
		&regexp.Regexp{},
		&regexp.Regexp{},
		[]time.Weekday{},
		[]util.TimePeriod{},
		[]time.Time{},
		time.UTC,
		time.Duration(0),
		false,
		10,
		1,
	)

	err := chaoskube.TerminateVictims()
	suite.Require().NoError(err)

	suite.AssertLog(logOutput, log.DebugLevel, msgVictimNotFound, log.Fields{})
}

// helper functions

func (suite *Suite) assertCandidates(chaoskube *Chaoskube, expected []map[string]string) {
	pods, err := chaoskube.Candidates()
	suite.Require().NoError(err)

	suite.AssertPods(pods, expected)
}

func (suite *Suite) assertVictims(chaoskube *Chaoskube, count int, expected []map[string]string) {
	victims, err := chaoskube.Victims()
	suite.Require().NoError(err)

	suite.Require().Len(victims, count)

	found := false

	for _, pod := range expected {
		for _, victim := range victims {
			if victim.Namespace == pod["namespace"] && victim.Name == pod["name"] {
				found = true
			}
		}
	}

	suite.True(found)
}

func (suite *Suite) assertVictim(chaoskube *Chaoskube, expected map[string]string) {
	suite.assertVictims(chaoskube, 1, []map[string]string{expected})
}

func (suite *Suite) assertNotified(notifier *notifier.Noop) {
	suite.Assert().Greater(notifier.Calls, 0)
}

func (suite *Suite) setupWithPods(labelSelector labels.Selector, annotations labels.Selector, namespaces labels.Selector, namespaceLabels labels.Selector, includedPodNames *regexp.Regexp, excludedPodNames *regexp.Regexp, excludedWeekdays []time.Weekday, excludedTimesOfDay []util.TimePeriod, excludedDaysOfYear []time.Time, timezone *time.Location, minimumAge time.Duration, dryRun bool, gracePeriod time.Duration) *Chaoskube {
	chaoskube := suite.setup(
		labelSelector,
		annotations,
		namespaces,
		namespaceLabels,
		includedPodNames,
		excludedPodNames,
		excludedWeekdays,
		excludedTimesOfDay,
		excludedDaysOfYear,
		timezone,
		minimumAge,
		dryRun,
		gracePeriod,
		1,
	)

	for _, namespace := range []v1.Namespace{
		util.NewNamespace("default"),
		util.NewNamespace("testing"),
	} {
		_, err := chaoskube._Client.CoreV1().Namespaces().Create(&namespace)
		suite.Require().NoError(err)
	}

	pods := []v1.Pod{
		util.NewPod("default", "foo", v1.PodRunning),
		util.NewPod("testing", "bar", v1.PodRunning),
		util.NewPod("testing", "baz", v1.PodPending), // Non-running pods are ignored
	}

	for _, pod := range pods {
		_, err := chaoskube._Client.CoreV1().Pods(pod.Namespace).Create(&pod)
		suite.Require().NoError(err)
	}

	_ = chaoskube._Informer.InformerFor(&v1.Pod{}, nil)

	stopChan := make(chan struct{})
	chaoskube._Informer.WaitForCacheSync(stopChan)

	// spew.Dump(inf.HasSynced())

	return chaoskube
}

func (suite *Suite) createPods(client kubernetes.Interface, podsInfo []map[string]string) {
	for _, p := range podsInfo {
		namespace := util.NewNamespace(p["namespace"])
		_, err := client.CoreV1().Namespaces().Create(&namespace)
		suite.Require().NoError(err)
		pod := util.NewPod(p["namespace"], p["name"], v1.PodRunning)
		_, err = client.CoreV1().Pods(p["namespace"]).Create(&pod)
		suite.Require().NoError(err)
	}
}

func (suite *Suite) setup(labelSelector labels.Selector, annotations labels.Selector, namespaces labels.Selector, namespaceLabels labels.Selector, includedPodNames *regexp.Regexp, excludedPodNames *regexp.Regexp, excludedWeekdays []time.Weekday, excludedTimesOfDay []util.TimePeriod, excludedDaysOfYear []time.Time, timezone *time.Location, minimumAge time.Duration, dryRun bool, gracePeriod time.Duration, maxKill int) *Chaoskube {
	logOutput.Reset()

	client := fake.NewSimpleClientset()
	nullLogger, _ := test.NewNullLogger()

	return New(
		client,
		labelSelector,
		annotations,
		namespaces,
		namespaceLabels,
		includedPodNames,
		excludedPodNames,
		excludedWeekdays,
		excludedTimesOfDay,
		excludedDaysOfYear,
		timezone,
		minimumAge,
		logger,
		dryRun,
		terminator.NewDeletePodTerminator(client, nullLogger, gracePeriod),
		maxKill,
		testNotifier,
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
			labels.Everything(),
			&regexp.Regexp{},
			&regexp.Regexp{},
			[]time.Weekday{},
			[]util.TimePeriod{},
			[]time.Time{},
			time.UTC,
			tt.minimumAge,
			false,
			10,
			1,
		)
		chaoskube.Now = tt.now

		for _, p := range tt.pods {
			pod := util.NewPod(p.namespace, p.name, v1.PodRunning)
			pod.ObjectMeta.CreationTimestamp = metav1.Time{Time: p.creationTime}
			_, err := chaoskube._Client.CoreV1().Pods(pod.Namespace).Create(&pod)
			suite.Require().NoError(err)
		}

		inf := chaoskube._Informer.InformerFor(&v1.Pod{}, nil)

		stopChan := make(chan struct{})
		chaoskube._Informer.WaitForCacheSync(stopChan)

		spew.Dump(inf.HasSynced())

		pods, err := chaoskube.Candidates()
		suite.Require().NoError(err)

		suite.Len(pods, tt.candidates)
	}
}

func (suite *Suite) TestFilterDeletedPods() {
	deletedPod := util.NewPod("default", "deleted", v1.PodRunning)
	now := metav1.NewTime(time.Now())
	deletedPod.SetDeletionTimestamp(&now)

	runningPod := util.NewPod("default", "running", v1.PodRunning)

	pods := []v1.Pod{runningPod, deletedPod}

	filtered := filterTerminatingPods(pods)
	suite.Equal(len(filtered), 1)
	suite.Equal(pods[0].Name, "running")
}

func (suite *Suite) TestFilterByOwnerReference() {
	foo := util.NewPodWithOwner("default", "foo", v1.PodRunning, "parent")
	foo1 := util.NewPodWithOwner("default", "foo-1", v1.PodRunning, "parent")
	bar := util.NewPodWithOwner("default", "bar", v1.PodRunning, "other-parent")
	baz := util.NewPod("default", "baz", v1.PodRunning)
	baz1 := util.NewPod("default", "baz-1", v1.PodRunning)

	for _, tt := range []struct {
		name     string
		pods     []v1.Pod
		expected []v1.Pod
	}{
		{
			name:     "2 pods, same parent",
			pods:     []v1.Pod{foo, foo1},
			expected: []v1.Pod{foo},
		},
		{
			name:     "2 pods, different parents",
			pods:     []v1.Pod{foo, bar},
			expected: []v1.Pod{foo, bar},
		},
		{
			name:     "2 pods, one with/without parent",
			pods:     []v1.Pod{foo, baz},
			expected: []v1.Pod{foo, baz},
		},
		{
			name:     "2 pods, no parents",
			pods:     []v1.Pod{baz, baz1},
			expected: []v1.Pod{baz, baz1},
		},
	} {
		results := filterByOwnerReference(tt.pods)
		for i, result := range results {
			suite.Assert().Equal(tt.expected[i], result, tt.name)
		}
	}
}

func (suite *Suite) TestNotifierCall() {
	chaoskube := suite.setupWithPods(
		labels.Everything(),
		labels.Everything(),
		labels.Everything(),
		labels.Everything(),
		&regexp.Regexp{},
		&regexp.Regexp{},
		[]time.Weekday{},
		[]util.TimePeriod{},
		[]time.Time{},
		time.UTC,
		time.Duration(0),
		false,
		10,
	)

	victim := util.NewPod("default", "foo", v1.PodRunning)
	err := chaoskube.DeletePod(victim)

	suite.Require().NoError(err)
	suite.assertNotified(testNotifier)
}

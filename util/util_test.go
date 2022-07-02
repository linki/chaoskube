package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
)

type Suite struct {
	suite.Suite
}

func (suite *Suite) TestNewTimePeriod() {
	timezone, err := time.LoadLocation("Australia/Brisbane")
	suite.Require().NoError(err)

	for _, tt := range []struct {
		from     time.Time
		to       time.Time
		expected TimePeriod
	}{
		// when it's already normalized
		{
			time.Date(0, 0, 0, 15, 04, 05, 06, time.UTC),
			time.Date(0, 0, 0, 16, 04, 05, 06, time.UTC),
			TimePeriod{
				From: time.Date(0, 0, 0, 15, 04, 05, 06, time.UTC),
				To:   time.Date(0, 0, 0, 16, 04, 05, 06, time.UTC),
			},
		},
		// it normalizes to very first day
		{
			time.Date(1869, 9, 24, 15, 04, 05, 06, time.UTC),
			time.Date(1869, 9, 24, 16, 04, 05, 06, time.UTC),
			TimePeriod{
				From: time.Date(0, 0, 0, 15, 04, 05, 06, time.UTC),
				To:   time.Date(0, 0, 0, 16, 04, 05, 06, time.UTC),
			},
		},
		// it ignores the timezone
		{
			time.Date(1869, 9, 24, 15, 04, 05, 06, timezone),
			time.Date(1869, 9, 24, 16, 04, 05, 06, timezone),
			TimePeriod{
				From: time.Date(0, 0, 0, 15, 04, 05, 06, time.UTC),
				To:   time.Date(0, 0, 0, 16, 04, 05, 06, time.UTC),
			},
		},
	} {
		suite.Equal(tt.expected, NewTimePeriod(tt.from, tt.to))
	}
}

func (suite *Suite) TestTimePeriodIncludes() {
	atTheMoment := NewTimePeriod(
		time.Now().Add(-1*time.Minute),
		time.Now().Add(+1*time.Minute),
	)
	midnight := NewTimePeriod(
		time.Date(1869, 9, 23, 23, 00, 00, 00, time.UTC),
		time.Date(1869, 9, 24, 01, 00, 00, 00, time.UTC),
	)
	now := time.Now()

	for _, tt := range []struct {
		pointInTime time.Time
		timeOfDay   TimePeriod
		expected    bool
	}{
		// it's included
		{
			now,
			atTheMoment,
			true,
		},
		// it's one day before
		{
			now.Add(-24 * time.Hour),
			atTheMoment,
			true,
		},
		// it's one day after
		{
			now.Add(+24 * time.Hour),
			atTheMoment,
			true,
		},
		// it's just before
		{
			now.Add(-2 * time.Minute),
			atTheMoment,
			false,
		},
		// it's just after
		{
			now.Add(+2 * time.Minute),
			atTheMoment,
			false,
		},
		// it's slightly inside before day switch
		{
			time.Date(1869, 9, 23, 23, 30, 00, 00, time.UTC),
			midnight,
			true,
		},
		// it's slightly inside after day switch
		{
			time.Date(1869, 9, 24, 00, 30, 00, 00, time.UTC),
			midnight,
			true,
		},
		// it's just before
		{
			time.Date(1869, 9, 23, 22, 30, 00, 00, time.UTC),
			midnight,
			false,
		},
		// it's just after
		{
			time.Date(1869, 9, 24, 01, 30, 00, 00, time.UTC),
			midnight,
			false,
		},
		// it's exactly matching a point in time
		{
			now,
			TimePeriod{From: TimeOfDay(now), To: TimeOfDay(now)},
			true,
		},
		// it's right after exactly matching a point in time
		{
			now.Add(+1 * time.Second),
			TimePeriod{From: TimeOfDay(now), To: TimeOfDay(now)},
			false,
		},
		// it's right before exactly matching a point in time
		{
			now.Add(-1 * time.Second),
			TimePeriod{From: TimeOfDay(now), To: TimeOfDay(now)},
			false,
		},
	} {
		suite.Equal(tt.expected, tt.timeOfDay.Includes(tt.pointInTime))
	}
}

func (suite *Suite) TestTimePeriodString() {
	for _, tt := range []struct {
		given    TimePeriod
		expected string
	}{
		// empty time period
		{
			TimePeriod{},
			"00:00-00:00",
		},
		// simple time period
		{
			TimePeriod{
				From: time.Date(0, 0, 0, 8, 0, 0, 0, time.UTC),
				To:   time.Date(0, 0, 0, 16, 0, 0, 0, time.UTC),
			},
			"08:00-16:00",
		},
		// time period across days
		{
			TimePeriod{
				From: time.Date(0, 0, 0, 16, 0, 0, 0, time.UTC),
				To:   time.Date(0, 0, 0, 8, 0, 0, 0, time.UTC),
			},
			"16:00-08:00",
		},
	} {
		suite.Equal(tt.expected, tt.given.String())
	}
}

func (suite *Suite) TestTimeOfDay() {
	timezone, err := time.LoadLocation("Australia/Brisbane")
	suite.Require().NoError(err)

	for _, tt := range []struct {
		pointInTime time.Time
		expected    time.Time
	}{
		// strips of any day information
		{
			time.Date(1869, 9, 24, 15, 04, 05, 06, time.UTC),
			time.Date(0, 0, 0, 15, 04, 05, 06, time.UTC),
		},
		// it normalizes to UTC timezone
		{
			time.Date(1869, 9, 24, 15, 04, 05, 06, timezone),
			time.Date(0, 0, 0, 15, 04, 05, 06, time.UTC),
		},
	} {
		suite.Equal(tt.expected, TimeOfDay(tt.pointInTime))
	}
}

func (suite *Suite) TestParseWeekdays() {
	for _, tt := range []struct {
		given    string
		expected []time.Weekday
	}{
		// empty string
		{
			"",
			[]time.Weekday{},
		},
		// single weekday
		{
			"sat",
			[]time.Weekday{time.Saturday},
		},
		// multiple weekdays
		{
			"sat,sun",
			[]time.Weekday{time.Saturday, time.Sunday},
		},
		// case-insensitive
		{
			"SaT,SUn",
			[]time.Weekday{time.Saturday, time.Sunday},
		},
		// ignore whitespace
		{
			" sat , sun ",
			[]time.Weekday{time.Saturday, time.Sunday},
		},
		// ignore unknown weekdays
		{
			"sat,unknown,sun",
			[]time.Weekday{time.Saturday, time.Sunday},
		},
		// deal with all kinds at the same time
		{
			"Fri, sat ,,,,  ,foobar,tue",
			[]time.Weekday{time.Friday, time.Saturday, time.Tuesday},
		},
	} {
		suite.Equal(tt.expected, ParseWeekdays(tt.given))
	}
}

func (suite *Suite) TestParseTimePeriods() {
	for _, tt := range []struct {
		given    string
		expected []TimePeriod
	}{
		// empty time period string
		{
			"",
			[]TimePeriod{},
		},
		// single range string
		{
			"08:00-16:00",
			[]TimePeriod{
				{
					From: time.Date(0, 0, 0, 8, 0, 0, 0, time.UTC),
					To:   time.Date(0, 0, 0, 16, 0, 0, 0, time.UTC),
				},
			},
		},
		// multiple ranges string
		{
			"08:00-16:00,20:00-22:00",
			[]TimePeriod{
				{
					From: time.Date(0, 0, 0, 8, 0, 0, 0, time.UTC),
					To:   time.Date(0, 0, 0, 16, 0, 0, 0, time.UTC),
				},
				{
					From: time.Date(0, 0, 0, 20, 0, 0, 0, time.UTC),
					To:   time.Date(0, 0, 0, 22, 0, 0, 0, time.UTC),
				},
			},
		},
		// string containing whitespace
		{
			" 08:00 - 16:00 ,, , 20:00 - 22:00 ",
			[]TimePeriod{
				{
					From: time.Date(0, 0, 0, 8, 0, 0, 0, time.UTC),
					To:   time.Date(0, 0, 0, 16, 0, 0, 0, time.UTC),
				},
				{
					From: time.Date(0, 0, 0, 20, 0, 0, 0, time.UTC),
					To:   time.Date(0, 0, 0, 22, 0, 0, 0, time.UTC),
				},
			},
		},
	} {
		timePeriods, err := ParseTimePeriods(tt.given)
		suite.Require().NoError(err)

		suite.Equal(tt.expected, timePeriods)
	}
}

func (suite *Suite) TestParseDates() {
	for _, tt := range []struct {
		given    string
		expected []time.Time
	}{
		// empty string
		{
			"",
			[]time.Time{},
		},
		// single date
		{
			"Apr 1",
			[]time.Time{
				time.Date(0, 4, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		// single date leaving out the space
		{
			"Apr1",
			[]time.Time{
				time.Date(0, 4, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		// multiple dates
		{
			"Apr 1,Dec 24",
			[]time.Time{
				time.Date(0, 4, 1, 0, 0, 0, 0, time.UTC),
				time.Date(0, 12, 24, 0, 0, 0, 0, time.UTC),
			},
		},
		// case-insensitive
		{
			"apr 1,dEc 24",
			[]time.Time{
				time.Date(0, 4, 1, 0, 0, 0, 0, time.UTC),
				time.Date(0, 12, 24, 0, 0, 0, 0, time.UTC),
			},
		},
		// ignore whitespace
		{
			" apr 1 , dec 24 ",
			[]time.Time{
				time.Date(0, 4, 1, 0, 0, 0, 0, time.UTC),
				time.Date(0, 12, 24, 0, 0, 0, 0, time.UTC),
			},
		},
		// deal with all kinds at the same time
		{
			",Apr 1, dEc 24 ,,,,  ,jun08,,",
			[]time.Time{
				time.Date(0, 4, 1, 0, 0, 0, 0, time.UTC),
				time.Date(0, 12, 24, 0, 0, 0, 0, time.UTC),
				time.Date(0, 6, 8, 0, 0, 0, 0, time.UTC),
			},
		},
	} {
		days, err := ParseDays(tt.given)
		suite.Require().NoError(err)

		suite.Equal(tt.expected, days)
	}
}

func (suite *Suite) TestParseFrequency() {
	interval := 10 * time.Minute

	for _, tt := range []struct {
		given          string
		expectedApprox float64
	}{
		{
			"1 / hour",
			0.166666667,
		}, {
			"1 / minute",
			10.0,
		}, {
			"2.5 / hour",
			0.416666667,
		}, {
			"60 / day",
			0.416666667,
		},
	} {
		result, err := ParseFrequency(tt.given, interval)
		suite.Require().NoError(err)

		suite.Assert().InDelta(tt.expectedApprox, result, 0.001)
	}
}

func (suite *Suite) TestFormatDays() {
	for _, tt := range []struct {
		given    []time.Time
		expected []string
	}{
		{
			[]time.Time{
				time.Date(1869, 9, 24, 15, 04, 05, 06, time.UTC),
			},
			[]string{"Sep24"},
		}, {
			[]time.Time{
				time.Date(1869, 9, 24, 15, 04, 05, 06, time.UTC),
				time.Date(0, 4, 1, 0, 0, 0, 0, time.UTC),
			},
			[]string{"Sep24", "Apr 1"},
		},
	} {
		suite.Equal(tt.expected, FormatDays(tt.given))
	}
}

func (suite *Suite) TestNewPod() {
	pod := NewPodBuilder("namespace", "name").
		WithPhase("phase").Build()

	suite.Equal("v1", pod.APIVersion)
	suite.Equal("Pod", pod.Kind)
	suite.Equal("namespace", pod.Namespace)
	suite.Equal("name", pod.Name)
	suite.Equal("name", pod.Labels["app"])
	suite.Equal("name", pod.Annotations["chaos"])
	suite.Equal("/api/v1/namespaces/namespace/pods/name", pod.SelfLink)
	suite.EqualValues("phase", pod.Status.Phase)
}

func (suite *Suite) TestNewNamespace() {
	namespace := NewNamespace("name")

	suite.Equal("name", namespace.Name)
	suite.Equal("name", namespace.Labels["env"])
}

func (suite *Suite) TestRandomPodSublice() {
	pods := []v1.Pod{
		NewPodBuilder("default", "foo").Build(),
		NewPodBuilder("testing", "bar").Build(),
		NewPodBuilder("test", "baz").Build(),
	}

	for _, tt := range []struct {
		name     string
		in       []v1.Pod
		count    int
		expected int
	}{
		{"max kill = len(pods)", pods, 3, 3},
		{"empyt pod list should return empty subslice", []v1.Pod{}, 3, 0},
		{"maxKill > len(pods)", pods[0:1], 3, 1},
		{"maxKill = 0 ", pods, 0, 0},
	} {
		results := RandomPodSubSlice(tt.in, tt.count)
		suite.Assert().Equal(len(results), tt.expected, tt.name)
	}
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

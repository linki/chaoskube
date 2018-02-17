package util

import (
	"fmt"
	"testing"
	"time"
)

func TestNewTimePeriod(t *testing.T) {
	timezone, err := time.LoadLocation("Australia/Brisbane")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
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
		got := NewTimePeriod(tc.from, tc.to)
		if tc.expected.From != got.From {
			t.Fatalf("expected %v, got %v", tc.expected, got)
		}
		if tc.expected.To != got.To {
			t.Fatalf("expected %v, got %v", tc.expected, got)
		}
	}
}

func TestTimePeriodIncludes(t *testing.T) {
	atTheMoment := NewTimePeriod(time.Now().Add(-1*time.Minute), time.Now().Add(+1*time.Minute))

	midnight := NewTimePeriod(
		time.Date(1869, 9, 23, 23, 00, 00, 00, time.UTC),
		time.Date(1869, 9, 24, 01, 00, 00, 00, time.UTC),
	)

	now := time.Now()

	for _, tc := range []struct {
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
		got := tc.timeOfDay.Includes(tc.pointInTime)
		if tc.expected != got {
			t.Errorf("expected %v, got %v", tc.expected, got)
		}
	}
}

func TestTimePeriodString(t *testing.T) {
	for _, tc := range []struct {
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
		got := fmt.Sprintf("%s", tc.given)
		if tc.expected != got {
			t.Fatalf("expected %s, got %s", tc.expected, got)
		}
	}
}

func TestTimeOfDay(t *testing.T) {
	timezone, err := time.LoadLocation("Australia/Brisbane")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
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
		got := TimeOfDay(tc.pointInTime)
		if tc.expected != got {
			t.Fatalf("expected %v, got %v", tc.expected, got)
		}
	}
}

func TestParseWeekdays(t *testing.T) {
	for _, tc := range []struct {
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
		got := ParseWeekdays(tc.given)

		if len(tc.expected) != len(got) {
			t.Fatalf("expected %d, got %d", len(tc.expected), len(got))
		}

		for i := range tc.expected {
			if tc.expected[i] != got[i] {
				t.Errorf("expected %v, got %v", tc.expected[i], got[i])
			}
		}
	}
}

func TestParseTimePeriods(t *testing.T) {
	for _, tc := range []struct {
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
		got, err := ParseTimePeriods(tc.given)
		if err != nil {
			t.Fatal(err)
		}

		if len(tc.expected) != len(got) {
			t.Fatalf("expected %d, got %d", len(tc.expected), len(got))
		}

		for i := range tc.expected {
			if tc.expected[i] != got[i] {
				t.Errorf("expected %v, got %v", tc.expected[i], got[i])
			}
		}
	}
}

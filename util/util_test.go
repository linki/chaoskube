package util

import (
	"testing"
	"time"
)

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

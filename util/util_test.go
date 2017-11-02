package util

import (
	"testing"
	"time"
)

func TestStringToTime(t *testing.T) {
	nine, err := stringToTime("9:00")
	if err != nil {
		t.Fatal("stringToTime errored")
	}
	if nine.Hour() != 9 {
		t.Fatal("stringToTime failed to parse hour")
	}
	if nine.Minute() != 0 {
		t.Fatal("stringToTime failed to parse minutes")
	}
	_, err = stringToTime("9:00:00")
	if err == nil {
		t.Fatal("stringToTime should have failed")
	}
}

func TestStartAndEndTime(t *testing.T) {
	t1, t2, err := startAndEndTime("09:00", "17:00")
	if err != nil {
		t.Fatal("startAndEndTime errored")
	}
	if t1.Hour() != 9 {
		t.Fatal("startAndEndTime didn't parse time correctly")
	}
	if t2.Hour() != 17 {
		t.Fatal("startAndEndTime didn't parse time correctly")
	}
	if t1.After(t2) {
		t.Fatal("startAndEndTime didn't return the right times")
	}
	y_now, m_now, d_now := time.Now().Date()
	y_1, m_1, d_1 := t1.Date()
	y_2, m_2, d_2 := t2.Date()
	if y_now != y_1 || y_1 != y_2 {
		t.Fatal("startAndEndTime years are wrong", y_now, y_1, y_2)
	}
	if m_now != m_1 || m_1 != m_2 {
		t.Fatal("startAndEndTime months are wrong", m_now, m_1, m_2)
	}
	if d_now != d_1 || d_1 != d_2 {
		t.Fatal("startAndEndTime days are wrong", d_now, d_1, d_2)
	}
}

func TestShouldRunNow(t *testing.T) {
	y_now, m_now, d_now := time.Now().Date()

	// within the window it should run
	timeNow = func() time.Time { return time.Date(y_now, m_now, d_now, 11, 30, 0, 0, time.UTC) }
	if !ShouldRunNow(false, "9:00", "17:00") {
		t.Fatal("ShouldRunNow for 11:30 returned false")
	}

	// outside the window it should run
	timeNow = func() time.Time { return time.Date(y_now, m_now, d_now, 19, 30, 0, 0, time.UTC) }
	if ShouldRunNow(false, "9:00", "17:00") {
		t.Fatal("ShouldRunNow for 19:30 returned true")
	}

	// during a weekend, excludeWeekends = true, date is a this is a Sunday
	timeNow = func() time.Time { return time.Date(2017, 12, 31, 11, 30, 0, 0, time.UTC) }
	if ShouldRunNow(true, "9:00", "17:00") {
		t.Fatal("ShouldRunNow for excludeWeekends, but within the time window returned false")
	}

	// always run, but exclude the weekend
	if ShouldRunNow(true, "0:00", "0:00") {
		t.Fatal("ShouldRunNow for excludeWeekends returned true")
	}

	// always run and include the weekend
	if !ShouldRunNow(false, "0:00", "0:00") {
		t.Fatal("ShouldRunNow for excludeWeekends returned false")
	}
}

func TestParseLabel(t *testing.T) {
	labels := map[string]map[string]int{
		"1.hour": {"rate": 1, "span": 60},
		"2.day":  {"rate": 2, "span": 1440},
		"3.week": {"rate": 3, "span": 10080},
	}
	for k, v := range labels {
		rate, span, err := parseLabel(k)
		if err != nil {
			t.Fatal("parseLabel errored")
		}
		if rate != v["rate"] {
			t.Fatalf("parseLabel returned wrong rate want: %v got: %v", v["rate"], rate)
		}
		if span != v["span"] {
			t.Fatalf("parseLabel returned wrong span want: %v got: %v", v["span"], span)
		}
	}
}

func TestGetOdds(t *testing.T) {
	schedules := map[string]float64{"1.hour": 1.0 / float64(60), "2.day": 2.0 / float64(60*24), "3.week": 3.0 / float64(60*24*7)}
	percentage := 0.5
	for schedule, initial_odd := range schedules {
		p := NewPod("default", "foo", schedule)
		intervalls := []time.Duration{time.Minute * 1, time.Minute * 5, time.Minute * 10, time.Minute * 60}
		for _, interval := range intervalls {
			odd := getOdds(p, interval, 0.5)
			target_odd := int(initial_odd * interval.Minutes() * 100)
			conv_odd := int(odd * 100)
			if conv_odd != target_odd {
				t.Fatalf("getOdds returned wrong odd want: %v got: %v, schedule: %v, interval: %v, percentage: %v",
					target_odd, odd, schedule, interval, percentage)
			}
		}
	}
	p := NewPod("default", "foo")
	interval := 10 * time.Minute
	odd := getOdds(p, interval, 0.5)
	if odd != 0.5 {
		t.Fatalf("getOdds returned wrong odd want: %v got: %v, schedule: %v, interval: %v, percentage: %v",
			percentage, odd, "", interval, percentage)
	}
}

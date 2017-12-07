package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

const dhmsLayout = "2006-01-02 15:04:05"

func Test_handleOfftimeConfig_01(t *testing.T) {
	descr := "Zero value locationStr"
	var locationStr string
	var offDaysStr string
	var chaosHrsStr string
	var holidaysStr string
	_, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err == nil {
		t.Errorf("%s: Expected err", descr)
	} else {
		substr := "required"
		msg := err.Error()
		if !strings.Contains(msg, substr) {
			t.Errorf(`%s: Expected "%s" in error message. Got "%s"`, descr, substr, msg)
		}
	}
}

func Test_handleOfftimeConfig_02(t *testing.T) {
	descr := "Unparsable locationStr"
	var locationStr = "PDT"
	var offDaysStr string
	var chaosHrsStr string
	var holidaysStr string
	_, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err == nil {
		t.Errorf("%s: Expected err", descr)
	} else {
		substr := "tz database"
		msg := err.Error()
		if !strings.Contains(msg, substr) {
			t.Errorf(`%s: Expected "%s" in error message. Got "%s"`, descr, substr, msg)
		}
	}
}

func Test_handleOfftimeConfig_03(t *testing.T) {
	descr := "Good timezone locationStr"
	var locationStr = "America/Los_Angeles"
	var offDaysStr string
	var chaosHrsStr string
	var holidaysStr string
	offcfg, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf("%s: Unexpected err: %s", descr, err.Error())
	} else {
		actual := offcfg.location.String()
		if actual != locationStr {
			t.Errorf(`"%s: offcfg.location: Expected "%s", got "%s"`, descr, locationStr, actual)
		}
	}
}

func Test_handleOfftimeConfig_04(t *testing.T) {
	descr := `Unset/Empty offDays`
	var locationStr = "UTC"
	var offDaysStr string
	var chaosHrsStr string
	var holidaysStr string
	offcfg, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf("%s: Unexpected err: %s", descr, err.Error())
	} else {
		expected := []time.Weekday{time.Saturday, time.Sunday}
		actual := offcfg.offDays
		if !wkdSlicesEquivalent(expected, actual) {
			t.Errorf(`%s: "offcfg.offDays: Expected "%#v", got "%#v"`, descr, expected, actual)
		}
	}
	// Test empty string for offDaysStr
	offDaysStr = ""
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf("%s: Unexpected err: %s", descr, err.Error())
	} else {
		expected := []time.Weekday{time.Saturday, time.Sunday}
		actual := offcfg.offDays
		if !wkdSlicesEquivalent(expected, actual) {
			t.Errorf(`%s: "offcfg.offDays: Expected "%#v", got "%#v"`, descr, expected, actual)
		}
	}
}

func Test_handleOfftimeConfig_05(t *testing.T) {
	descr := `offDays = "none"`
	var locationStr = "UTC"
	var offDaysStr = "none"
	var chaosHrsStr string
	var holidaysStr string
	offcfg, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf("%s: Unexpected err: %s", descr, err.Error())
	} else {
		var expected []time.Weekday
		actual := offcfg.offDays
		if !wkdSlicesEquivalent(expected, actual) {
			t.Errorf(`%s: "offcfg.offDays: Expected "%#v", got "%#v"`, descr, expected, actual)
		}
	}
	// Test empty string for offDaysStr
	offDaysStr = ""
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf("%s: Unexpected err: %s", descr, err.Error())
	} else {
		expected := []time.Weekday{time.Saturday, time.Sunday}
		actual := offcfg.offDays
		if !wkdSlicesEquivalent(expected, actual) {
			t.Errorf(`%s: "offcfg.offDays: Expected "%#v", got "%#v"`, descr, expected, actual)
		}
	}
}

func Test_handleOfftimeConfig_06(t *testing.T) {
	descr := `Bad offDays`
	var locationStr = "UTC"
	var offDaysStr = "Saturday, Zontag"
	var chaosHrsStr string
	var holidaysStr string
	_, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err == nil {
		t.Errorf("%s: Expected err", descr)
	} else {
		substr := "unrecognized"
		msg := err.Error()
		if !strings.Contains(msg, substr) {
			t.Errorf(`%s: Expected "%s" in error message. Got "%s"`, descr, substr, msg)
		}
	}
}

func Test_handleOfftimeConfig_07(t *testing.T) {
	descr := `Three off days`
	var locationStr = "UTC"
	var offDaysStr = "Thursday,   Monday,Friday" // various whitespace too
	var chaosHrsStr string
	var holidaysStr string
	offcfg, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf("%s: Unexpected err: %s", descr, err.Error())
	} else {
		expected := []time.Weekday{time.Monday, time.Thursday, time.Friday}
		actual := offcfg.offDays
		if !wkdSlicesEquivalent(expected, actual) {
			t.Errorf(`%s: "offcfg.offDays: Expected "%#v", got "%#v"`, descr, expected, actual)
		}
	}
}

func Test_handleOfftimeConfig_08(t *testing.T) {
	descr := `Empty chaos hours`
	var locationStr = "UTC"
	var offDaysStr string
	var chaosHrsStr string
	var holidaysStr string
	offcfg, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf("%s: Unexpected err: %s", descr, err.Error())
	} else {
		loc, _ := time.LoadLocation(locationStr)
		expected := offtimeCfg{
			enabled:       true,
			location:      loc,
			offDays:       []time.Weekday{time.Saturday, time.Sunday},
			chaosStartHr:  defaultStartHr,
			chaosStartMin: defaultStartMin,
			chaosEndHr:    defaultEndHr,
			chaosEndMin:   defaultEndMin,
		}
		actual := offcfg
		// Just compare the start/end times
		if !(actual.chaosStartHr == expected.chaosStartHr &&
			actual.chaosStartMin == expected.chaosStartMin &&
			actual.chaosEndHr == expected.chaosEndHr &&
			actual.chaosEndMin == expected.chaosEndMin) {
			t.Errorf(`%s: Expected "%#v", got "%#v"`, descr, expected, actual)
		}
	}
}

func Test_handleOfftimeConfig_09(t *testing.T) {
	descr := `chaos hours with leading zeros should be OK`
	var locationStr = "UTC"
	var offDaysStr string
	var chaosHrsStr = "start: 00:01, end: 03:00"
	var holidaysStr string
	offcfg, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf("%s: Unexpected err: %s", descr, err.Error())
	} else {
		loc, _ := time.LoadLocation(locationStr)
		expected := offtimeCfg{
			enabled:       true,
			location:      loc,
			offDays:       []time.Weekday{time.Saturday, time.Sunday},
			chaosStartHr:  0,
			chaosStartMin: 1,
			chaosEndHr:    3,
			chaosEndMin:   0,
		}
		actual := offcfg
		// Just compare the start/end times
		if !(actual.chaosStartHr == expected.chaosStartHr &&
			actual.chaosStartMin == expected.chaosStartMin &&
			actual.chaosEndHr == expected.chaosEndHr &&
			actual.chaosEndMin == expected.chaosEndMin) {
			t.Errorf(`%s: Expected "%#v", got "%#v"`, descr, expected, actual)
		}
	}
}

func Test_handleOfftimeConfig_10(t *testing.T) {
	descr := `chaos hours may not span midnight`
	var locationStr = "UTC"
	var offDaysStr string
	var chaosHrsStr = "start: 11:59, end: 00:00"
	var holidaysStr string
	_, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err == nil {
		t.Errorf("%s: Expected error.", descr)
	} else {
		substr := "midnight"
		msg := err.Error()
		if !strings.Contains(msg, substr) {
			t.Errorf(`%s: Expected "%s" in error message. Got "%s"`, descr, substr, msg)
		}
	}
}

func Test_handleOfftimeConfig_11(t *testing.T) {
	descr := `chaos hours - bad number`
	var locationStr = "UTC"
	var offDaysStr string
	var chaosHrsStr = "start: 1O:59, end: 13:00" // capital O (letter) not 0
	var holidaysStr string
	_, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err == nil {
		t.Errorf("%s: Expected error.", descr)
	} else {
		substr := "could not parse"
		msg := err.Error()
		if !strings.Contains(msg, substr) {
			t.Errorf(`%s: Expected "%s" in error message. Got "%s"`, descr, substr, msg)
		}
	}
}

func Test_handleOfftimeConfig_12(t *testing.T) {
	descr := `Empty holidays should be OK`
	var locationStr = "UTC"
	var offDaysStr string
	var chaosHrsStr string
	var holidaysStr string
	offcfg, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf("%s: Unexpected err: %s", descr, err.Error())
	} else {
		actual := len(offcfg.holidays)
		if actual != 0 {
			t.Errorf(`%s: Expected 0 holidays, got %d`, descr, actual)
		}
	}
}

func Test_handleOfftimeConfig_13(t *testing.T) {
	descr := `One holiday: New Years Day in Los Angeles`
	var locationStr = "America/Los_Angeles"
	var offDaysStr string
	var chaosHrsStr string
	var holidaysStr = "2016-01-01"
	offcfg, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf("%s: Unexpected err: %s", descr, err.Error())
	} else {
		actual := len(offcfg.holidays)
		if actual != 1 {
			t.Errorf(`%s: Expected 1 holiday, got %d`, descr, actual)
		}
		dateStr := offcfg.holidays[0].Format(time.RFC822Z)
		expected := "01 Jan 16 00:00 -0800"
		if dateStr != expected {
			t.Errorf(`%s: Expected "%s", got "%s"`, descr, expected, dateStr)
		}
	}
}

func Test_handleOfftimeConfig_14(t *testing.T) {
	descr := `Two holidays in Los Angeles`
	var locationStr = "America/Los_Angeles"
	var offDaysStr string
	var chaosHrsStr string
	var holidaysStr = "2016-01-01, 2014-12-25"
	offcfg, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf("%s: Unexpected err: %s", descr, err.Error())
	} else {
		actual := len(offcfg.holidays)
		if actual != 2 {
			t.Errorf(`%s: Expected 2 holidays, got %d`, descr, actual)
		}

		dateStr := offcfg.holidays[0].Format(time.RFC822Z)
		expected := "01 Jan 16 00:00 -0800"
		if dateStr != expected {
			t.Errorf(`%s: Expected "%s", got "%s"`, descr, expected, dateStr)
		}

		dateStr = offcfg.holidays[1].Format(time.RFC822Z)
		expected = "25 Dec 14 00:00 -0800"
		if dateStr != expected {
			t.Errorf(`%s: Expected "%s", got "%s"`, descr, expected, dateStr)
		}
	}
}

func Test_handleOfftimeConfig_15(t *testing.T) {
	descr := `Bad holiday string`
	var locationStr = "UTC"
	var offDaysStr string
	var chaosHrsStr string
	var holidaysStr = "1/1/2016"
	_, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err == nil {
		t.Errorf("%s: Expected err", descr)
	} else {
		substr := "invalid date format"
		msg := err.Error()
		if !strings.Contains(msg, substr) {
			t.Errorf(`%s: Expected "%s" in error message. Got "%s"`, descr, substr, msg)
		}
	}
}

func Test_timeToSuspend_01(t *testing.T) {
	descr := "Suspend - respect stated timezone"
	var locationStr = "America/Los_Angeles"
	var offDaysStr = "none"
	var chaosHrsStr string
	var holidaysStr = "2016-01-01"
	offcfg, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST`, descr)
	}
	utc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST`, descr)
	}

	// Set testTime to same day as holiday, but in UTC
	testTime, err := time.ParseInLocation(iso8601, holidaysStr, utc)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST`, descr)
	}

	// Date is correct, current time is UTC, so Los Angeles is still day before
	actual := timeToSuspend(testTime, *offcfg)
	expected := false
	if actual != expected {
		t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
	}

	// Now use first instant of holiday in Los Angeles TZ
	testTime, err = time.ParseInLocation(iso8601, holidaysStr, offcfg.location)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST`, descr)
	}
	actual = timeToSuspend(testTime, *offcfg)
	expected = true
	if actual != expected {
		t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
	}
}

func Test_timeToSuspend_02(t *testing.T) {
	descr := "Suspend - respect off days"
	var locationStr = "America/Los_Angeles"
	var offDaysStr = "none"
	var chaosHrsStr string
	var holidaysStr string
	offcfg, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST`, descr)
	}

	var testDayStr = "2016-01-05 13:00:00" // this is a Tuesday

	testTime, err := time.ParseInLocation(dhmsLayout, testDayStr, offcfg.location)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST`, descr)
	}

	// No holidays, no off days, default chaosHours
	actual := timeToSuspend(testTime, *offcfg)
	expected := false
	if actual != expected {
		t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
	}

	// Set offDays to Wednesday
	offDaysStr = "Wednesday"
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST`, descr)
	}
	actual = timeToSuspend(testTime, *offcfg)
	expected = false
	if actual != expected {
		t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
	}

	// Set offDays to Tuesday
	offDaysStr = "Tuesday"
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST`, descr)
	}
	actual = timeToSuspend(testTime, *offcfg)
	expected = true
	if actual != expected {
		t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
	}
}

func Test_timeToSuspend_03(t *testing.T) {
	descr := "Suspend - respect chaos hrs"
	var locationStr = "America/Los_Angeles"
	var offDaysStr = "none"
	var chaosHrsStr string // use defaults
	var holidaysStr string
	offcfg, err := handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST`, descr)
	}

	var testTimeStr = fmt.Sprintf("2016-01-05 %d:%d:00", offcfg.chaosStartHr, offcfg.chaosStartMin)
	testTime, err := time.ParseInLocation(dhmsLayout, testTimeStr, offcfg.location)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST`, descr)
	}

	// No holidays, no off days, default chaosHours - testTime is start of chaos period
	actual := timeToSuspend(testTime, *offcfg)
	expected := false
	if actual != expected {
		t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
	}

	// Subtract to one sec before chaos period
	minusOneSec, err := time.ParseDuration("-1s")
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST`, descr)
	}
	testTime = testTime.Add(minusOneSec)
	actual = timeToSuspend(testTime, *offcfg)
	expected = true
	if actual != expected {
		t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
	}

	// Set to end of chaos period
	testTimeStr = fmt.Sprintf("2016-01-05 %d:%d:00", offcfg.chaosEndHr, offcfg.chaosEndMin)
	testTime, err = time.ParseInLocation(dhmsLayout, testTimeStr, offcfg.location)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST`, descr)
	}
	actual = timeToSuspend(testTime, *offcfg)
	expected = true
	if actual != expected {
		t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
	}
}

// -----------------------------------------------------------------------------

// wkdSlicesEquivalent compares two slices of time.Weekday but ignores order.
func wkdSlicesEquivalent(a []time.Weekday, b []time.Weekday) bool {
	if len(a) != len(b) {
		return false
	}
	for _, aw := range a {
		found := false
		for _, bw := range b {
			if bw == aw {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

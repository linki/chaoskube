package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

const dhmsLayout = "2006-01-02 15:04:05"

func Test_handleOfftimeConfig_location(t *testing.T) {
	var err error
	var descr string
	var locationStr string
	var offDaysStr string
	var chaosHrsStr string
	var holidaysStr string
	var offcfg *offtimeCfg

	descr = "Zero value locationStr"
	locationStr = ""
	offDaysStr = ""
	chaosHrsStr = ""
	holidaysStr = ""
	_, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err == nil {
		t.Errorf("%s: Expected err", descr)
	} else {
		substr := "required"
		msg := err.Error()
		if !strings.Contains(msg, substr) {
			t.Errorf(`%s: Expected "%s" in error message. Got "%s"`, descr, substr, msg)
		}
	}

	// -------------------------------------------------------------------------

	descr = "Unparsable locationStr"
	locationStr = "PDT"
	offDaysStr = ""
	chaosHrsStr = ""
	holidaysStr = ""
	_, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err == nil {
		t.Errorf("%s: Expected err", descr)
	} else {
		substr := "tz database"
		msg := err.Error()
		if !strings.Contains(msg, substr) {
			t.Errorf(`%s: Expected "%s" in error message. Got "%s"`, descr, substr, msg)
		}
	}

	// -------------------------------------------------------------------------

	descr = "Good timezone locationStr"
	locationStr = "America/Los_Angeles"
	offDaysStr = ""
	chaosHrsStr = ""
	holidaysStr = ""
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf("%s: Unexpected err: %s", descr, err.Error())
	} else {
		actual := offcfg.location.String()
		if actual != locationStr {
			t.Errorf(`"%s: offcfg.location: Expected "%s", got "%s"`, descr, locationStr, actual)
		}
	}
}

func Test_handleOfftimeConfig_offDays(t *testing.T) {
	var err error
	var descr string
	var locationStr string
	var offDaysStr string
	var chaosHrsStr string
	var holidaysStr string
	var offcfg *offtimeCfg

	descr = `Empty offDays`
	locationStr = "UTC"
	offDaysStr = ""
	chaosHrsStr = ""
	holidaysStr = ""
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

	// -------------------------------------------------------------------------

	descr = `offDays = "none"`
	locationStr = "UTC"
	offDaysStr = "none"
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf("%s: Unexpected err: %s", descr, err.Error())
	} else {
		var expected []time.Weekday
		actual := offcfg.offDays
		if !wkdSlicesEquivalent(expected, actual) {
			t.Errorf(`%s: "offcfg.offDays: Expected "%#v", got "%#v"`, descr, expected, actual)
		}
	}

	// -------------------------------------------------------------------------

	descr = `Bad offDays`
	locationStr = "UTC"
	offDaysStr = "Saturday, Zontag"
	chaosHrsStr = ""
	holidaysStr = ""
	_, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err == nil {
		t.Errorf("%s: Expected err", descr)
	} else {
		substr := "unrecognized"
		msg := err.Error()
		if !strings.Contains(msg, substr) {
			t.Errorf(`%s: Expected "%s" in error message. Got "%s"`, descr, substr, msg)
		}
	}

	// -------------------------------------------------------------------------

	descr = `Three off days`
	locationStr = "UTC"
	offDaysStr = "Thursday,   Monday,Friday" // various whitespace too
	chaosHrsStr = ""
	holidaysStr = ""
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
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

func Test_handleOfftimeConfig_chaosHrs(t *testing.T) {
	var err error
	var descr string
	var locationStr string
	var offDaysStr string
	var chaosHrsStr string
	var holidaysStr string
	var offcfg *offtimeCfg

	descr = `Empty chaos hours`
	locationStr = "UTC"
	offDaysStr = ""
	chaosHrsStr = ""
	holidaysStr = ""
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
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

	// -------------------------------------------------------------------------

	descr = `chaos hours with leading zeros should be OK`
	locationStr = "UTC"
	offDaysStr = ""
	chaosHrsStr = "start: 00:01, end: 03:00"
	holidaysStr = ""
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
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

	// -------------------------------------------------------------------------

	descr = `chaos hours may not span midnight`
	locationStr = "UTC"
	offDaysStr = ""
	chaosHrsStr = "start: 11:59, end: 00:00"
	holidaysStr = ""
	_, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err == nil {
		t.Errorf("%s: Expected error.", descr)
	} else {
		substr := "midnight"
		msg := err.Error()
		if !strings.Contains(msg, substr) {
			t.Errorf(`%s: Expected "%s" in error message. Got "%s"`, descr, substr, msg)
		}
	}

	// -------------------------------------------------------------------------

	descr = `chaos hours - bad number`
	locationStr = "UTC"
	offDaysStr = ""
	chaosHrsStr = "start: 1O:59, end: 13:00" // capital O (letter) not 0
	holidaysStr = ""
	_, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
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

func Test_handleOfftimeConfig_holidays(t *testing.T) {
	var err error
	var descr string
	var locationStr string
	var offDaysStr string
	var chaosHrsStr string
	var holidaysStr string
	var offcfg *offtimeCfg

	locationStr = "UTC"
	offDaysStr = ""
	chaosHrsStr = ""
	holidaysStr = ""
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf("%s: Unexpected err: %s", descr, err.Error())
	} else {
		actual := len(offcfg.holidays)
		if actual != 0 {
			t.Errorf(`%s: Expected 0 holidays, got %d`, descr, actual)
		}
	}

	// -------------------------------------------------------------------------

	descr = `One holiday: New Years Day in Los Angeles`
	locationStr = "America/Los_Angeles"
	offDaysStr = ""
	chaosHrsStr = ""
	holidaysStr = "2016-01-01"
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
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

	// -------------------------------------------------------------------------

	descr = `Two holidays in Los Angeles`
	locationStr = "America/Los_Angeles"
	offDaysStr = ""
	chaosHrsStr = ""
	holidaysStr = "2016-01-01, 2014-12-25"
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
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

	// -------------------------------------------------------------------------

	descr = `Bad holiday string`
	locationStr = "UTC"
	offDaysStr = ""
	chaosHrsStr = ""
	holidaysStr = "1/1/2016"
	_, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
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

func Test_timeToSuspend(t *testing.T) {
	var err error
	var descr string
	var locationStr string
	var offDaysStr string
	var chaosHrsStr string
	var holidaysStr string
	var offcfg *offtimeCfg
	var testTime time.Time

	descr = "Suspend - respect stated timezone"
	locationStr = "America/Los_Angeles"
	offDaysStr = "none"
	chaosHrsStr = ""
	holidaysStr = "2016-01-01"
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST - %v`, descr, err)
		return
	}
	utc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST - %v`, descr, err)
		return
	}

	// Set testTime to same day as holiday, but in UTC
	testTime, err = time.ParseInLocation(iso8601, holidaysStr, utc)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST - %v`, descr, err)
	} else {
		// Date is correct, current time is UTC, so Los Angeles is still day before
		actual := timeToSuspend(testTime, *offcfg)
		expected := false
		if actual != expected {
			t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
		}
	}

	// Now use first instant of holiday in Los Angeles TZ
	testTime, err = time.ParseInLocation(iso8601, holidaysStr, offcfg.location)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST - %v`, descr, err)
	} else {
		actual := timeToSuspend(testTime, *offcfg)
		expected := true
		if actual != expected {
			t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
		}
	}

	// -------------------------------------------------------------------------

	descr = "Suspend - respect off days"
	locationStr = "America/Los_Angeles"
	offDaysStr = "none"
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST - %v`, descr, err)
	}

	var testDayStr = "2016-01-05 13:00:00" // this is a Tuesday

	testTime, err = time.ParseInLocation(dhmsLayout, testDayStr, offcfg.location)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST - %v`, descr, err)
	} else {
		// No holidays, no off days, default chaosHours
		actual := timeToSuspend(testTime, *offcfg)
		expected := false
		if actual != expected {
			t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
		}
	}

	// Set offDays to Wednesday
	offDaysStr = "Wednesday"
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST - %v`, descr, err)
	} else {
		actual := timeToSuspend(testTime, *offcfg)
		expected := false
		if actual != expected {
			t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
		}
	}

	// Set offDays to Tuesday
	offDaysStr = "Tuesday"
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST - %v`, descr, err)
	} else {
		actual := timeToSuspend(testTime, *offcfg)
		expected := true
		if actual != expected {
			t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
		}
	}

	// -------------------------------------------------------------------------

	descr = "Suspend - respect chaos hrs"
	locationStr = "America/Los_Angeles"
	offDaysStr = "none"
	chaosHrsStr = "" // use defaults
	holidaysStr = ""
	offcfg, err = handleOfftimeConfig(true, locationStr, offDaysStr, chaosHrsStr, holidaysStr)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST - %v`, descr, err)
		return
	}

	var testTimeStr = fmt.Sprintf("2016-01-05 %d:%d:00", offcfg.chaosStartHr, offcfg.chaosStartMin)
	testTime, err = time.ParseInLocation(dhmsLayout, testTimeStr, offcfg.location)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST - %v`, descr, err)
	} else {
		// No holidays, no off days, default chaosHours - testTime is start of chaos period
		actual := timeToSuspend(testTime, *offcfg)
		expected := false
		if actual != expected {
			t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
		}
	}

	// Subtract to one sec before chaos period
	minusOneSec, err := time.ParseDuration("-1s")
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST - %v`, descr, err)
	} else {
		testTime = testTime.Add(minusOneSec)
		actual := timeToSuspend(testTime, *offcfg)
		expected := true
		if actual != expected {
			t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
		}
	}

	// Set to end of chaos period
	testTimeStr = fmt.Sprintf("2016-01-05 %d:%d:00", offcfg.chaosEndHr, offcfg.chaosEndMin)
	testTime, err = time.ParseInLocation(dhmsLayout, testTimeStr, offcfg.location)
	if err != nil {
		t.Errorf(`%s: ERROR IN TEST - %v`, descr, err)
	} else {
		actual := timeToSuspend(testTime, *offcfg)
		expected := true
		if actual != expected {
			t.Errorf(`%s: Got %v, expected %v`, descr, actual, expected)
		}
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

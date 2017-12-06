package util

import (
	"math/rand"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

var timeNow = timeNowFunc

func init() {
	rand.Seed(timeNow().Unix())
}

func timeNowFunc() time.Time {
	return time.Now()
}

// NewPod returns a new pod instance for testing purposes.
func NewPod(namespace, name string, schedule ...string) v1.Pod {
	labels := map[string]string{"app": name}
	if len(schedule) > 0 {
		labels["chaos.schedule"] = schedule[0]
	}
	return v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
			Annotations: map[string]string{
				"chaos": name,
			},
		},
	}
}

// takes a string containing a time (e.g. "23:42" and returns time object with that time today
func stringToTime(str string) (time.Time, error) {
	now := timeNow()
	year, month, day := now.Date()
	time, err := time.Parse("15:04", str)
	if err != nil {
		return now, err
	}
	return time.AddDate(year, int(month)-1, day-1), nil
}

// takes two strings containing a time (e.g. "09:00" and "17:00") and returns 2
// time objects so that the "runFrom" one is before the "runUntil" one unless
// it needs to be after because of situations like runfrom 17:00 to 05:00
func startAndEndTime(runFrom string, runUntil string) (time.Time, time.Time, error) {
	start, err := stringToTime(runFrom)
	if err != nil {
		return timeNow(), timeNow(), err
	}
	end, err := stringToTime(runUntil)
	if err != nil {
		return timeNow(), timeNow(), err
	}
	// start this day and end the next day and be after start which means end
	// will have to be moved to the next day
	if end.Before(start) && timeNow().After(start) {
		return start, end.AddDate(0, 0, 1), nil
	}
	return start, end, nil
}

// checks whether time.Now() is between runFrom and runUntil and whether it
// should run during the weekend
func ShouldRunNow(excludeWeekends bool, runFrom string, runUntil string) bool {
	now := timeNow()
	// Exclude weekends, sunday = day 0, saturday = day 6
	weekday := now.Weekday()
	if excludeWeekends && (weekday == 0 || weekday == 6) {
		return false
	}
	// no input was specified
	if runFrom == runUntil && runFrom == "0:00" {
		return true
	}
	start, end, err := startAndEndTime(runFrom, runUntil)
	if err != nil {
		log.Info("Converting times errored. No action will be taken.")
		return false
	}
	if now.After(start) && now.Before(end) {
		return true
	}
	return false
}

func parseLabel(label string) (rate int, span int, err error) {
	split := strings.Split(label, ".")
	if len(split) != 2 {
		return 0, 0, err
	}
	rate_str, span_str := split[0], split[1]
	rate, err = strconv.Atoi(rate_str)
	if err != nil {
		return 0, 0, err
	}
	switch span_str {
	case "hour":
		span = 60
	case "day":
		span = 60 * 24
	case "week":
		span = 60 * 24 * 7
	}
	return
}

func getOdds(p v1.Pod, interval time.Duration, percentage float64) float64 {
	labels := p.GetLabels()
	if labels["chaos.schedule"] == "" {
		return percentage
	}
	rate, span, err := parseLabel(labels["chaos.schedule"])
	if err != nil {
		log.Errorf("Error: %v from parsing %v's chaos.schedule, which is %s", err, p.Name, labels["chaos.schedule"])
		return 0.0
	}
	return (float64(rate) * interval.Minutes()) / float64(span)
}

func PodShouldDie(p v1.Pod, interval time.Duration, percentage float64) bool {
	odds := getOdds(p, interval, percentage)
	random := rand.Float64()
	return (random <= odds)
}

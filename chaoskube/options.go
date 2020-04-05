package chaoskube

import (
	"regexp"
	"time"

	"github.com/linki/chaoskube/notifier"
	"github.com/linki/chaoskube/terminator"
	"github.com/linki/chaoskube/util"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
)

type Option func(*Chaoskube)

func WithLabels(selector labels.Selector) Option {
	return func(c *Chaoskube) {
		c.Labels = selector
	}
}

func WithAnnotations(selector labels.Selector) Option {
	return func(c *Chaoskube) {
		c.Annotations = selector
	}
}

func WithNamespaces(selector labels.Selector) Option {
	return func(c *Chaoskube) {
		c.Namespaces = selector
	}
}

func WithNamespaceLabels(selector labels.Selector) Option {
	return func(c *Chaoskube) {
		c.NamespaceLabels = selector
	}
}

func WithIncludedPodNames(regexp *regexp.Regexp) Option {
	return func(c *Chaoskube) {
		c.IncludedPodNames = regexp
	}
}

func WithExcludedPodNames(regexp *regexp.Regexp) Option {
	return func(c *Chaoskube) {
		c.ExcludedPodNames = regexp
	}
}

func WithExcludedWeekdays(weekdays []time.Weekday) Option {
	return func(c *Chaoskube) {
		c.ExcludedWeekdays = weekdays
	}
}

func WithExcludedTimesOfDay(timesOfDay []util.TimePeriod) Option {
	return func(c *Chaoskube) {
		c.ExcludedTimesOfDay = timesOfDay
	}
}

func WithExcludedDaysOfYear(daysOfYear []time.Time) Option {
	return func(c *Chaoskube) {
		c.ExcludedDaysOfYear = daysOfYear
	}
}

func WithTimezone(timezone *time.Location) Option {
	return func(c *Chaoskube) {
		c.Timezone = timezone
	}
}

func WithMinimumAge(duration time.Duration) Option {
	return func(c *Chaoskube) {
		c.MinimumAge = duration
	}
}

func WithLogger(logger *logrus.Logger) Option {
	return func(c *Chaoskube) {
		c.Logger = logger
	}
}

func WithDryRun(dryRun bool) Option {
	return func(c *Chaoskube) {
		c.DryRun = dryRun
	}
}

func WithTerminator(terminator terminator.Terminator) Option {
	return func(c *Chaoskube) {
		c.Terminator = terminator
	}
}

func WithMaxKill(num int) Option {
	return func(c *Chaoskube) {
		c.MaxKill = num
	}
}

func WithNotifier(notifier notifier.Notifier) Option {
	return func(c *Chaoskube) {
		c.Notifier = notifier
	}
}

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// PodsDeletedTotal is the total number of deleted pods.
	PodsDeletedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "chaoskube",
		Name:      "pods_deleted_total",
		Help:      "The total number of pods deleted",
	}, []string{"namespace"})
	// IntervalsTotal is the total number of intervals, i.e. call to Run().
	IntervalsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "chaoskube",
		Name:      "intervals_total",
		Help:      "The total number of pod termination logic runs",
	})
	// ErrorsTotal is the total number of errors encountered while trying to terminate pods.
	ErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "chaoskube",
		Name:      "errors_total",
		Help:      "The total number of errors on terminate victim operation",
	})
	// TerminationDurationSeconds is a histogram over the time it took to terminate pods.
	TerminationDurationSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "chaoskube",
		Name:      "termination_duration_seconds",
		Help:      "The time it took a single pod termination to finish",
	})
)

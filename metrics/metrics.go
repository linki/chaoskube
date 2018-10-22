package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RunCounter is the run function executions counter
	IntervalsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "chaoskube",
		Name:      "intervals_total",
		Help:      "The total number of pod termination logic runs",
	})
	// ErrorCounter is the run function executions counter
	ErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "chaoskube",
		Name:      "errors_total",
		Help:      "The total number of errors on terminate victim operation",
	})
	// PodsDeletedCounter is the pods deleted counter
	PodsDeletedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "chaoskube",
		Name:      "pods_deleted_total",
		Help:      "The total number of pods deleted",
	})
	// errorCounter is the run function executions counter
	RequestDurationSeconds = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "chaoskube",
		Name:      "request_duration_seconds",
		Help:      "The time took single pod termination to finish",
	}, []string{"method"})
)

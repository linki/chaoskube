package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// PodsDeletedCounter is the pods deleted counter
	PodsDeletedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pods_deleted",
		Help: "The total number of pods deleted",
	})
	// RunCounter is the run function executions counter
	RunCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "run_counts",
		Help: "The total number of pod termination logic runs",
	})
	// ErrorCounter is the run function executions counter
	ErrorCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "termination_errors",
		Help: "The total number of errors on terminate victim operation",
	})
	// errorCounter is the run function executions counter
	TerminationHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "termination_time_seconds",
		Help: "The time took single pod termination to finish",
	})
)

package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	NumEvictions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "chaoskube",
			Name:      "pod_evictions_total",
			Help:      "Total number of Pod evictions",
		},
		[]string{"namespace"},
	)
)

func init() {
	prometheus.MustRegister(NumEvictions)
}

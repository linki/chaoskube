package chaoskube

import (
	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/client-go/pkg/api/v1"
)

var (
	// NumEvictions holds the number of successful pod terminations.
	NumEvictions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "chaoskube",
			Name:      "pod_evictions_total",
			Help:      "Total number of Pod evictions",
		},
		[]string{"pod_namespace"},
	)
)

// Instrumented represents an instance of Chaoskube that counts pod terminations.
type Instrumented struct {
	// parent Chaoskube
	Chaoskube
}

func init() {
	prometheus.MustRegister(NumEvictions)
}

// NewInstrumented returns a new instance of Instrumented. It expects an instance of Chaoskube.
func NewInstrumented(base Chaoskube) *Instrumented {
	return &Instrumented{Chaoskube: base}
}

// DeletePod delegates to the parent and if successful counts it.
func (c *Instrumented) DeletePod(victim v1.Pod) error {
	err := c.Chaoskube.DeletePod(victim)
	if err != nil {
		return err
	}

	// count a successful pod termination.
	NumEvictions.WithLabelValues(victim.Namespace).Inc()

	return nil
}

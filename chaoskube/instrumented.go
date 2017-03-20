package chaoskube

import (
	"k8s.io/client-go/pkg/api/v1"

	"github.com/linki/chaoskube/metrics"
)

type InstrumentedChaoskube struct {
	Interface
}

func NewInstrumented(base Interface) *InstrumentedChaoskube {
	return &InstrumentedChaoskube{base}
}

func (c *InstrumentedChaoskube) DeletePod(victim v1.Pod) error {
	err := c.Interface.DeletePod(victim)
	if err != nil {
		return err
	}

	metrics.NumEvictions.WithLabelValues(victim.Namespace).Inc()

	return nil
}

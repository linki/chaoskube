package chaoskube

import (
	"k8s.io/client-go/pkg/api/v1"

	"github.com/linki/chaoskube/metrics"
)

type InstrumentedChaoskube struct {
	*Chaoskube
}

func (c *InstrumentedChaoskube) DeletePod(victim v1.Pod) error {
	err := c.Chaoskube.DeletePod(victim)
	if err != nil {
		return err
	}

	metrics.NumEvictions.WithLabelValues(victim.Namespace).Inc()

	return nil
}

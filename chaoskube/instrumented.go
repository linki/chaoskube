package chaoskube

import (
	log "github.com/Sirupsen/logrus"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/labels"

	"github.com/linki/chaoskube/metrics"
)

type InstrumentedChaoskube struct {
	*Chaoskube
}

func NewInstrumented(client kubernetes.Interface, labels, annotations, namespaces labels.Selector, logger log.StdLogger, dryRun bool, seed int64) *InstrumentedChaoskube {
	return &InstrumentedChaoskube{
		New(client, labels, annotations, namespaces, logger, dryRun, seed),
	}
}

func (c *InstrumentedChaoskube) DeletePod(victim v1.Pod) error {
	err := c.Chaoskube.DeletePod(victim)
	if err != nil {
		return err
	}

	metrics.NumEvictions.WithLabelValues(victim.Namespace).Inc()

	return nil
}

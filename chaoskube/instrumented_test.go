package chaoskube

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	prommodel "github.com/prometheus/client_model/go"

	"k8s.io/client-go/pkg/labels"
)

var _ Chaoskube = &Instrumented{}

func TestDeletePodMetrics(t *testing.T) {
	chaoskube := setupInstrumented(t)

	victim := newPod("default", "foo")

	if err := chaoskube.DeletePod(victim); err != nil {
		t.Fatal(err)
	}

	// remove when we run the base tests against this implementation
	validateCandidates(t, chaoskube.Chaoskube, []map[string]string{
		{"namespace": "testing", "name": "bar"},
	})

	metric, err := NumEvictions.GetMetricWith(prometheus.Labels{"pod_namespace": "default"})
	if err != nil {
		t.Fatal(err)
	}

	validateCounter(t, metric, 1)
}

//

func validateCounter(t *testing.T, counter prometheus.Counter, value int) {
	rawMetric := prommodel.Metric{}
	counter.Write(&rawMetric)
	counterValue := int(rawMetric.Counter.GetValue())

	if counterValue != value {
		t.Errorf("expected %d, got %d", value, counterValue)
	}
}

func setupInstrumented(t *testing.T) *Instrumented {
	NumEvictions.Reset()

	return NewInstrumented(setup(t, labels.Everything(), labels.Everything(), labels.Everything(), false, 0))
}

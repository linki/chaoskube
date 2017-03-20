package chaoskube

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	prommodel "github.com/prometheus/client_model/go"

	"k8s.io/client-go/pkg/labels"

	"github.com/linki/chaoskube/metrics"
)

//
func TestDeletePodMetrics(t *testing.T) {
	chaoskubeOld := setup(t, labels.Everything(), labels.Everything(), labels.Everything(), false, 0)

	chaoskube := &InstrumentedChaoskube{chaoskubeOld}

	victim := newPod("default", "foo")

	metrics.NumEvictions.Reset()

	if err := chaoskube.DeletePod(victim); err != nil {
		t.Fatal(err)
	}

	validateLog(t, "Killing pod default/foo")

	metric, err := metrics.NumEvictions.GetMetricWith(prometheus.Labels{"pod_namespace": "default"})
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

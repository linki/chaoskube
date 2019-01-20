package strategy

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/linki/chaoskube/internal/testutil"
	"github.com/linki/chaoskube/util"

	"github.com/stretchr/testify/suite"
)

type DeletePodStrategySuite struct {
	testutil.TestSuite
}

var (
	logger, logOutput = test.NewNullLogger()
)

func (suite *DeletePodStrategySuite) SetupTest() {
	logger.SetLevel(log.DebugLevel)
	logOutput.Reset()
}

func (suite *DeletePodStrategySuite) TestInterface() {
	suite.Implements((*Strategy)(nil), new(DeletePodStrategy))
}

func (suite *DeletePodStrategySuite) TestTerminate() {
	logOutput.Reset()
	client := fake.NewSimpleClientset()
	strategy := NewDeletePodStrategy(client, logger, 10*time.Second)

	pods := []v1.Pod{
		util.NewPod("default", "foo", v1.PodRunning),
		util.NewPod("testing", "bar", v1.PodRunning),
	}

	for _, pod := range pods {
		_, err := client.CoreV1().Pods(pod.Namespace).Create(&pod)
		suite.Require().NoError(err)
	}

	victim := util.NewPod("default", "foo", v1.PodRunning)

	err := strategy.Terminate(victim)
	suite.Require().NoError(err)

	suite.AssertLog(logOutput, log.DebugLevel, "calling deletePod endpoint", log.Fields{"namespace": "default", "name": "foo"})

	remainingPods, err := client.CoreV1().Pods(v1.NamespaceAll).List(metav1.ListOptions{})
	suite.Require().NoError(err)

	suite.AssertPods(remainingPods.Items, []map[string]string{
		{"namespace": "testing", "name": "bar"},
	})
}

func (suite *DeletePodStrategySuite) TestDeleteOptions() {
	for _, tt := range []struct {
		gracePeriod time.Duration
		expected    *metav1.DeleteOptions
	}{
		{
			-1,
			nil,
		},
		{
			0,
			&metav1.DeleteOptions{GracePeriodSeconds: int64Ptr(0)},
		},
		{
			300,
			&metav1.DeleteOptions{GracePeriodSeconds: int64Ptr(300)},
		},
	} {
		suite.Equal(tt.expected, deleteOptions(tt.gracePeriod))
	}
}

func TestDeletePodStrategySuite(t *testing.T) {
	suite.Run(t, new(DeletePodStrategySuite))
}

func int64Ptr(value int64) *int64 {
	return &value
}

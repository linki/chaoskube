package strategy

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/linki/chaoskube/util"

	"github.com/stretchr/testify/suite"
)

type DeletePodSuite struct {
	suite.Suite
}

var (
	logger, logOutput = test.NewNullLogger()
)

func (suite *DeletePodSuite) SetupTest() {
	logger.SetLevel(log.DebugLevel)
	logOutput.Reset()
}

func (suite *DeletePodSuite) TestInterface() {
	suite.Implements((*Strategy)(nil), new(DeletePodStrategy))
}

func (suite *DeletePodSuite) TestTerminate() {
	foo := map[string]string{"namespace": "default", "name": "foo"}
	bar := map[string]string{"namespace": "testing", "name": "bar"}

	for _, tt := range []struct {
		dryRun        bool
		remainingPods []map[string]string
	}{
		{false, []map[string]string{bar}},
		{true, []map[string]string{foo, bar}},
	} {
		logOutput.Reset()
		client := fake.NewSimpleClientset()
		_strategy := NewDeletePodStrategy(client, 10*time.Second, tt.dryRun, logger)

		pods := []v1.Pod{
			util.NewPod("default", "foo", v1.PodRunning),
			util.NewPod("testing", "bar", v1.PodRunning),
		}

		for _, pod := range pods {
			_, err := client.CoreV1().Pods(pod.Namespace).Create(&pod)
			suite.Require().NoError(err)
		}

		victim := util.NewPod("default", "foo", v1.PodRunning)

		err := _strategy.Terminate(victim)
		suite.Require().NoError(err)

		suite.assertLog(log.InfoLevel, "terminating pod", log.Fields{"namespace": "default", "name": "foo"})
		suite.assertPods(client, tt.remainingPods)
	}
}

func (suite *DeletePodSuite) assertPods(client kubernetes.Interface, expected []map[string]string) {
	pods, err := client.CoreV1().Pods(v1.NamespaceAll).List(metav1.ListOptions{})
	suite.Require().NoError(err)

	suite.Require().Len(pods.Items, len(expected))
	for i, pod := range pods.Items {
		suite.assertPod(pod, expected[i])
	}
}

func (suite *DeletePodSuite) assertPod(pod v1.Pod, expected map[string]string) {
	suite.Equal(expected["namespace"], pod.Namespace)
	suite.Equal(expected["name"], pod.Name)
}

func (suite *DeletePodSuite) assertLog(level log.Level, msg string, fields log.Fields) {
	suite.Require().NotEmpty(logOutput.Entries)

	lastEntry := logOutput.LastEntry()
	suite.Equal(level, lastEntry.Level)
	suite.Equal(msg, lastEntry.Message)
	for k := range fields {
		suite.Equal(fields[k], lastEntry.Data[k])
	}
}

func (suite *DeletePodSuite) TestDeleteOptions() {
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

func TestDeletePodSuite(t *testing.T) {
	suite.Run(t, new(DeletePodSuite))
}

func int64Ptr(value int64) *int64 {
	return &value
}

package terminator

import (
	"context"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/linki/chaoskube/internal/testutil"
	"github.com/linki/chaoskube/util"

	"github.com/stretchr/testify/suite"
)

type DeletePodTerminatorSuite struct {
	testutil.TestSuite
}

var (
	logger, logOutput = test.NewNullLogger()
)

func (suite *DeletePodTerminatorSuite) SetupTest() {
	logger.SetLevel(log.DebugLevel)
	logOutput.Reset()
}

func (suite *DeletePodTerminatorSuite) TestInterface() {
	suite.Implements((*Terminator)(nil), new(DeletePodTerminator))
}

func (suite *DeletePodTerminatorSuite) TestTerminate() {
	logOutput.Reset()
	client := fake.NewSimpleClientset()
	terminator := NewDeletePodTerminator(client, logger, 10*time.Second)

	pods := []v1.Pod{
		util.NewPodBuilder("default", "foo").Build(),
		util.NewPodBuilder("testing", "bar").Build(),
	}

	for _, pod := range pods {
		_, err := client.CoreV1().Pods(pod.Namespace).Create(context.Background(), &pod, metav1.CreateOptions{})
		suite.Require().NoError(err)
	}

	victim := util.NewPodBuilder("default", "foo").Build()

	err := terminator.Terminate(context.Background(), victim)
	suite.Require().NoError(err)

	suite.AssertLog(logOutput, log.DebugLevel, "calling deletePod endpoint", log.Fields{"namespace": "default", "name": "foo"})

	remainingPods, err := client.CoreV1().Pods(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	suite.Require().NoError(err)

	suite.AssertPods(remainingPods.Items, []map[string]string{
		{"namespace": "testing", "name": "bar"},
	})
}

func (suite *DeletePodTerminatorSuite) TestDeleteOptions() {
	for _, tt := range []struct {
		gracePeriod time.Duration
		expected    metav1.DeleteOptions
	}{
		{
			-1,
			metav1.DeleteOptions{},
		},
		{
			0,
			metav1.DeleteOptions{GracePeriodSeconds: int64Ptr(0)},
		},
		{
			300,
			metav1.DeleteOptions{GracePeriodSeconds: int64Ptr(300)},
		},
	} {
		suite.Equal(tt.expected, deleteOptions(tt.gracePeriod))
	}
}

func TestDeletePodTerminatorSuite(t *testing.T) {
	suite.Run(t, new(DeletePodTerminatorSuite))
}

func int64Ptr(value int64) *int64 {
	return &value
}

package notifier

import (
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/linki/chaoskube/internal/testutil"

	"github.com/stretchr/testify/suite"
)

type NotifierSuite struct {
	testutil.TestSuite
}

func (suite *NotifierSuite) TestMultiNotifierWithoutNotifiers() {
	manager := New()
	err := manager.NotifyPodTermination(v1.Pod{})
	suite.NoError(err)
}

func (suite *NotifierSuite) TestMultiNotifierWithNotifier() {
	manager := New()
	n := Noop{}
	manager.Add(&n)
	err := manager.NotifyPodTermination(v1.Pod{})
	suite.Require().NoError(err)

	suite.Equal(1, n.Calls)
}

func (suite *NotifierSuite) TestMultiNotifierWithMultipleNotifier() {
	manager := New()
	n1 := Noop{}
	n2 := Noop{}
	manager.Add(&n1)
	manager.Add(&n2)

	err := manager.NotifyPodTermination(v1.Pod{})
	suite.Require().NoError(err)

	suite.Equal(1, n1.Calls)
	suite.Equal(1, n2.Calls)
}

func TestNotifierSuite(t *testing.T) {
	suite.Run(t, new(NotifierSuite))
}

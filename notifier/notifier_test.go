package notifier

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/linki/chaoskube/internal/testutil"

	"github.com/stretchr/testify/suite"
)

type NotifierSuite struct {
	testutil.TestSuite
}

type FailingNotifier struct{}

func (f FailingNotifier) NotifyPodTermination(pod v1.Pod) error {
	return fmt.Errorf("notify error")
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

func (suite *NotifierSuite) TestMultiNotifierWithNotifierError() {
	manager := New()
	f := FailingNotifier{}
	manager.Add(&f)
	err := manager.NotifyPodTermination(v1.Pod{})
	suite.Require().Error(err)
}

func (suite *NotifierSuite) TestMultiNotifierWithNotifierMultipleError() {
	manager := New()
	f0 := FailingNotifier{}
	f1 := FailingNotifier{}
	manager.Add(&f0)
	manager.Add(&f1)
	err := manager.NotifyPodTermination(v1.Pod{}).(*multierror.Error)
	suite.Require().Error(err)
	suite.Require().Len(err.Errors, 2)
}

func (suite *NotifierSuite) TestMultiNotifierWithOneFailingNotifier() {
	manager := New()
	f := FailingNotifier{}
	n := Noop{}
	manager.Add(&n)
	manager.Add(&f)
	err := manager.NotifyPodTermination(v1.Pod{}).(*multierror.Error)
	suite.Require().Error(err)
	suite.Require().Len(err.Errors, 1)
}

func TestNotifierSuite(t *testing.T) {
	suite.Run(t, new(NotifierSuite))
}

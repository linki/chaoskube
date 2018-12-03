package strategy

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
}

var _ Strategy = &DeletePodStrategy{}

func (suite *Suite) TestDeleteOptions() {
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

func int64Ptr(value int64) *int64 {
	return &value
}

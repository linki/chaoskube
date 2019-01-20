package testutil

import (
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"k8s.io/api/core/v1"

	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
}

func (suite *TestSuite) AssertPods(pods []v1.Pod, expected []map[string]string) {
	suite.Require().Len(pods, len(expected))

	for i, pod := range pods {
		suite.AssertPod(pod, expected[i])
	}
}

func (suite *TestSuite) AssertPod(pod v1.Pod, expected map[string]string) {
	suite.Equal(expected["namespace"], pod.Namespace)
	suite.Equal(expected["name"], pod.Name)
}

func (suite *TestSuite) AssertLog(output *test.Hook, level log.Level, msg string, fields log.Fields) {
	suite.Require().NotEmpty(output.Entries)

	lastEntry := output.LastEntry()
	suite.Equal(level, lastEntry.Level)
	suite.Equal(msg, lastEntry.Message)
	for k := range fields {
		suite.Equal(fields[k], lastEntry.Data[k])
	}
}

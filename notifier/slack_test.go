package notifier

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linki/chaoskube/internal/testutil"
	"github.com/linki/chaoskube/util"

	"github.com/stretchr/testify/suite"
)

type SlackSuite struct {
	testutil.TestSuite
}

func (suite *SlackSuite) TestSlackNotificationForTerminationStatusOk() {
	webhookPath := "/services/T07M5HUDA/BQ1U5VDGA/yhpIczRK0cZ3jDLK1U8qD634"

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		suite.Require().Equal(webhookPath, req.URL.Path)
		res.WriteHeader(200)
		_, err := res.Write([]byte("ok"))
		suite.Require().NoError(err)
	}))
	defer testServer.Close()

	testPod := util.NewPodBuilder("chaos", "chaos-57df4db6b-h9ktj").Build()

	slack := NewSlackNotifier(testServer.URL + webhookPath)
	err := slack.NotifyPodTermination(testPod)

	suite.NoError(err)
}

func (suite *SlackSuite) TestSlackNotificationForTerminationStatus500() {
	webhookPath := "/services/T07M5HUDA/BQ1U5VDGA/yhpIczRK0cZ3jDLK1U8qD634"

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		suite.Require().Equal(webhookPath, req.URL.Path)
		res.WriteHeader(500)
		_, err := res.Write([]byte("ok"))
		suite.Require().NoError(err)
	}))
	defer testServer.Close()

	testPod := util.NewPodBuilder("chaos", "chaos-57df4db6b-h9ktj").Build()

	slack := NewSlackNotifier(testServer.URL + webhookPath)
	err := slack.NotifyPodTermination(testPod)

	suite.Error(err)
}

func TestSlackSuite(t *testing.T) {
	suite.Run(t, new(SlackSuite))
}

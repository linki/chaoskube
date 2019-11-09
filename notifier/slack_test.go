package notifier

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSlackNotificationForTerminationStatusOk(t *testing.T) {
	webhookPath := "/services/T07M5HUDA/BQ1U5VDGA/yhpIczRK0cZ3jDLK1U8qD634"

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		assert.Equal(t, req.URL.Path, webhookPath)
		res.WriteHeader(200)
		res.Write([]byte("ok"))
	}))

	defer testServer.Close()

	slack := NewSlackNotifier(testServer.URL + webhookPath)
	err := slack.NotifyTermination(Termination{
		Pod:       "chaos-57df4db6b-h9ktj",
		Namespace: "chaos",
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestSlackNotificationForTerminationStatus500(t *testing.T) {
	webhookPath := "/services/T07M5HUDA/BQ1U5VDGA/yhpIczRK0cZ3jDLK1U8qD634"

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		assert.Equal(t, req.URL.Path, webhookPath)
		res.WriteHeader(500)
		if _, err := res.Write([]byte("ok")); err != nil {
			t.Fatal(err)
		}
	}))
	defer testServer.Close()

	slack := NewSlackNotifier(testServer.URL + webhookPath)
	err := slack.NotifyTermination(Termination{
		Pod:       "chaos-57df4db6b-h9ktj",
		Namespace: "chaos",
	})

	if err == nil {
		t.Fatal("expected error on status code 500")
	}
}

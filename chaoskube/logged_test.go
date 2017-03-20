package chaoskube

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/labels"
)

var _ Chaoskube = &Logged{}

var logOutput = bytes.NewBuffer([]byte{})
var logger = log.New(logOutput, "", 0)

func TestNewLogged(t *testing.T) {
	chaoskube := NewLogged(logger, setup(t, labels.Everything(), labels.Everything(), labels.Everything(), false, 0))

	if chaoskube.Logger != logger {
		t.Errorf("expected %#v, got %#v", logger, chaoskube.Logger)
	}
}

func TestDeletePodLog(t *testing.T) {
	chaoskube := setupLogged(t)

	victim := newPod("default", "foo")

	if err := chaoskube.DeletePod(victim); err != nil {
		t.Fatal(err)
	}

	validateLog(t, "Killing pod default/foo")

	// replace once we run base tests against this one
	validateCandidates(t, chaoskube.Chaoskube, []map[string]string{
		{"namespace": "testing", "name": "bar"},
	})
}

// TODO: replace when we run all tests against this
func TestTerminateVictimTerminates(t *testing.T) {
	chaoskube := setupLogged(t)

	if err := chaoskube.TerminateVictim(nil, nil); err != nil {
		t.Fatal(err)
	}

	validateCandidates(t, chaoskube.Chaoskube, []map[string]string{
		{"namespace": "testing", "name": "bar"},
	})
}

func TestTerminateNoVictimLogs(t *testing.T) {
	logOutput.Reset()

	chaoskube := NewLogged(logger, New(fake.NewSimpleClientset(), labels.Everything(), labels.Everything(), labels.Everything(), false, 0))

	if err := chaoskube.TerminateVictim(nil, nil); err != nil {
		t.Fatal(err)
	}

	validateLog(t, msgVictimNotFound)
}

// helpers

func validateLog(t *testing.T, msg string) {
	if !strings.Contains(logOutput.String(), msg) {
		t.Errorf("expected string '%s' in '%s'.", msg, logOutput.String())
	}
}

func setupLogged(t *testing.T) *Logged {
	logOutput.Reset()

	return NewLogged(logger, setup(t, labels.Everything(), labels.Everything(), labels.Everything(), false, 2000))
}

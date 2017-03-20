package chaoskube

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/labels"
)

var _ Interface = &LoggedChaoskube{}

var logOutput = bytes.NewBuffer([]byte{})
var logger = log.New(logOutput, "", 0)

func TestNewLogged(t *testing.T) {
	chaoskube := NewLogged(logger, setup(t, labels.Everything(), labels.Everything(), labels.Everything(), false, 0))

	if chaoskube.Logger != logger {
		t.Errorf("expected %#v, got %#v", logger, chaoskube.Logger)
	}
}

// TestDeletePod tests deleting a particular pod
func TestDeletePodLog(t *testing.T) {
	logOutput.Reset()

	chaoskube := NewLogged(logger, setup(t, labels.Everything(), labels.Everything(), labels.Everything(), false, 0))

	victim := newPod("default", "foo")

	if err := chaoskube.DeletePod(victim); err != nil {
		t.Fatal(err)
	}

	validateLog(t, "Killing pod default/foo")

	validateCandidates(t, chaoskube.Interface.(*Chaoskube), []map[string]string{
		{"namespace": "testing", "name": "bar"},
	})
}

// TestTerminateVictim tests that the correct victim pod is chosen and deleted
func TestTerminateVictimTerminates(t *testing.T) {
	chaoskube := NewLogged(logger, setup(t, labels.Everything(), labels.Everything(), labels.Everything(), false, 2000))

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateCandidates(t, chaoskube.Interface.(*Chaoskube), []map[string]string{
		{"namespace": "testing", "name": "bar"},
	})
}

// TestTerminateNoVictimLogsInfo tests that missing victim prints a log message
func TestTerminateNoVictimLogs(t *testing.T) {
	logOutput.Reset()

	chaoskube := NewLogged(logger, New(fake.NewSimpleClientset(), labels.Everything(), labels.Everything(), labels.Everything(), false, 0))

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateLog(t, msgVictimNotFound)
}

//

func validateLog(t *testing.T, msg string) {
	if !strings.Contains(logOutput.String(), msg) {
		t.Errorf("expected string '%s' in '%s'.", msg, logOutput.String())
	}
}

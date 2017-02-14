package chaoskube

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/labels"
)

var logOutput = bytes.NewBuffer([]byte{})
var logger = log.New(logOutput, "", 0)

// TestNew tests that arguments are passed to the new instance correctly
func TestNew(t *testing.T) {
	client := fake.NewSimpleClientset()
	selector, _ := labels.Parse("foo=bar")
	namespaces, _ := labels.Parse("qux")

	chaoskube := New(client, selector, namespaces, logger, false, 42)

	if chaoskube == nil {
		t.Errorf("expected Chaoskube but got nothing")
	}

	if chaoskube.Client != client {
		t.Errorf("expected %#v, got %#v", client, chaoskube.Client)
	}

	if chaoskube.Labels.String() != "foo=bar" {
		t.Errorf("expected %s, got %s", "foo=bar", chaoskube.Labels.String())
	}

	if chaoskube.Namespaces.String() != "qux" {
		t.Errorf("expected %s, got %s", "qux", chaoskube.Namespaces.String())
	}

	if chaoskube.Logger != logger {
		t.Errorf("expected %#v, got %#v", logger, chaoskube.Logger)
	}

	if chaoskube.DryRun != false {
		t.Errorf("expected %t, got %t", false, chaoskube.DryRun)
	}

	if chaoskube.Seed != 42 {
		t.Errorf("expected %d, got %d", 42, chaoskube.Seed)
	}
}

// TestCandidates tests the set of pods available for termination
func TestCandidates(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), labels.Everything(), false, 0)

	validateCandidates(t, chaoskube, []map[string]string{
		{"namespace": "default", "name": "foo"},
		{"namespace": "testing", "name": "bar"},
	})
}

// TestCandidatesLabelSelector tests that the list of pods available for
// termination can be restricted by providing a label selector.
func TestCandidatesLabelSelector(t *testing.T) {
	selector, err := labels.Parse("app=foo")
	if err != nil {
		t.Fatal(err)
	}

	chaoskube := setup(t, selector, labels.Everything(), false, 0)

	validateCandidates(t, chaoskube, []map[string]string{
		{"namespace": "default", "name": "foo"},
	})
}

// TestCandidatesExcludingLabelSelector tests that label selector supports exclusion
func TestCandidatesExcludingLabelSelector(t *testing.T) {
	selector, err := labels.Parse("app!=foo")
	if err != nil {
		t.Fatal(err)
	}

	chaoskube := setup(t, selector, labels.Everything(), false, 0)

	validateCandidates(t, chaoskube, []map[string]string{
		{"namespace": "testing", "name": "bar"},
	})
}

// TestCandidatesNamespaces tests that the list of pods available for
// termination can be restricted by namespaces.
func TestCandidatesNamespaces(t *testing.T) {
	foo := map[string]string{"namespace": "default", "name": "foo"}
	bar := map[string]string{"namespace": "testing", "name": "bar"}

	for _, test := range []struct {
		namespaces string
		pods       []map[string]string
	}{
		{"", []map[string]string{foo, bar}},
		{"default", []map[string]string{foo}},
		{"default,testing", []map[string]string{foo, bar}},
		{"!testing", []map[string]string{foo}},
		{"!default,!testing", []map[string]string{}},
		{"default,!testing", []map[string]string{foo}},
		{"default,!default", []map[string]string{}},
	} {
		namespaces, err := labels.Parse(test.namespaces)
		if err != nil {
			t.Fatal(err)
		}

		chaoskube := setup(t, labels.Everything(), namespaces, false, 0)

		validateCandidates(t, chaoskube, test.pods)
	}
}

// TestVictim tests that a pod is chosen from the candidates
func TestVictim(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), labels.Everything(), false, 2000)

	validateVictim(t, chaoskube, map[string]string{
		"namespace": "default", "name": "foo",
	})
}

// TestAnotherVictim tests that the chosen victim is different for another seed
func TestAnotherVictim(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), labels.Everything(), false, 4000)

	validateVictim(t, chaoskube, map[string]string{
		"namespace": "testing", "name": "bar",
	})
}

// TestAnotherVictimRespectsLabelSelector tests that a pod chosen from the
// candidates respects the provided label selector
func TestAnotherVictimRespectsLabelSelector(t *testing.T) {
	selector, err := labels.Parse("app=foo")
	if err != nil {
		t.Fatal(err)
	}

	chaoskube := setup(t, selector, labels.Everything(), false, 4000)

	validateVictim(t, chaoskube, map[string]string{
		"namespace": "default", "name": "foo",
	})
}

// TestNoVictimReturnsError tests that on missing victim it returns a known error
func TestNoVictimReturnsError(t *testing.T) {
	chaoskube := New(fake.NewSimpleClientset(), labels.Everything(), labels.Everything(), logger, false, 2000)

	if _, err := chaoskube.Victim(); err != ErrPodNotFound {
		t.Errorf("expected %#v, got %#v", ErrPodNotFound, err)
	}
}

// TestDeletePod tests deleting a particular pod
func TestDeletePod(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), labels.Everything(), false, 0)

	victim := newPod("default", "foo")

	if err := chaoskube.DeletePod(victim); err != nil {
		t.Fatal(err)
	}

	validateLog(t, "Killing pod default/foo")

	validateCandidates(t, chaoskube, []map[string]string{
		{"namespace": "testing", "name": "bar"},
	})
}

// TestDeletePodDryRun tests that enabled dry run doesn't delete the pod
func TestDeletePodDryRun(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), labels.Everything(), true, 0)

	victim := newPod("default", "foo")

	if err := chaoskube.DeletePod(victim); err != nil {
		t.Fatal(err)
	}

	validateCandidates(t, chaoskube, []map[string]string{
		{"namespace": "default", "name": "foo"},
		{"namespace": "testing", "name": "bar"},
	})
}

// TestTerminateVictim tests that the correct victim pod is chosen and deleted
func TestTerminateVictim(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), labels.Everything(), false, 2000)

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateCandidates(t, chaoskube, []map[string]string{
		{"namespace": "testing", "name": "bar"},
	})
}

// TestTerminateNoVictimLogsInfo tests that missing victim prints a log message
func TestTerminateNoVictimLogsInfo(t *testing.T) {
	logOutput.Reset()
	chaoskube := New(fake.NewSimpleClientset(), labels.Everything(), labels.Everything(), logger, false, 0)

	if err := chaoskube.TerminateVictim(); err != nil {
		t.Fatal(err)
	}

	validateLog(t, "No victim could be found")
}

// helper functions

func validateCandidates(t *testing.T, chaoskube *Chaoskube, expected []map[string]string) {
	pods, err := chaoskube.Candidates()
	if err != nil {
		t.Fatal(err)
	}

	validatePods(t, pods, expected)
}

func validateVictim(t *testing.T, chaoskube *Chaoskube, expected map[string]string) {
	victim, err := chaoskube.Victim()
	if err != nil {
		t.Fatal(err)
	}

	validatePod(t, victim, expected)
}

func validatePods(t *testing.T, pods []v1.Pod, expected []map[string]string) {
	if len(pods) != len(expected) {
		t.Fatalf("expected %d pod(s), got %d", len(expected), len(pods))
	}

	for i, pod := range pods {
		validatePod(t, pod, expected[i])
	}
}

func validatePod(t *testing.T, pod v1.Pod, expected map[string]string) {
	if pod.Namespace != expected["namespace"] {
		t.Errorf("expected %s, got %s", expected["namespace"], pod.Namespace)
	}

	if pod.Name != expected["name"] {
		t.Errorf("expected %s, got %s", expected["name"], pod.Name)
	}
}

func validateLog(t *testing.T, msg string) {
	if !strings.Contains(logOutput.String(), msg) {
		t.Errorf("expected string '%s' in '%s'.", msg, logOutput.String())
	}
}

func newPod(namespace, name string) v1.Pod {
	pod := v1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				"app": name,
			},
		},
	}

	return pod
}

func setup(t *testing.T, selector labels.Selector, namespaces labels.Selector, dryRun bool, seed int64) *Chaoskube {
	pods := []v1.Pod{
		newPod("default", "foo"),
		newPod("testing", "bar"),
	}

	client := fake.NewSimpleClientset()

	for _, pod := range pods {
		if _, err := client.Core().Pods(pod.Namespace).Create(&pod); err != nil {
			t.Fatal(err)
		}
	}

	logOutput.Reset()

	return New(client, selector, namespaces, logger, dryRun, seed)
}

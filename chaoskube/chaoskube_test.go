package chaoskube

import (
	"testing"

	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/labels"
)

// TestNew tests that arguments are passed to the new instance correctly
func TestNew(t *testing.T) {
	client := fake.NewSimpleClientset()
	selector := labels.SelectorFromSet(labels.Set{"foo": "bar"})

	chaoskube := New(client, selector, false, 42)

	if chaoskube == nil {
		t.Errorf("expected Chaoskube but got nothing")
	}

	if chaoskube.Client != client {
		t.Errorf("expected %#v, got %#v", client, chaoskube.Client)
	}

	if chaoskube.Selector.String() != "foo=bar" {
		t.Errorf("expected %s, got %s", "foo=bar", chaoskube.Selector.String())
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
	chaoskube := setup(t, labels.Everything(), false, 0)

	pods, err := chaoskube.Candidates()
	if err != nil {
		t.Fatal(err)
	}

	validatePods(t, pods, []map[string]string{
		{"namespace": "default", "name": "foo"},
		{"namespace": "default", "name": "bar"},
	})
}

// TestCandidatesLabelSelector tests that the list of pods available for
// termination can be restricted by providing a label selector.
func TestCandidatesLabelSelector(t *testing.T) {
	selector, err := labels.Parse("app=foo")
	if err != nil {
		t.Fatal(err)
	}

	chaoskube := setup(t, selector, false, 0)

	pods, err := chaoskube.Candidates()
	if err != nil {
		t.Fatal(err)
	}

	validatePods(t, pods, []map[string]string{
		{"namespace": "default", "name": "foo"},
	})
}

// TestCandidatesExcludingLabelSelector tests that label selector supports exclusion
func TestCandidatesExcludingLabelSelector(t *testing.T) {
	selector, err := labels.Parse("app!=foo")
	if err != nil {
		t.Fatal(err)
	}

	chaoskube := setup(t, selector, false, 0)

	pods, err := chaoskube.Candidates()
	if err != nil {
		t.Fatal(err)
	}

	validatePods(t, pods, []map[string]string{
		{"namespace": "default", "name": "bar"},
	})
}

// TestVictim tests that a pod is chosen from the candidates
func TestVictim(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), false, 2000)

	victim, err := chaoskube.Victim()
	if err != nil {
		t.Fatal(err)
	}

	validatePod(t, victim, map[string]string{
		"namespace": "default", "name": "foo",
	})
}

// TestAnotherVictim tests that the chosen victim is different for another seed
func TestAnotherVictim(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), false, 4000)

	victim, err := chaoskube.Victim()
	if err != nil {
		t.Fatal(err)
	}

	validatePod(t, victim, map[string]string{
		"namespace": "default", "name": "bar",
	})
}

// TestAnotherVictimRespectsLabelSelector tests that a pod chosen from the
// candidates respects the provided label selector
func TestAnotherVictimRespectsLabelSelector(t *testing.T) {
	selector, err := labels.Parse("app=foo")
	if err != nil {
		t.Fatal(err)
	}

	chaoskube := setup(t, selector, false, 4000)

	victim, err := chaoskube.Victim()
	if err != nil {
		t.Fatal(err)
	}

	validatePod(t, victim, map[string]string{
		"namespace": "default", "name": "foo",
	})
}

// TestDeletePod tests deleting a particular pod
func TestDeletePod(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), false, 0)

	victim := newPod("default", "foo")

	if err := chaoskube.DeletePod(victim); err != nil {
		t.Fatal(err)
	}

	pods, err := chaoskube.Candidates()
	if err != nil {
		t.Fatal(err)
	}

	validatePods(t, pods, []map[string]string{
		{"namespace": "default", "name": "bar"},
	})
}

// TestDeletePodDryRun tests that enabled dry run doesn't delete the pod
func TestDeletePodDryRun(t *testing.T) {
	chaoskube := setup(t, labels.Everything(), true, 0)

	victim := newPod("default", "foo")

	if err := chaoskube.DeletePod(victim); err != nil {
		t.Fatal(err)
	}

	pods, err := chaoskube.Candidates()
	if err != nil {
		t.Fatal(err)
	}

	validatePods(t, pods, []map[string]string{
		{"namespace": "default", "name": "foo"},
		{"namespace": "default", "name": "bar"},
	})
}

// helper functions

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

func setup(t *testing.T, selector labels.Selector, dryRun bool, seed int64) *Chaoskube {
	pods := []v1.Pod{
		newPod("default", "foo"),
		newPod("default", "bar"),
	}

	client := fake.NewSimpleClientset()

	for _, pod := range pods {
		if _, err := client.Core().Pods(pod.Namespace).Create(&pod); err != nil {
			t.Fatal(err)
		}
	}

	return New(client, selector, dryRun, seed)
}

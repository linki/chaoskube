package chaoskube

import "k8s.io/client-go/pkg/api/v1"

// Chaoskube TODO(linki)
type Chaoskube interface {
	// Candidates should return the list of possible victim pods.
	Candidates() ([]v1.Pod, error)
	// Victim should return a random pod from the list of Candidates.
	Victim() (v1.Pod, error)
	// DeletePod should terminate any pod that is passed to it.
	DeletePod(victim v1.Pod) error
	// TerminateVictim should pick and terminate a victim.
	TerminateVictim() error
}

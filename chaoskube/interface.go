package chaoskube

import "k8s.io/client-go/pkg/api/v1"

type Interface interface {
	Candidates() ([]v1.Pod, error)
	Victim() (v1.Pod, error)
	DeletePod(victim v1.Pod) error
	TerminateVictim() error
}

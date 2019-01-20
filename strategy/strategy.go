package strategy

import (
	"k8s.io/api/core/v1"
)

// Strategy is the interface for implementations of pod terminators.
type Strategy interface {
	// Terminate terminates the given pod.
	Terminate(victim v1.Pod) error
}

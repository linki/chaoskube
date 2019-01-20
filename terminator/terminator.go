package terminator

import (
	"k8s.io/api/core/v1"
)

// Terminator is the interface for implementations of pod terminators.
type Terminator interface {
	// Terminate terminates the given pod.
	Terminate(victim v1.Pod) error
}

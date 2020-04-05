package terminator

import (
	"context"

	v1 "k8s.io/api/core/v1"
)

// Terminator is the interface for implementations of pod terminators.
type Terminator interface {
	// Terminate terminates the given pod.
	Terminate(ctx context.Context, victim v1.Pod) error
}

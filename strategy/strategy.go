package strategy

import (
	"k8s.io/api/core/v1"
)

type Strategy interface {
	// Terminate todo
	Terminate(victim v1.Pod) error
}

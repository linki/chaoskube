package testutil

import (
	"k8s.io/api/core/v1"
)

type TestNotifier struct {
	Calls int
}

func NewTestNotifier() *TestNotifier {
	return &TestNotifier{}
}

func (t *TestNotifier) NotifyTermination(victim v1.Pod) error {
	t.Calls++
	return nil
}

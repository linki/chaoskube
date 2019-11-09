package notifier

import "k8s.io/api/core/v1"

const NotifierNoop = "noop"

type NoopNotifier struct{}

func (n NoopNotifier) NotifyTermination(victim v1.Pod) error {
	return nil
}

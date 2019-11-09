package notifier

import v1 "k8s.io/api/core/v1"

type Notifier interface {
	NotifyTermination(victim v1.Pod) error
}

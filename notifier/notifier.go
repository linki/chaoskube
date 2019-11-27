package notifier

import (
	v1 "k8s.io/api/core/v1"
)

type Notifier interface {
	NotifyTermination(pod v1.Pod) error
}

type Notifiers struct {
	notifiers []Notifier
}

func New() *Notifiers {
	return &Notifiers{notifiers: make([]Notifier, 0)}
}

func (m *Notifiers) NotifyTermination(pod v1.Pod) error {
	for _, n := range m.notifiers {
		if err := n.NotifyTermination(pod); err != nil {
			return err
		}
	}
	return nil
}

func (m *Notifiers) Add(notifier Notifier) {
	m.notifiers = append(m.notifiers, notifier)
}

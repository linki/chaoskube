package notifier

type Notifier interface {
	NotifyTermination(term Termination) error
}

type Termination struct {
	Pod       string
	Namespace string
}

type Notifiers struct {
	notifiers []Notifier
}

func New() *Notifiers {
	return &Notifiers{notifiers: make([]Notifier, 0)}
}

func (m *Notifiers) NotifyTermination(term Termination) error {
	for _, n := range m.notifiers {
		if err := n.NotifyTermination(term); err != nil {
			return err
		}
	}
	return nil
}

func (m *Notifiers) Add(notifier Notifier) {
	m.notifiers = append(m.notifiers, notifier)
}

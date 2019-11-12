package notifier

const NotifierNoop = "noop"

type Noop struct {
	Calls int
}

func (t *Noop) NotifyTermination(termination Termination) error {
	t.Calls++
	return nil
}

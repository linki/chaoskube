package notifier

import (
	"testing"
)

func TestMultiNotifierWithoutNotifiers(t *testing.T) {
	manager := New()
	err := manager.NotifyTermination(Termination{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMultiNotifierWithNotifier(t *testing.T) {
	manager := New()
	n := Noop{}
	manager.Add(&n)
	err := manager.NotifyTermination(Termination{})
	if err != nil {
		t.Fatal(err)
	}

	if n.Calls != 1 {
		t.Errorf("expected %d calls to notifier but got %d", 1, n.Calls)
	}
}

func TestMultiNotifierWithMultipleNotifier(t *testing.T) {
	manager := New()
	n1 := Noop{}
	n2 := Noop{}
	manager.Add(&n1)
	manager.Add(&n2)

	err := manager.NotifyTermination(Termination{})
	if err != nil {
		t.Fatal(err)
	}

	if n1.Calls != 1 {
		t.Errorf("expected %d calls to notifier n1 but got %d", 1, n1.Calls)
	}

	if n2.Calls != 1 {
		t.Errorf("expected %d calls to notifier n2 but got %d", 1, n2.Calls)
	}
}

package chaoskube

import (
	log "github.com/Sirupsen/logrus"

	"k8s.io/client-go/pkg/api/v1"
)

// Logged represents an instance of Chaoskube that logs messages
type Logged struct {
	// parent Chaoskube
	Chaoskube
	// an instance of logrus.StdLogger to write log messages to
	Logger log.StdLogger
}

// msgVictimNotFound is the log message when no victim was found
var msgVictimNotFound = "No victim could be found. If that's surprising double-check your selectors."

// NewLogged returns a new instance of Logged. It expects a logger and an instance of Chaoskube.
func NewLogged(logger log.StdLogger, base Chaoskube) *Logged {
	return &Logged{Chaoskube: base, Logger: logger}
}

// DeletePod logs a message about the pod being terminated then delegates the call.
func (c *Logged) DeletePod(victim v1.Pod) error {
	c.Logger.Printf("Killing pod %s/%s", victim.Namespace, victim.Name)

	return c.Chaoskube.DeletePod(victim)
}

// TerminateVictim handles a missing victim error and logs it instead of failing.
func (c *Logged) TerminateVictim() error {
	victim, err := c.Victim()
	if err == ErrPodNotFound {
		c.Logger.Printf(msgVictimNotFound)
		return nil
	}
	if err != nil {
		return err
	}

	return c.DeletePod(victim)
}

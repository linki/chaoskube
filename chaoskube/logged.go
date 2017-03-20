package chaoskube

import (
	log "github.com/Sirupsen/logrus"

	"k8s.io/client-go/pkg/api/v1"
)

type LoggedChaoskube struct {
	// an instance of logrus.StdLogger to write log messages to
	Logger log.StdLogger

	Interface
}

// msgVictimNotFound is the log message when no victim was found
var msgVictimNotFound = "No victim could be found. If that's surprising double-check your selectors."

func NewLogged(logger log.StdLogger, base Interface) *LoggedChaoskube {
	return &LoggedChaoskube{Logger: logger, Interface: base}
}

func (c *LoggedChaoskube) DeletePod(victim v1.Pod) error {
	c.Logger.Printf("Killing pod %s/%s", victim.Namespace, victim.Name)

	return c.Interface.DeletePod(victim)
}

func (c *LoggedChaoskube) TerminateVictim() error {
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

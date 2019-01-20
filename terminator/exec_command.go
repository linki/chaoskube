package terminator

import (
	"os"

	log "github.com/sirupsen/logrus"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// ExecCommandTerminator todo
type ExecCommandTerminator struct {
	client        restclient.Interface
	config        *restclient.Config
	containerName string
	command       []string
	dryRun        bool
	logger        log.FieldLogger
}

// NewExecCommandTerminator todo
func NewExecCommandTerminator(client restclient.Interface, config *restclient.Config, containerName string, command []string, dryRun bool, logger log.FieldLogger) *ExecCommandTerminator {
	return &ExecCommandTerminator{
		client:        client,
		config:        config,
		containerName: containerName,
		command:       command,
		dryRun:        dryRun,
		logger:        logger.WithField("terminator", "ExecCommand"),
	}
}

func (t *ExecCommandTerminator) Terminate(victim v1.Pod) error {
	t.logger.WithFields(log.Fields{
		"namespace": victim.Namespace,
		"name":      victim.Name,
	}).Info("terminating pod") // todo

	if t.dryRun {
		return nil
	}

	var container string
	if t.containerName == "" {
		for _, c := range victim.Spec.Containers {
			container = c.Name
			break
		}
	} else {
		container = t.containerName
	}

	req := t.client.Post().
		Resource("pods").
		Name(victim.Name).
		Namespace(victim.Namespace).
		SubResource("exec").
		Param("container", container)
	req.VersionedParams(&v1.PodExecOptions{
		Container: container,
		Command:   t.command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(t.config, "POST", req.URL())
	if err != nil {
		return err
	}
	// TODO: Collect stderr/stdout in RAM and log
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:             nil,
		Stdout:            os.Stdout,
		Stderr:            os.Stderr,
		Tty:               false,
		TerminalSizeQueue: nil,
	})

	return err
}

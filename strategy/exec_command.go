package strategy

import (
	"os"

	log "github.com/sirupsen/logrus"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// ExecCommandStrategy simply asks k8s to delete the victim pod
type ExecCommandStrategy struct {
	client        restclient.Interface
	config        *restclient.Config
	containerName string
	command       []string
	dryRun        bool
	logger        log.FieldLogger
}

// NewExecCommandStrategy todo
func NewExecCommandStrategy(client restclient.Interface, config *restclient.Config, containerName string, command []string, dryRun bool, logger log.FieldLogger) Strategy {
	return &ExecCommandStrategy{
		client:        client,
		config:        config,
		containerName: containerName,
		command:       command,
		dryRun:        dryRun,
		logger:        logger.WithField("strategy", "ExecCommand"),
	}
}

func (s *ExecCommandStrategy) Terminate(victim v1.Pod) error {
	s.logger.WithFields(log.Fields{
		"namespace": victim.Namespace,
		"name":      victim.Name,
	}).Info("terminating pod") // todo

	if s.dryRun {
		return nil
	}

	var container string
	if s.containerName == "" {
		for _, c := range victim.Spec.Containers {
			container = c.Name
			break
		}
	} else {
		container = s.containerName
	}

	req := s.client.Post().
		Resource("pods").
		Name(victim.Name).
		Namespace(victim.Namespace).
		SubResource("exec").
		Param("container", container)
	req.VersionedParams(&v1.PodExecOptions{
		Container: container,
		Command:   s.command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(s.config, "POST", req.URL())
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

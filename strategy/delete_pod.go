package strategy

import (
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DeletePodStrategy simply asks k8s to delete the victim pod
type DeletePodStrategy struct {
	client      kubernetes.Interface
	gracePeriod time.Duration
	dryRun      bool
	logger      log.FieldLogger
}

// NewDeletePodStrategy todo
func NewDeletePodStrategy(client kubernetes.Interface, gracePeriod time.Duration, dryRun bool, logger log.FieldLogger) Strategy {
	return &DeletePodStrategy{
		client:      client,
		gracePeriod: gracePeriod,
		dryRun:      dryRun,
		logger:      logger.WithField("strategy", "DeletePod"),
	}
}

func (s *DeletePodStrategy) Terminate(victim v1.Pod) error {
	s.logger.WithFields(log.Fields{
		"namespace": victim.Namespace,
		"name":      victim.Name,
	}).Info("terminating pod") // todo

	if s.dryRun {
		return nil
	}

	return s.client.CoreV1().Pods(victim.Namespace).Delete(victim.Name, deleteOptions(s.gracePeriod))
}

func deleteOptions(gracePeriod time.Duration) *metav1.DeleteOptions {
	if gracePeriod < 0 {
		return nil
	}

	return &metav1.DeleteOptions{GracePeriodSeconds: (*int64)(&gracePeriod)}
}

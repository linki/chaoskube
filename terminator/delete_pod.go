package terminator

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DeletePodTerminator simply asks k8s to delete the victim pod.
type DeletePodTerminator struct {
	client      kubernetes.Interface
	logger      log.FieldLogger
	gracePeriod time.Duration
}

// NewDeletePodTerminator creates and returns a DeletePodTerminator object.
func NewDeletePodTerminator(client kubernetes.Interface, logger log.FieldLogger, gracePeriod time.Duration) *DeletePodTerminator {
	return &DeletePodTerminator{
		client:      client,
		logger:      logger.WithField("terminator", "DeletePod"),
		gracePeriod: gracePeriod,
	}
}

// Terminate sends a request to Kubernetes to delete the pod.
func (t *DeletePodTerminator) Terminate(ctx context.Context, victim v1.Pod) error {
	t.logger.WithFields(log.Fields{
		"namespace": victim.Namespace,
		"name":      victim.Name,
	}).Debug("calling deletePod endpoint")

	return t.client.CoreV1().Pods(victim.Namespace).Delete(ctx, victim.Name, deleteOptions(t.gracePeriod))
}

func deleteOptions(gracePeriod time.Duration) metav1.DeleteOptions {
	if gracePeriod < 0 {
		return metav1.DeleteOptions{}
	}

	return metav1.DeleteOptions{GracePeriodSeconds: (*int64)(&gracePeriod)}
}

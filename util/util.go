package util

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

// NewPod returns a new pod instance for testing purposes.
func NewPod(namespace, name string) v1.Pod {
	return v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				"app": name,
			},
			Annotations: map[string]string{
				"chaos": name,
			},
		},
	}
}

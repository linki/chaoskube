package chaoskube

import (
	"errors"
	"math/rand"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/labels"
)

// Chaoskube represents an instance of chaoskube
type Chaoskube struct {
	// a kubernetes client object
	Client kubernetes.Interface
	// a label selector which restricts the pods to choose from
	Selector labels.Selector
	// a namespace selector which restricts the pods to choose from
	Namespaces labels.Selector
	// dry run will not allow any pod terminations
	DryRun bool
	// seed value for the randomizer
	Seed int64
}

// ErrPodNotFound is returned when no victim could be found
var ErrPodNotFound = errors.New("pod not found")

// New returns a new instance of Chaoskube. It expects a kubernetes client, a
// label and namespace selector to reduce the amount of affected pods as well as
// whether to enable dryRun mode and a seed to seed the randomizer with.
func New(client kubernetes.Interface, selector labels.Selector, namespaces labels.Selector, dryRun bool, seed int64) *Chaoskube {
	c := &Chaoskube{
		Client:     client,
		Selector:   selector,
		Namespaces: namespaces,
		DryRun:     dryRun,
		Seed:       seed,
	}

	rand.Seed(c.Seed)

	return c
}

// Candidates returns the list of pods that are available for termination.
// It returns all pods in all namespaces matching the label selector.
func (c *Chaoskube) Candidates() ([]v1.Pod, error) {
	listOptions := v1.ListOptions{LabelSelector: c.Selector.String()}

	podList, err := c.Client.Core().Pods(c.Namespaces.String()).List(listOptions)
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

// Victim returns a random pod from the list of Candidates.
func (c *Chaoskube) Victim() (v1.Pod, error) {
	pods, err := c.Candidates()
	if err != nil {
		return v1.Pod{}, err
	}

	if len(pods) == 0 {
		return v1.Pod{}, ErrPodNotFound
	}

	index := rand.Intn(len(pods))

	return pods[index], nil
}

// DeletePod deletes the passed in pod iff dry run mode is enabled.
func (c *Chaoskube) DeletePod(victim v1.Pod) error {
	if c.DryRun {
		return nil
	}

	return c.Client.Core().Pods(victim.Namespace).Delete(victim.Name, nil)
}

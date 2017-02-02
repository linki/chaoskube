package chaoskube

import (
	"errors"
	"math/rand"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

// Chaoskube represents an instance of chaoskube
type Chaoskube struct {
	// a kubernetes client object
	Client kubernetes.Interface
	// dry run will not allow any pod terminations
	DryRun bool
	// seed value for the randomizer
	Seed int64
}

// New returns a new instance of Chaoskube. It expects a kubernetes client,
// allows enabling dryRun mode and seeds the randomizer with the given seed.
func New(client kubernetes.Interface, dryRun bool, seed int64) *Chaoskube {
	c := &Chaoskube{
		Client: client,
		DryRun: dryRun,
		Seed:   seed,
	}

	rand.Seed(c.Seed)

	return c
}

// Candidates returns the list of pods that are available for termination.
// Currently it returns all pods in all namespaces.
func (c *Chaoskube) Candidates() ([]v1.Pod, error) {
	podList, err := c.Client.Core().Pods(v1.NamespaceAll).List(v1.ListOptions{})
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
		return v1.Pod{}, errors.New("pod not found")
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

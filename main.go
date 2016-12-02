package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

var (
	interval  time.Duration
	inCluster bool
)

func init() {
	kingpin.Flag("interval", "Interval between Pod terminations").Short('i').Default("10m").DurationVar(&interval)
	kingpin.Flag("in-cluster", "If true, finds the Kubernetes cluster from the environment").Short('c').BoolVar(&inCluster)
}

func main() {
	kingpin.Parse()

	client, err := newClient()
	if err != nil {
		log.Fatal(err)
	}

	for {
		pods, err := client.Core().Pods(v1.NamespaceAll).List(v1.ListOptions{})
		if err != nil {
			log.Fatal(err)
		}

		victim := pods.Items[rand.Intn(len(pods.Items))]

		fmt.Printf("Killing pod %s/%s\n", victim.Namespace, victim.Name)

		err = client.Core().Pods(victim.Namespace).Delete(victim.Name, nil)
		if err != nil {
			log.Fatal(err)
		}

		time.Sleep(interval)
	}
}

func newClient() (*kubernetes.Clientset, error) {
	var (
		config *rest.Config
		err    error
	)

	if inCluster {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		config = &rest.Config{
			Host: "http://127.0.0.1:8001",
		}
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

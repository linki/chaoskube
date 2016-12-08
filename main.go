package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
)

var (
	interval  time.Duration
	inCluster bool
	deploy    bool
)

func init() {
	kingpin.Flag("interval", "Interval between Pod terminations").Short('i').Default("10m").DurationVar(&interval)
	kingpin.Flag("in-cluster", "If true, finds the Kubernetes cluster from the environment").Short('c').BoolVar(&inCluster)
	kingpin.Flag("deploy", "If true, deploys chaoskube in the target cluster").Short('d').BoolVar(&deploy)
}

func main() {
	kingpin.Parse()

	client, err := newClient()
	if err != nil {
		log.Fatal(err)
	}

	if deploy {
		fmt.Printf("Deploying Chaoskube\n")

		_, err := client.Extensions().Deployments(v1.NamespaceDefault).Create(manifest)
		if err != nil {
			log.Fatal(err)
		}

		os.Exit(0)
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

var manifest = &v1beta1.Deployment{
	TypeMeta: unversioned.TypeMeta{
		APIVersion: "extensions/v1beta1",
		Kind:       "Deployment",
	},
	ObjectMeta: v1.ObjectMeta{
		Name: "chaoskube",
		Labels: map[string]string{
			"app":      "chaoskube",
			"heritage": "chaoskube",
		},
	},
	Spec: v1beta1.DeploymentSpec{
		Template: v1.PodTemplateSpec{
			ObjectMeta: v1.ObjectMeta{
				Labels: map[string]string{
					"app": "chaoskube",
				},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					v1.Container{
						Name:  "chaoskube",
						Image: "quay.io/linki/chaoskube:v0.2.2",
						Args: []string{
							"--in-cluster",
							"--interval=10m",
						},
					},
				},
			},
		},
	},
}

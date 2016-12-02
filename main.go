package main

import (
	"fmt"
	"math/rand"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/rest"
)

var (
	interval time.Duration
)

func init() {
	kingpin.Flag("interval", "Interval between Pod terminations").Short('i').Default("10m").DurationVar(&interval)
}

func main() {
	kingpin.Parse()

	config := &rest.Config{
		Host: "http://127.0.0.1:8001",
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	for {
		pods, err := clientset.Core().Pods("").List(api.ListOptions{})
		if err != nil {
			panic(err.Error())
		}

		victim := pods.Items[rand.Intn(len(pods.Items))]

		fmt.Printf("Killing pod %s/%s\n", victim.Namespace, victim.Name)

		err = clientset.Core().Pods(victim.Namespace).Delete(victim.Name, nil)
		if err != nil {
			panic(err.Error())
		}

		time.Sleep(interval)
	}
}

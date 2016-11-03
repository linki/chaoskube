package main

import (
	"fmt"
	"time"
	"math/rand"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/rest"
)

func main() {
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

		time.Sleep(10 * time.Second)
	}
}

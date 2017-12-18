package main

import (
	"fmt"
	"os"

	"bitbucket.org/bdsengineering/perceptor/clustermanager"
)

func main() {
	namespace := os.Args[1]
	name := os.Args[2]

	kubeClient, err := clustermanager.NewKubeClient()
	if err != nil {
		panic(err)
	}

	pod, err := kubeClient.GetPod(namespace, name)
	if err != nil {
		panic(err)
	}

	// fmt.Printf("found your pod: %v\n\nannotations: %v\n", pod, pod.Annotations)
	// fmt.Printf("found your pod: %v\n", pod.Annotations)
	fmt.Print("found your pod:\n")
	for key, value := range pod.Annotations {
		fmt.Printf("%s: %s\n", key, value)
	}
	fmt.Print("\n")
}

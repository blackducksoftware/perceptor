package main

import (
	"fmt"
	"os"
	"os/user"
	"sort"

	"bitbucket.org/bdsengineering/perceptor/clustermanager"
	"github.com/prometheus/common/log"
)

func main() {
	namespace := os.Args[1]
	name := os.Args[2]

	usr, err := user.Current()
	if err != nil {
		log.Errorf("unable to find current user's home dir: %s", err.Error())
		panic(err)
	}

	hubHost := "34.227.56.110.xip.io"
	// hubHost := "localhost"
	kubeconfigPath := usr.HomeDir + "/.kube/config"
	clusterMasterURL := "https://" + hubHost + ":8443"

	kubeClient, err := clustermanager.NewKubeClient(clusterMasterURL, kubeconfigPath)
	if err != nil {
		panic(err)
	}

	pod, err := kubeClient.GetPod(namespace, name)
	if err != nil {
		panic(err)
	}

	// fmt.Printf("found your pod: %v\n\nannotations: %v\n", pod, pod.Annotations)
	// fmt.Printf("found your pod: %v\n", pod.Annotations)
	// fmt.Print("found your pod:\n")
	keys := []string{}
	for key := range pod.Annotations {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := pod.Annotations[key]
		fmt.Printf("%s: %s\n", key, value)
	}
	// fmt.Print("\n")
}

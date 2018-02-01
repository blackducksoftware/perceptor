package main

import (
	"encoding/json"
	"fmt"

	"bitbucket.org/bdsengineering/perceptor/pkg/clustermanager"
)

func main() {
	hubHost := "34.227.56.110.xip.io"
	masterURL := "https://" + hubHost + ":8443"
	kubeconfigPath := "/Users/mfenwick/.kube/config"
	client, err := clustermanager.NewKubeClient(masterURL, kubeconfigPath)
	if err != nil {
		panic(err)
	}
	pods, err := client.GetAllPods()
	if err != nil {
		panic(err)
	}
	jsonBytes, err := json.Marshal(pods)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", string(jsonBytes))
}

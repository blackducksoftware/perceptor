package main

import (
	"fmt"
	"os"
	"os/user"

	"bitbucket.org/bdsengineering/perceptor/pkg/clustermanager"
	"github.com/prometheus/common/log"
)

func main() {
	namespace := os.Args[1]

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

	pods, err := kubeClient.GetPodsForNamespace(namespace)
	if err != nil {
		panic(err)
	}
	for _, pod := range pods {
		annotations, err := kubeClient.GetBlackDuckPodAnnotations(namespace, pod.Name)
		if err == nil {
			if len(annotations.ImageAnnotations) > 0 {
				fmt.Printf("Images for pod %s:\n", pod.Name)
				for image, info := range annotations.ImageAnnotations {
					fmt.Printf("  image %s\n", image)
					fmt.Printf("    %d vulnerabilities\n", info.VulnerabilityCount)
					fmt.Printf("    %d policy violations\n", info.PolicyViolationCount)
				}
			} else {
				fmt.Printf("No images for pod %s\n", pod.Name)
			}
			fmt.Println()
		} else {
			fmt.Printf("No vulnerability info for pod%s\n%v\n%s\n\n", pod.Name, pod.Annotations, err.Error())
		}
	}
}

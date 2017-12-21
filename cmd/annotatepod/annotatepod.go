package main

import (
	"os"
	"os/user"

	"bitbucket.org/bdsengineering/perceptor/clustermanager"
	"github.com/prometheus/common/log"
)

func main() {
	namespace := os.Args[1]
	name := os.Args[2]
	key := os.Args[3]

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

	// some debug code
	// allPods, err := kubeClient.GetPods()
	// if err != nil {
	// 	panic(err)
	// }
	// for _, p := range allPods {
	// 	fmt.Printf("found a pod: %s:%s  (%v)\n\n", p.Namespace, p.Name, p.Annotations) //p)
	// }
	// okay, done with the debug

	pod, err := kubeClient.GetPod(namespace, name)
	if err != nil {
		panic(err)
	}
	annotations := pod.Annotations
	if len(os.Args) > 4 {
		value := os.Args[4]
		annotations[key] = value
	} else {
		delete(annotations, key)
	}
	// not sure if this next line is necessary
	pod.Annotations = annotations
	kubeClient.UpdatePod(pod)
}

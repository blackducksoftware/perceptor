package main

import (
	"os"

	"bitbucket.org/bdsengineering/perceptor/clustermanager"
)

func splat4(vs []string) (string, string, string, string) {
	return vs[0], vs[1], vs[2], vs[3]
}

func main() {
	namespace, name, key, value := splat4(os.Args[1:])
	kubeClient, err := clustermanager.NewKubeClient()
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
	annotations[key] = value
	// not sure if this next line is necessary
	pod.Annotations = annotations
	kubeClient.UpdatePod(pod)
}

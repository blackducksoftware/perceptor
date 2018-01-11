package main

import (
	"os"

	core "bitbucket.org/bdsengineering/perceptor/pkg/core"
)

func main() {
	var kubeconfigPath string
	if len(os.Args) >= 2 {
		kubeconfigPath = os.Args[1]
	} else {
		kubeconfigPath = "~/.kube/config"
	}

	core.RunLocally(kubeconfigPath)

	// hack to prevent main from returning
	select {}
}

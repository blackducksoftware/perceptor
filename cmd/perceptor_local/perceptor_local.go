package main

import (
	"os"
	"os/user"

	core "bitbucket.org/bdsengineering/perceptor/pkg/core"
	"github.com/prometheus/common/log"
)

func main() {
	var kubeconfigPath string
	if len(os.Args) >= 2 {
		kubeconfigPath = os.Args[1]
	} else {
		usr, err := user.Current()
		if err != nil {
			log.Errorf("unable to find current user's home dir: %s", err.Error())
			panic(err)
		}

		kubeconfigPath = usr.HomeDir + "/.kube/config"
	}

	core.RunLocally(kubeconfigPath)

	// hack to prevent main from returning
	select {}
}

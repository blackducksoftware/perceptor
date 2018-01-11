package main

import (
	core "bitbucket.org/bdsengineering/perceptor/pkg/core"
)

func main() {
	core.RunFromInsideCluster()

	// hack to prevent main from returning
	select {}
}

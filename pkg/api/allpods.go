package api

import "bitbucket.org/bdsengineering/perceptor/pkg/common"

type AllPods struct {
	Pods []common.Pod
}

func NewAllPods(pods []common.Pod) *AllPods {
	return &AllPods{Pods: pods}
}

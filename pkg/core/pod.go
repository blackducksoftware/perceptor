package core

import (
	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
)

type Pod struct {
	Name       string
	UID        string
	Namespace  string
	Containers []Container
}

func (pod *Pod) hasImage(image common.Image) bool {
	for _, cont := range pod.Containers {
		if cont.Image == image {
			return true
		}
	}
	return false
}

func NewPod(name string, uid string, namespace string, containers []Container) *Pod {
	return &Pod{
		Name:       name,
		UID:        uid,
		Namespace:  namespace,
		Containers: containers,
	}
}

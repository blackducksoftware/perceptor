package clustermanager

import (
	common "bitbucket.org/bdsengineering/perceptor/pkg/common"

	"k8s.io/api/core/v1"
)

func NewPod(kubePod *v1.Pod) *Pod {
	return &Pod{
		Name:        kubePod.Name,
		Namespace:   kubePod.Namespace,
		Annotations: kubePod.Annotations,
		Spec:        *NewSpec(&kubePod.Spec),
		UID:         string(kubePod.UID)}
}

func NewSpec(kubeSpec *v1.PodSpec) *Spec {
	containers := []Container{}
	for _, kubeCont := range kubeSpec.Containers {
		containers = append(containers, *NewContainer(&kubeCont))
	}
	return &Spec{Containers: containers}
}

func NewContainer(kubeCont *v1.Container) *Container {
	return &Container{
		Image: common.Image(kubeCont.Image),
		Name:  kubeCont.Name}
}

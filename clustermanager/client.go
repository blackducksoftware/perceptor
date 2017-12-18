package clustermanager

import "k8s.io/api/core/v1"

// Client provides the BlackDuck-specific interface to a cluster
type Client interface {
	// TODO add funcs for start, stop monitoring pods?
	ClearBlackDuckPodAnnotations(pod *v1.Pod) error
	GetBlackDuckPodAnnotations(pod *v1.Pod) (*BlackDuckAnnotations, error)
	SetBlackDuckPodAnnotations(pod *v1.Pod, bdAnnotations BlackDuckAnnotations) error
	// event channels
	PodAdd() <-chan AddPod
	PodUpdate() <-chan UpdatePod
	PodDelete() <-chan DeletePod
}

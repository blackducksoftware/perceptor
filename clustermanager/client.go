package clustermanager

// Client provides the BlackDuck-specific interface to a cluster
type Client interface {
	// TODO add funcs for start, stop monitoring pods?
	ClearBlackDuckPodAnnotations(pod Pod) error
	GetBlackDuckPodAnnotations(pod Pod) (*BlackDuckAnnotations, error)
	SetBlackDuckPodAnnotations(pod Pod, bdAnnotations BlackDuckAnnotations) error
	// event channels
	PodAdd() <-chan AddPod
	PodUpdate() <-chan UpdatePod
	PodDelete() <-chan DeletePod
}

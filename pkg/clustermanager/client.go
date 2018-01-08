package clustermanager

// Client provides the BlackDuck-specific interface to a cluster
type Client interface {
	// TODO add funcs for start, stop monitoring pods?
	ClearBlackDuckPodAnnotations(namespace string, name string) error
	GetBlackDuckPodAnnotations(namespace string, name string) (*BlackDuckAnnotations, error)
	SetBlackDuckPodAnnotations(namespace string, name string, bdAnnotations BlackDuckAnnotations) error
	// event channels
	PodAdd() <-chan AddPod
	PodUpdate() <-chan UpdatePod
	PodDelete() <-chan DeletePod
}

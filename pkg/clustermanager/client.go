package clustermanager

// Client provides the BlackDuck-specific interface to a cluster
type Client interface {
	ClearBlackDuckPodAnnotations(namespace string, name string) error
	GetBlackDuckPodAnnotations(namespace string, name string) (*BlackDuckAnnotations, error)
	SetBlackDuckPodAnnotations(namespace string, name string, bdAnnotations BlackDuckAnnotations) error

	// TODO change these types to use the ones in the common package
	PodAdd() <-chan AddPod
	PodUpdate() <-chan UpdatePod
	PodDelete() <-chan DeletePod
}

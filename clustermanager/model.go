package clustermanager

import "k8s.io/api/core/v1"

// AddPod is a wrapper around the kubernetes API object
type AddPod struct {
	New v1.Pod
}

// UpdatePod holds the old and new versions of the changed pod
type UpdatePod struct {
	Old v1.Pod
	New v1.Pod
}

// DeletePod holds the id of the deleted pod
type DeletePod struct {
	ID string
}

type BlackDuckAnnotations struct {
	Containers map[string]Container
	// TODO remove KeyVals, this is just for testing, to be able
	// to jam random stuff somewhere
	KeyVals map[string]string
}

func NewBlackDuckAnnotations() *BlackDuckAnnotations {
	return &BlackDuckAnnotations{
		Containers: make(map[string]Container),
		KeyVals:    make(map[string]string),
	}
}

type Container struct {
	Image string
	// vulnerabilities ?
}

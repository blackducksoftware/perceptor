package clustermanager

import "fmt"

// AddPod is a wrapper around the kubernetes API object
type AddPod struct {
	New Pod
}

// UpdatePod holds the old and new versions of the changed pod
type UpdatePod struct {
	Old Pod
	New Pod
}

// DeletePod holds the id of the deleted pod
type DeletePod struct {
	ID string
}

type Pod struct {
	Name        string
	Namespace   string
	Annotations map[string]string
	Spec        Spec
	UID         string
}

// TODO this is currently being used as a unique identifier;
// is that a bad idea?
func (pod *Pod) GetKey() string {
	return fmt.Sprintf("%s:%s", pod.Namespace, pod.Name)
}

type Spec struct {
	Containers []Container
}

type Container struct {
	Image string
	Name  string
}

type BlackDuckAnnotations struct {
	// TODO Container isn't the right type; but what is?
	ContainerAnnotations map[string]ContainerAnnotation
	// TODO remove KeyVals, this is just for testing, to be able
	// to jam random stuff somewhere
	KeyVals map[string]string
}

func NewBlackDuckAnnotations() *BlackDuckAnnotations {
	return &BlackDuckAnnotations{
		ContainerAnnotations: make(map[string]ContainerAnnotation),
		KeyVals:              make(map[string]string),
	}
}

type ContainerAnnotation struct {
	Image string
	// TODO vulnerabilities ?
}

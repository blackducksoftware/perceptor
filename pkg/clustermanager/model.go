package clustermanager

import (
	"fmt"

	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
)

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
	Image common.Image
	Name  string
}

package clustermanager

import (
	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
)

// AddPod is a wrapper around the kubernetes API object
type AddPod struct {
	New common.Pod
}

// UpdatePod holds the old and new versions of the changed pod
type UpdatePod struct {
	Old common.Pod
	New common.Pod
}

// DeletePod holds the name of the deleted pod as a string of namespace/name
type DeletePod struct {
	QualifiedName string
}

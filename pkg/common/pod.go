package common

import "fmt"

type Pod struct {
	Name       string
	UID        string
	Namespace  string
	Containers []Container
	// TODO probably need to add Annotations map[string]string
}

func (pod *Pod) QualifiedName() string {
	return fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
}

func (pod *Pod) hasImage(image Image) bool {
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

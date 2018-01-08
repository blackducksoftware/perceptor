package core

type Pod struct {
	Name       string
	UID        string
	Namespace  string
	Containers []Container
}

func (pod *Pod) hasImage(image string) bool {
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

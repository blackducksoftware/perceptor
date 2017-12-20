package core

type ImageScanComplete struct {
	AffectedPods []Pod
	Image        string
	ScanResults  ScanResults
}

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

type Container struct {
	Image string
	Name  string
}

func NewContainer(image string, name string) *Container {
	return &Container{
		Image: image,
		Name:  name,
	}
}

type Image struct {
	ScanStatus  ScanStatus
	ScanResults *ScanResults
	// Name string
}

func NewImage() *Image {
	return &Image{
		ScanStatus:  ScanStatusNotScanned,
		ScanResults: nil,
	}
}

type ScanResults struct {
	OverallStatus        string
	VulnerabilityCount   int
	PolicyViolationCount int
	// TODO also add:
	// scanner version
	// hub version
	// project URL
}

type ScanStatus int

const (
	ScanStatusNotScanned     ScanStatus = iota
	ScanStatusInProgress     ScanStatus = iota
	ScanStatusAnnotatingPods ScanStatus = iota // or should it be AnnotatingContainers? TODO
	ScanStatusComplete       ScanStatus = iota // TODO may need to rethink this, since we can
	// always pull from kube to check the annotations, and see if it's missing anything
	// that we have TODO
)

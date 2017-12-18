package model

import (
	"bitbucket.org/bdsengineering/perceptor/clustermanager"
	log "github.com/sirupsen/logrus"
)

type Pod struct {
	Name       string
	UID        string
	Namespace  string
	Containers []Container
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
	Vulnerabilities []Vulnerability
	ScanStatus      ScanStatus
	// Name string
}

func NewImage(vulnerabilities []Vulnerability) *Image {
	return &Image{
		Vulnerabilities: vulnerabilities,
		ScanStatus:      NotScanned,
	}
}

type Vulnerability struct {
	Name string
}

type ScanStatus int

const (
	NotScanned         ScanStatus = iota
	ScanningInProgress ScanStatus = iota
	AnnotatingPods     ScanStatus = iota // or should it be AnnotatingContainers? TODO
	Scanned            ScanStatus = iota // TODO may need to rethink this, since we can
	// always pull from kube to check the annotations, and see if it's missing anything
	// that we have TODO
)

// VulnerabilityCache is the root model
type VulnerabilityCache struct {
	// TODO add lock here, and lock in every
	// "method" call down below

	// TODO what do we do about pod namespaces?
	Pods   map[string]Pod
	Images map[string]Image

	ImagesToBeScanned chan string

	// ?? queue of things waiting to be picked up from hub
	// ?? queue of things waiting to be sent to API server
}

// What should this be set to?  ... who knows
// This is in place of a queue implementation to keep track
// of the images that need to be scanned by the hub.
// Maybe we can change that later.
// Let's hope 300 is large enough to keep it from blocking,
// but doesn't waste tons of memory.
var pendingImageScanLimit = 300

func NewVulnerabilityCache() *VulnerabilityCache {
	return &VulnerabilityCache{
		Pods:              make(map[string]Pod),
		Images:            make(map[string]Image),
		ImagesToBeScanned: make(chan string, 300),
	}
}

// AddPod returns true if it hasn't yet seen the pod,
// and false if the pod has already been added.
// It extract the containers and images from the pod,
// adding them into the cache.
func (cache *VulnerabilityCache) AddPod(newPod clustermanager.Pod) bool {
	_, ok := cache.Pods[newPod.Name]
	if ok {
		// TODO should we update the cache?
		// skipping for now
		return false
	}
	log.Info("about to add pod: %v", newPod)
	containers := []Container{}
	for _, newCont := range newPod.Spec.Containers {
		addedCont := NewContainer(newCont.Image, newCont.Name)
		containers = append(containers, *addedCont)
		_, hasImage := cache.Images[newCont.Image]
		if !hasImage {
			addedImage := NewImage([]Vulnerability{})
			cache.Images[newCont.Image] = *addedImage
			cache.ImagesToBeScanned <- newCont.Image
		}
	}
	addedPod := NewPod(newPod.Name, string(newPod.UID), newPod.Namespace, containers)
	cache.Pods[newPod.Name] = *addedPod
	return true
}

func (model *VulnerabilityCache) addPod(podUID string, namespace string) *Pod {
	pod, ok := model.Pods[podUID]
	if ok {
		return &pod
	}
	pod = *new(Pod)
	pod.Namespace = namespace
	model.Pods[podUID] = pod
	return &pod
}

/* TODO are these methods relevant anymore?
func (model *VulnerabilityCache) addContainer(podUID string, namespace string, containerImage string) *Container {
	cont, ok := model.Containers[containerImage]
	if ok {
		return &cont
	}
	pod := model.addPod(podUID, namespace)
	pod.ContainerImages = append(pod.ContainerImages, containerImage)
	cont = *NewContainer()
	model.Containers[containerImage] = cont
	return &cont
}

func (model *VulnerabilityCache) hasContainerOfImage(image string) bool {
	_, ok := model.Containers[image]
	return ok
}

func (model *VulnerabilityCache) startPodScan(image string) {
	cont, ok := model.Containers[image]
	if ok {
		cont.ScanStatus = ScanningInProgress
	}
}

func (model *VulnerabilityCache) finishPodScan(image string) {
	cont, ok := model.Containers[image]
	if ok {
		cont.ScanStatus = AnnotatingPods
	}
}

func (model *VulnerabilityCache) finishAnnotation(image string) {
	cont, ok := model.Containers[image]
	if ok {
		cont.ScanStatus = Scanned
	}
}
*/

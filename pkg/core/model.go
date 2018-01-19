package core

import (
	"fmt"

	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
	"bitbucket.org/bdsengineering/perceptor/pkg/scanner"
	log "github.com/sirupsen/logrus"
)

// Model is the root of the core model
type Model struct {
	// map of "<namespace>/<name>" to pod
	Pods           map[string]common.Pod
	Images         map[common.Image]*ImageScanResults
	ImageScanQueue []common.Image
}

func NewModel() *Model {
	return &Model{
		Pods:           make(map[string]common.Pod),
		Images:         make(map[common.Image]*ImageScanResults),
		ImageScanQueue: []common.Image{}}
}

// DeletePod removes the record of a pod, but does not affect images.
func (vc *Model) DeletePod(podName string) {
	delete(vc.Pods, podName)
}

// AddPod should be called when receiving new pods from the
// clustermanager.  It returns true if it hasn't yet seen the pod,
// and false if the pod has already been added.
// It extract the containers and images from the pod,
// adding them into the cache.
func (vc *Model) AddPod(newPod common.Pod) bool {
	_, ok := vc.Pods[newPod.QualifiedName()]
	if ok {
		// TODO should we update the cache?
		// skipping for now
		return false
	}
	log.Infof("about to add pod: UID %s, qualfied name %s", newPod.UID, newPod.QualifiedName())
	for _, newCont := range newPod.Containers {
		_, hasImage := vc.Images[newCont.Image]
		if !hasImage {
			addedImage := NewImageScanResults()
			vc.Images[newCont.Image] = addedImage
			log.Infof("adding image %s to image scan queue", newCont.Image)
			vc.addImageToQueue(newCont.Image)
		} else {
			log.Infof("not adding image %s to image scan queue, already have in cache", newCont.Image)
		}
	}
	log.Infof("done adding containers+images from pod %s -- %s", newPod.UID, newPod.QualifiedName())
	vc.Pods[newPod.QualifiedName()] = newPod
	return true
}

func (vc *Model) AddScanResultsFromProject(project scanner.Project) error {
	for _, version := range project.Versions {
		err := vc.addScanResult(version)
		if err != nil {
			return err
		}
	}
	return nil
}

// image state transitions

func (vc *Model) safeGet(image common.Image) *ImageScanResults {
	results, ok := vc.Images[image]
	if !ok {
		message := fmt.Sprintf("expected to already have image %s, but did not", image.Name())
		log.Error(message)
		panic(message)
	}
	return results
}

func (vc *Model) addImageToQueue(image common.Image) {
	results := vc.safeGet(image)
	switch results.ScanStatus {
	case ScanStatusNotScanned, ScanStatusError:
		break
	default:
		message := fmt.Sprintf("cannot add image %s to queue, status is neither NotScanned nor Error (%d)", image.Name(), results.ScanStatus)
		log.Error(message)
		panic(message)
	}
	results.ScanStatus = ScanStatusInQueue
	vc.ImageScanQueue = append(vc.ImageScanQueue, image)
}

func (vc *Model) getNextImageFromQueue() *common.Image {
	if len(vc.ImageScanQueue) == 0 {
		return nil
	}

	first := vc.ImageScanQueue[0]
	results := vc.safeGet(first)
	if results.ScanStatus != ScanStatusInQueue {
		message := fmt.Sprintf("can not start scanning image %s, status is not InQueue (%d)", first.Name(), results.ScanStatus)
		log.Errorf(message)
		panic(message)
	}

	results.ScanStatus = ScanStatusRunningScanClient
	vc.ImageScanQueue = vc.ImageScanQueue[1:]
	return &first
}

func (vc *Model) errorRunningScanClient(image common.Image) {
	results := vc.safeGet(image)
	if results.ScanStatus != ScanStatusRunningScanClient {
		message := fmt.Sprintf("can not error out scan client for image %s, scan client not in progress (%d)", image.Name(), results.ScanStatus)
		log.Errorf(message)
		panic(message)
	}
	results.ScanStatus = ScanStatusError
	// for now, just readd the image to the queue upon error
	vc.addImageToQueue(image)
}

func (vc *Model) finishRunningScanClient(image common.Image) {
	results := vc.safeGet(image)
	if results.ScanStatus != ScanStatusRunningScanClient {
		message := fmt.Sprintf("can not finish running scan client for image %s, scan client not in progress (%d)", image.Name(), results.ScanStatus)
		log.Errorf(message)
		panic(message)
	}
	results.ScanStatus = ScanStatusRunningHubScan
}

// func (vc *Model) finishRunningHubScan(image common.Image) {
// 	results := vc.safeGet(image)
// 	if results.ScanStatus != ScanStatusRunningHubScan {
// 		message := fmt.Sprintf("can not finish running hub scan for image %s, scan not in progress (%d)", image.Name(), results.ScanStatus)
// 		log.Errorf(message)
// 		panic(message)
// 	}
// 	results.ScanStatus = ScanStatusComplete
// }

// additional methods

func (vc *Model) inProgressScanJobs() []common.Image {
	inProgressImages := []common.Image{}
	for image, results := range vc.Images {
		switch results.ScanStatus {
		case ScanStatusRunningScanClient, ScanStatusRunningHubScan:
			inProgressImages = append(inProgressImages, image)
		default:
			break
		}
	}
	return inProgressImages
}

func (vc *Model) inProgressScanCount() int {
	return len(vc.inProgressScanJobs())
}

func (vc *Model) addScanResult(version scanner.Version) error {
	image := common.Image(version.VersionName)

	// add scan results into cache
	scanResults, ok := vc.Images[image]
	if !ok {
		return fmt.Errorf("expected to already have image %s, but did not", image.Name())
	}

	if scanResults.ScanResults == nil {
		scanResults.ScanResults = NewScanResults()
	}

	scanResults.ScanResults.VulnerabilityCount = version.RiskProfile.HighRiskVulnerabilityCount()
	scanResults.ScanResults.OverallStatus = version.PolicyStatus.OverallStatus
	scanResults.ScanResults.PolicyViolationCount = version.PolicyStatus.ViolationCount()

	if version.IsImageScanDone() {
		scanResults.ScanStatus = ScanStatusComplete
	}

	return nil
}

func (vc *Model) scanResults(podName string) (*ScanResults, error) {
	pod, ok := vc.Pods[podName]
	if !ok {
		return nil, fmt.Errorf("could not find pod of name %s in cache", podName)
	}

	overallStatus := ""
	policyViolationCount := 0
	vulnerabilityCount := 0
	for _, container := range pod.Containers {
		imageScanResults, ok := vc.Images[container.Image]
		if !ok {
			continue
		}
		if imageScanResults.ScanStatus != ScanStatusComplete {
			continue
		}
		if imageScanResults.ScanResults == nil {
			continue
		}
		policyViolationCount += imageScanResults.ScanResults.PolicyViolationCount
		vulnerabilityCount += imageScanResults.ScanResults.VulnerabilityCount
		// TODO what's the right way to combine all the 'OverallStatus' values
		//   from the individual image scans?
		if imageScanResults.ScanResults.OverallStatus != "NOT_IN_VIOLATION" {
			overallStatus = imageScanResults.ScanResults.OverallStatus
		}
	}
	return &ScanResults{
		OverallStatus:        overallStatus,
		PolicyViolationCount: policyViolationCount,
		VulnerabilityCount:   vulnerabilityCount,
	}, nil
}

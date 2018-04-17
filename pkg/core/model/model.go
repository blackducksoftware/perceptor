/*
Copyright (C) 2018 Synopsys, Inc.

Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements. See the NOTICE file
distributed with this work for additional information
regarding copyright ownership. The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied. See the License for the
specific language governing permissions and limitations
under the License.
*/

package model

import (
	"fmt"

	"github.com/blackducksoftware/perceptor/pkg/hub"
	log "github.com/sirupsen/logrus"
)

// Model is the root of the core model
type Model struct {
	// Pods is a map of "<namespace>/<name>" to pod
	Pods                map[string]Pod
	Images              map[DockerImageSha]*ImageInfo
	ImageScanQueue      []DockerImageSha
	ImageHubCheckQueue  []DockerImageSha
	ConcurrentScanLimit int
	Config              *Config
	HubVersion          string
}

func NewModel(config *Config, hubVersion string) *Model {
	return &Model{
		Pods:                make(map[string]Pod),
		Images:              make(map[DockerImageSha]*ImageInfo),
		ImageScanQueue:      []DockerImageSha{},
		ImageHubCheckQueue:  []DockerImageSha{},
		ConcurrentScanLimit: config.ConcurrentScanLimit,
		Config:              config,
		HubVersion:          hubVersion}
}

// DeletePod removes the record of a pod, but does not affect images.
func (model *Model) DeletePod(podName string) {
	delete(model.Pods, podName)
}

// AddPod adds a pod and all the images in a pod to the model.
// If the pod is already present in the model, it will be removed
// and a new one created in its place.
// The key is the combination of the pod's namespace and name.
// It extracts the containers and images from the pod,
// adding them into the cache.
func (model *Model) AddPod(newPod Pod) {
	log.Debugf("about to add pod: UID %s, qualified name %s", newPod.UID, newPod.QualifiedName())
	if len(newPod.Containers) == 0 {
		recordEvent("adding pod with 0 containers")
		log.Warnf("adding pod %s with 0 containers: %+v", newPod.QualifiedName(), newPod)
	}
	for _, newCont := range newPod.Containers {
		model.AddImage(newCont.Image)
	}
	log.Debugf("done adding containers+images from pod %s -- %s", newPod.UID, newPod.QualifiedName())
	model.Pods[newPod.QualifiedName()] = newPod
}

// AddImage adds an image to the model, adding it to the queue for hub checking.
func (model *Model) AddImage(image Image) {
	added := model.createImage(image)
	if added {
		model.SetImageScanStatus(image.Sha, ScanStatusInHubCheckQueue)
	}
}

// image state transitions

func (model *Model) leaveState(sha DockerImageSha, state ScanStatus) {
	switch state {
	case ScanStatusUnknown:
		break
	case ScanStatusInHubCheckQueue:
		model.removeImageFromHubCheckQueue(sha)
	case ScanStatusInQueue:
		model.removeImageFromScanQueue(sha)
	case ScanStatusRunningScanClient:
		break
	case ScanStatusRunningHubScan:
		break
	case ScanStatusComplete:
		break
	}
}

func (model *Model) enterState(sha DockerImageSha, state ScanStatus) {
	switch state {
	case ScanStatusUnknown:
		break
	case ScanStatusInHubCheckQueue:
		model.addImageToHubCheckQueue(sha)
	case ScanStatusInQueue:
		model.addImageToScanQueue(sha)
	case ScanStatusRunningScanClient:
		break
	case ScanStatusRunningHubScan:
		break
	case ScanStatusComplete:
		break
	}
}

func (model *Model) setImageScanStatusForSha(sha DockerImageSha, newScanStatus ScanStatus) error {
	imageInfo, ok := model.Images[sha]
	if !ok {
		return fmt.Errorf("can not set scan status for sha %s, sha not found", string(sha))
	}

	log.Debugf("attempting to transition sha %s from %s to %s", sha, imageInfo.ScanStatus, newScanStatus)
	isLegal := IsLegalTransition(imageInfo.ScanStatus, newScanStatus)
	recordStateTransition(imageInfo.ScanStatus, newScanStatus, isLegal)
	if !isLegal {
		return fmt.Errorf("illegal image state transition from %s to %s for sha %s", imageInfo.ScanStatus, newScanStatus, sha)
	}

	model.leaveState(sha, imageInfo.ScanStatus)
	model.enterState(sha, newScanStatus)
	imageInfo.setScanStatus(newScanStatus)

	return nil
}

// createImage adds the image to the model, but not to the scan queue
func (model *Model) createImage(image Image) bool {
	_, hasImage := model.Images[image.Sha]
	if !hasImage {
		newInfo := NewImageInfo(image.Sha, image.Name)
		model.Images[image.Sha] = newInfo
		log.Debugf("added image %s to model", image.HumanReadableName())
	} else {
		log.Debugf("not adding image %s to model, already have in cache", image.HumanReadableName())
	}
	return !hasImage
}

// Be sure that `sha` is in `model.Images` before calling this method
func (model *Model) unsafeGet(sha DockerImageSha) *ImageInfo {
	results, ok := model.Images[sha]
	if !ok {
		message := fmt.Sprintf("expected to already have image %s, but did not", string(sha))
		log.Error(message)
		panic(message)
	}
	return results
}

// Adding and removing from scan queues.  These are "unsafe" calls and should
// only be called by methods that have already checked all the error conditions
// (things are in the right state, things that are expected to be present are
// actually present, etc.)

func (model *Model) addImageToHubCheckQueue(sha DockerImageSha) {
	model.ImageHubCheckQueue = append(model.ImageHubCheckQueue, sha)
}

func (model *Model) removeImageFromHubCheckQueue(sha DockerImageSha) {
	index := -1
	for i := 0; i < len(model.ImageHubCheckQueue); i++ {
		if model.ImageHubCheckQueue[i] == sha {
			index = i
			break
		}
	}
	if index < 0 {
		panic(fmt.Errorf("unable to remove sha %s from hub check queue, not found", string(sha)))
	}

	model.ImageHubCheckQueue = append(model.ImageHubCheckQueue[:index], model.ImageHubCheckQueue[index+1:]...)
}

func (model *Model) addImageToScanQueue(sha DockerImageSha) {
	model.ImageScanQueue = append(model.ImageScanQueue, sha)
}

func (model *Model) removeImageFromScanQueue(sha DockerImageSha) {
	index := -1
	for i := 0; i < len(model.ImageScanQueue); i++ {
		if model.ImageScanQueue[i] == sha {
			index = i
			break
		}
	}
	if index < 0 {
		panic(fmt.Errorf("unable to remove sha %s from hub check queue, not found", string(sha)))
	}

	model.ImageScanQueue = append(model.ImageScanQueue[:index], model.ImageScanQueue[index+1:]...)
}

// "Public" methods

func (model *Model) SetImageScanStatus(sha DockerImageSha, newScanStatus ScanStatus) {
	err := model.setImageScanStatusForSha(sha, newScanStatus)
	if err != nil {
		imageInfo, ok := model.Images[sha]
		statusString := "sha not found"
		if ok {
			statusString = imageInfo.ScanStatus.String()
		}
		log.Errorf("unable to transition image state for sha %s from <%s> to %s", sha, statusString, newScanStatus)
	}
}

func (model *Model) GetNextImageFromHubCheckQueue() *Image {
	if len(model.ImageHubCheckQueue) == 0 {
		log.Debug("hub check queue empty")
		return nil
	}

	first := model.ImageHubCheckQueue[0]
	image := model.unsafeGet(first).Image()

	return &image
}

func (model *Model) GetNextImageFromScanQueue() *Image {
	if model.InProgressScanCount() >= model.ConcurrentScanLimit {
		log.Debugf("max concurrent scan count reached, can't start a new scan -- %v", model.InProgressScans())
		return nil
	}

	if len(model.ImageScanQueue) == 0 {
		log.Debug("scan queue empty, can't start a new scan")
		return nil
	}

	first := model.ImageScanQueue[0]
	image := model.unsafeGet(first).Image()

	model.SetImageScanStatus(first, ScanStatusRunningScanClient)

	return &image
}

func (model *Model) FinishRunningScanClient(image *Image, scanClientError error) {
	_, ok := model.Images[image.Sha]

	// if we don't have this sha already, let's add it
	if !ok {
		log.Warnf("finish running scan client -- expected to already have image %s, but did not", string(image.Sha))
		model.createImage(*image)
	}

	scanStatus := ScanStatusRunningHubScan
	if scanClientError != nil {
		scanStatus = ScanStatusInQueue
		log.Errorf("error running scan client -- %s", scanClientError.Error())
	}

	model.SetImageScanStatus(image.Sha, scanStatus)
}

// additional methods

func (model *Model) InProgressScans() []DockerImageSha {
	inProgressShas := []DockerImageSha{}
	for sha, results := range model.Images {
		switch results.ScanStatus {
		case ScanStatusRunningScanClient, ScanStatusRunningHubScan:
			inProgressShas = append(inProgressShas, sha)
		default:
			break
		}
	}
	return inProgressShas
}

func (model *Model) InProgressScanCount() int {
	return len(model.InProgressScans())
}

func (model *Model) InProgressHubScans() []Image {
	inProgressHubScans := []Image{}
	for _, imageInfo := range model.Images {
		switch imageInfo.ScanStatus {
		case ScanStatusRunningHubScan:
			inProgressHubScans = append(inProgressHubScans, imageInfo.Image())
		}
	}
	return inProgressHubScans
}

func (model *Model) ScanResultsForPod(podName string) (*PodScan, error) {
	pod, ok := model.Pods[podName]
	if !ok {
		return nil, fmt.Errorf("could not find pod of name %s in cache", podName)
	}

	overallStatus := hub.PolicyStatusTypeNotInViolation
	policyViolationCount := 0
	vulnerabilityCount := 0
	for _, container := range pod.Containers {
		imageScan, err := model.ScanResultsForImage(container.Image.Sha)
		if err != nil {
			log.Errorf("unable to get scan results for image %s: %s", container.Image.Sha, err.Error())
			return nil, err
		}
		if imageScan == nil {
			return nil, nil
		}
		policyViolationCount += imageScan.PolicyViolations
		vulnerabilityCount += imageScan.Vulnerabilities
		imageScanOverallStatus := imageScan.OverallStatus
		if imageScanOverallStatus != hub.PolicyStatusTypeNotInViolation {
			overallStatus = imageScanOverallStatus
		}
	}
	podScan := &PodScan{
		OverallStatus:    overallStatus.String(),
		PolicyViolations: policyViolationCount,
		Vulnerabilities:  vulnerabilityCount}
	return podScan, nil
}

func (model *Model) ScanResultsForImage(sha DockerImageSha) (*ImageScan, error) {
	imageInfo, ok := model.Images[sha]
	if !ok {
		return nil, fmt.Errorf("could not find image of sha %s in cache", sha)
	}

	if imageInfo.ScanStatus != ScanStatusComplete {
		return nil, nil
	}
	if imageInfo.ScanResults == nil {
		return nil, fmt.Errorf("model inconsistency: could not find scan results for completed image %s", sha)
	}

	imageScan := &ImageScan{
		OverallStatus:    imageInfo.ScanResults.OverallStatus(),
		PolicyViolations: imageInfo.ScanResults.PolicyViolationCount(),
		Vulnerabilities:  imageInfo.ScanResults.VulnerabilityCount()}
	return imageScan, nil
}

func (model *Model) Metrics() *ModelMetrics {
	// number of images in each status
	statusCounts := make(map[ScanStatus]int)
	for _, imageResults := range model.Images {
		statusCounts[imageResults.ScanStatus]++
	}

	// number of containers per pod (as a histgram, but not a prometheus histogram ???)
	containerCounts := make(map[int]int)
	for _, pod := range model.Pods {
		containerCounts[len(pod.Containers)]++
	}

	// number of times each image is referenced from a pod's container
	imageCounts := make(map[Image]int)
	for _, pod := range model.Pods {
		for _, cont := range pod.Containers {
			imageCounts[cont.Image]++
		}
	}
	imageCountHistogram := make(map[int]int)
	for _, count := range imageCounts {
		imageCountHistogram[count]++
	}

	podStatus := map[string]int{}
	podPolicyViolations := map[int]int{}
	podVulnerabilities := map[int]int{}
	for podName, _ := range model.Pods {
		podScan, err := model.ScanResultsForPod(podName)
		if err != nil {
			log.Errorf("unable to get scan results for pod %s: %s", podName, err.Error())
			continue
		}
		if podScan != nil {
			podStatus[podScan.OverallStatus]++
			podPolicyViolations[podScan.PolicyViolations]++
			podVulnerabilities[podScan.Vulnerabilities]++
		} else {
			podStatus["Unknown"]++
			podPolicyViolations[-1]++
			podVulnerabilities[-1]++
		}
	}

	imageStatus := map[string]int{}
	imagePolicyViolations := map[int]int{}
	imageVulnerabilities := map[int]int{}
	for sha, imageInfo := range model.Images {
		if imageInfo.ScanStatus == ScanStatusComplete {
			imageScan := imageInfo.ScanResults
			if imageScan == nil {
				log.Errorf("found nil scan results for completed image %s", sha)
				continue
			}
			imageStatus[imageScan.OverallStatus().String()]++
			imagePolicyViolations[imageScan.PolicyViolationCount()]++
			imageVulnerabilities[imageScan.VulnerabilityCount()]++
		} else {
			imageStatus["Unknown"]++
			imagePolicyViolations[-1]++
			imageVulnerabilities[-1]++
		}
	}

	// TODO
	// number of images without a pod pointing to them
	return &ModelMetrics{
		ScanStatusCounts:      statusCounts,
		NumberOfImages:        len(model.Images),
		NumberOfPods:          len(model.Pods),
		ContainerCounts:       containerCounts,
		ImageCountHistogram:   imageCountHistogram,
		PodStatus:             podStatus,
		ImageStatus:           imageStatus,
		PodPolicyViolations:   podPolicyViolations,
		ImagePolicyViolations: imagePolicyViolations,
		PodVulnerabilities:    podVulnerabilities,
		ImageVulnerabilities:  imageVulnerabilities,
	}
}

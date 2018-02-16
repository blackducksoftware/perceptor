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

package core

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// Model is the root of the core model
type Model struct {
	// Pods is a map of "<namespace>/<name>" to pod
	Pods                map[string]Pod
	Images              map[DockerImageSha]*ImageInfo
	ImageScanQueue      []Image
	ImageHubCheckQueue  []Image
	ConcurrentScanLimit int
	Config              PerceptorConfig
}

func NewModel(concurrentScanLimit int, config PerceptorConfig) *Model {
	return &Model{
		Pods:                make(map[string]Pod),
		Images:              make(map[DockerImageSha]*ImageInfo),
		ImageScanQueue:      []Image{},
		ImageHubCheckQueue:  []Image{},
		ConcurrentScanLimit: concurrentScanLimit,
		Config:              config}
}

// DeletePod removes the record of a pod, but does not affect images.
func (model *Model) DeletePod(podName string) {
	delete(model.Pods, podName)
}

// AddPod adds a pod and all the images in a pod to the model.
// If the pod is already present in the model, it will be removed
// and a new one created in its place.
// The key is the combination of the pod's namespace and name.
// It extract the containers and images from the pod,
// adding them into the cache.
func (model *Model) AddPod(newPod Pod) {
	log.Debugf("about to add pod: UID %s, qualified name %s", newPod.UID, newPod.QualifiedName())
	for _, newCont := range newPod.Containers {
		model.AddImage(newCont.Image)
	}
	log.Debugf("done adding containers+images from pod %s -- %s", newPod.UID, newPod.QualifiedName())
	model.Pods[newPod.QualifiedName()] = newPod
}

// AddImage adds an image to the model, sets its status to NotScanned,
// and adds it to the queue for hub checking.
func (model *Model) AddImage(image Image) {
	_, hasImage := model.Images[image.Sha]
	if !hasImage {
		newInfo := NewImageInfo(image.Sha, image.Name)
		model.Images[image.Sha] = newInfo
		log.Debugf("added image %s to model", image.HumanReadableName())
		model.addImageToHubCheckQueue(image.Sha)
	} else {
		log.Debugf("not adding image %s to model, already have in cache", image.HumanReadableName())
	}
}

// image state transitions

func (model *Model) safeGet(sha DockerImageSha) *ImageInfo {
	results, ok := model.Images[sha]
	if !ok {
		message := fmt.Sprintf("expected to already have image %s, but did not", string(sha))
		log.Error(message)
		panic(message) // TODO get rid of panic
	}
	return results
}

func (model *Model) addImageToHubCheckQueue(sha DockerImageSha) {
	imageInfo := model.safeGet(sha)
	switch imageInfo.ScanStatus {
	case ScanStatusUnknown, ScanStatusError:
		break
	default:
		message := fmt.Sprintf("cannot add image %s to hub check queue, status is neither Unknown nor Error (%s)", sha, imageInfo.ScanStatus)
		log.Error(message)
		panic(message) // TODO get rid of panic
	}
	imageInfo.setScanStatus(ScanStatusInHubCheckQueue)
	model.ImageHubCheckQueue = append(model.ImageHubCheckQueue, imageInfo.image())
}

func (model *Model) addImageToScanQueue(sha DockerImageSha) {
	imageInfo := model.safeGet(sha)
	switch imageInfo.ScanStatus {
	case ScanStatusCheckingHub, ScanStatusError:
		break
	default:
		message := fmt.Sprintf("cannot add image %s to scan queue, status is neither CheckingHub nor Error (%s)", sha, imageInfo.ScanStatus)
		log.Error(message)
		panic(message) // TODO get rid of panic
	}
	imageInfo.setScanStatus(ScanStatusInQueue)
	model.ImageScanQueue = append(model.ImageScanQueue, imageInfo.image())
}

func (model *Model) getNextImageFromHubCheckQueue() *Image {
	if len(model.ImageHubCheckQueue) == 0 {
		log.Info("hub check queue empty")
		return nil
	}

	first := model.ImageHubCheckQueue[0]
	imageInfo := model.safeGet(first.Sha)
	if imageInfo.ScanStatus != ScanStatusInHubCheckQueue {
		message := fmt.Sprintf("can't start checking hub for image %s, status is not ScanStatusInHubCheckQueue (%s)", string(first.Sha), imageInfo.ScanStatus)
		log.Errorf(message)
		panic(message) // TODO get rid of this panic
	}

	imageInfo.setScanStatus(ScanStatusCheckingHub)
	model.ImageHubCheckQueue = model.ImageHubCheckQueue[1:]
	return &first
}

func (model *Model) getNextImageFromScanQueue() *Image {
	if model.inProgressScanCount() >= model.ConcurrentScanLimit {
		log.Infof("max concurrent scan count reached, can't start a new scan -- %v", model.inProgressScans())
		return nil
	}

	if len(model.ImageScanQueue) == 0 {
		log.Info("scan queue empty, can't start a new scan")
		return nil
	}

	first := model.ImageScanQueue[0]
	imageInfo := model.safeGet(first.Sha)
	if imageInfo.ScanStatus != ScanStatusInQueue {
		message := fmt.Sprintf("can't start scanning image %s, status is not InQueue (%s)", string(first.Sha), imageInfo.ScanStatus)
		log.Errorf(message)
		panic(message) // TODO get rid of this panic
	}

	imageInfo.setScanStatus(ScanStatusRunningScanClient)
	model.ImageScanQueue = model.ImageScanQueue[1:]
	return &first
}

func (model *Model) errorRunningScanClient(sha DockerImageSha) {
	results := model.safeGet(sha)
	if results.ScanStatus != ScanStatusRunningScanClient {
		message := fmt.Sprintf("cannot error out scan client for image %s, scan client not in progress (%s)", string(sha), results.ScanStatus)
		log.Errorf(message)
		panic(message)
	}
	results.setScanStatus(ScanStatusError)
	// TODO get rid of these
	// for now, just readd the image to the queue upon error
	model.addImageToScanQueue(sha)
}

func (model *Model) finishRunningScanClient(sha DockerImageSha) {
	results := model.safeGet(sha)
	if results.ScanStatus != ScanStatusRunningScanClient {
		message := fmt.Sprintf("cannot finish running scan client for image %s, scan client not in progress (%s)", string(sha), results.ScanStatus)
		log.Errorf(message)
		panic(message) // TODO get rid of panic
	}
	results.setScanStatus(ScanStatusRunningHubScan)
}

// additional methods

func (model *Model) inProgressScans() []DockerImageSha {
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

func (model *Model) inProgressScanCount() int {
	return len(model.inProgressScans())
}

func (model *Model) inProgressScanClientScans() []*ImageInfo {
	inProgressScans := []*ImageInfo{}
	for _, imageInfo := range model.Images {
		switch imageInfo.ScanStatus {
		case ScanStatusRunningScanClient:
			inProgressScans = append(inProgressScans, imageInfo)
		}
	}
	return inProgressScans
}

func (model *Model) inProgressHubScans() []Image {
	inProgressHubScans := []Image{}
	for _, imageInfo := range model.Images {
		switch imageInfo.ScanStatus {
		case ScanStatusRunningHubScan:
			inProgressHubScans = append(inProgressHubScans, imageInfo.image())
		}
	}
	return inProgressHubScans
}

func (model *Model) metrics() *ModelMetrics {
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

	// TODO
	// number of images without a pod pointing to them
	return &ModelMetrics{
		ScanStatusCounts:    statusCounts,
		NumberOfImages:      len(model.Images),
		NumberOfPods:        len(model.Pods),
		ContainerCounts:     containerCounts,
		ImageCountHistogram: imageCountHistogram,
	}
}

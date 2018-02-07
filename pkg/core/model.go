/*
Copyright (C) 2018 Black Duck Software, Inc.

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
	"encoding/json"
	"fmt"
	"strings"

	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
	log "github.com/sirupsen/logrus"
)

// Model is the root of the core model
type Model struct {
	// Pods is a map of "<namespace>/<name>" to pod
	Pods                map[string]common.Pod
	Images              map[common.Image]*ImageScanResults
	ImageScanQueue      []common.Image
	ImageHubCheckQueue  []common.Image
	ConcurrentScanLimit int
}

func (model Model) MarshalJSON() ([]byte, error) {
	strs := []string{"{"}
	// pods
	strs = append(strs, "\"Pods\":")
	podBytes, err := json.Marshal(model.Pods)
	if err != nil {
		return []byte{}, err
	}
	strs = append(strs, string(podBytes))
	strs = append(strs, ",")
	// images
	strs = append(strs, "\"Images\":{")
	images := []string{}
	for key, val := range model.Images {
		resultsBytes, err := json.Marshal(val.ScanResults)
		if err != nil {
			return []byte{}, err
		}
		myMap := map[string]string{
			"ScanStatus":  val.ScanStatus.String(),
			"ScanResults": string(resultsBytes),
			"Name":        key.Name,
			"DockerImage": key.DockerImage,
		}
		mapBytes, err := json.Marshal(myMap)
		if err != nil {
			return []byte{}, err
		}
		images = append(images, fmt.Sprintf("\"%s\":%s", key.Sha, string(mapBytes)))
	}
	strs = append(strs, strings.Join(images, ","))
	strs = append(strs, "},")
	// ImageScanQueue
	strs = append(strs, "\"ImageScanQueue\":")
	scanQueueBytes, err := json.Marshal(model.ImageScanQueue)
	if err != nil {
		return []byte{}, err
	}
	strs = append(strs, string(scanQueueBytes))
	strs = append(strs, ",")
	// ImageHubCheckQueue
	strs = append(strs, "\"ImageHubCheckQueue\":")
	checkQueueBytes, err := json.Marshal(model.ImageHubCheckQueue)
	if err != nil {
		return []byte{}, err
	}
	strs = append(strs, string(checkQueueBytes))
	strs = append(strs, ",")
	// ConcurrentScanLimit
	strs = append(strs, "\"ConcurrentScanLimit\":")
	concurrentLimitBytes, err := json.Marshal(model.ConcurrentScanLimit)
	if err != nil {
		return []byte{}, err
	}
	strs = append(strs, string(concurrentLimitBytes))
	// closing bracket
	strs = append(strs, "}")
	// done
	return []byte(strings.Join(strs, "")), nil
}

func NewModel(concurrentScanLimit int) *Model {
	return &Model{
		Pods:                make(map[string]common.Pod),
		Images:              make(map[common.Image]*ImageScanResults),
		ImageScanQueue:      []common.Image{},
		ImageHubCheckQueue:  []common.Image{},
		ConcurrentScanLimit: concurrentScanLimit}
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
func (model *Model) AddPod(newPod common.Pod) {
	log.Debugf("about to add pod: UID %s, qualified name %s", newPod.UID, newPod.QualifiedName())
	for _, newCont := range newPod.Containers {
		model.AddImage(newCont.Image)
	}
	log.Debugf("done adding containers+images from pod %s -- %s", newPod.UID, newPod.QualifiedName())
	model.Pods[newPod.QualifiedName()] = newPod
}

// AddImage adds an image to the model, sets its status to NotScanned,
// and adds it to the queue for hub checking.
func (model *Model) AddImage(image common.Image) {
	_, hasImage := model.Images[image]
	if !hasImage {
		addedImage := NewImageScanResults()
		model.Images[image] = addedImage
		log.Debugf("added image %s to model", image.HumanReadableName())
		model.addImageToHubCheckQueue(image)
	} else {
		log.Debugf("not adding image %s to model, already have in cache", image.HumanReadableName())
	}
}

// image state transitions

func (model *Model) safeGet(image common.Image) *ImageScanResults {
	results, ok := model.Images[image]
	if !ok {
		message := fmt.Sprintf("expected to already have image %s, but did not", image.HumanReadableName())
		log.Error(message)
		panic(message) // TODO get rid of panic
	}
	return results
}

func (model *Model) addImageToHubCheckQueue(image common.Image) {
	results := model.safeGet(image)
	switch results.ScanStatus {
	case ScanStatusUnknown, ScanStatusError:
		break
	default:
		message := fmt.Sprintf("cannot add image %s to hub check queue, status is neither Unknown nor Error (%s)", image.HumanReadableName(), results.ScanStatus)
		log.Error(message)
		panic(message) // TODO get rid of panic
	}
	results.ScanStatus = ScanStatusInHubCheckQueue
	model.ImageHubCheckQueue = append(model.ImageHubCheckQueue, image)
}

func (model *Model) addImageToScanQueue(image common.Image) {
	results := model.safeGet(image)
	switch results.ScanStatus {
	case ScanStatusCheckingHub, ScanStatusError:
		break
	default:
		message := fmt.Sprintf("cannot add image %s to scan queue, status is neither CheckingHub nor Error (%s)", image.HumanReadableName(), results.ScanStatus)
		log.Error(message)
		panic(message) // TODO get rid of panic
	}
	results.ScanStatus = ScanStatusInQueue
	model.ImageScanQueue = append(model.ImageScanQueue, image)
}

func (model *Model) getNextImageFromHubCheckQueue() *common.Image {
	if len(model.ImageHubCheckQueue) == 0 {
		log.Info("hub check queue empty")
		return nil
	}

	first := model.ImageHubCheckQueue[0]
	results := model.safeGet(first)
	if results.ScanStatus != ScanStatusInHubCheckQueue {
		message := fmt.Sprintf("can't start checking hub for image %s, status is not ScanStatusInHubCheckQueue (%s)", first.HumanReadableName(), results.ScanStatus)
		log.Errorf(message)
		panic(message) // TODO get rid of this panic
	}

	results.ScanStatus = ScanStatusCheckingHub
	model.ImageHubCheckQueue = model.ImageHubCheckQueue[1:]
	return &first
}

func (model *Model) getNextImageFromScanQueue() *common.Image {
	if model.inProgressScanCount() >= model.ConcurrentScanLimit {
		log.Infof("max concurrent scan count reached, can't start a new scan -- %v", model.inProgressScanJobs())
		return nil
	}

	if len(model.ImageScanQueue) == 0 {
		log.Info("scan queue empty, can't start a new scan")
		return nil
	}

	first := model.ImageScanQueue[0]
	results := model.safeGet(first)
	if results.ScanStatus != ScanStatusInQueue {
		message := fmt.Sprintf("can't start scanning image %s, status is not InQueue (%s)", first.HumanReadableName(), results.ScanStatus)
		log.Errorf(message)
		panic(message) // TODO get rid of this panic
	}

	results.ScanStatus = ScanStatusRunningScanClient
	model.ImageScanQueue = model.ImageScanQueue[1:]
	return &first
}

func (model *Model) errorRunningScanClient(image common.Image) {
	results := model.safeGet(image)
	if results.ScanStatus != ScanStatusRunningScanClient {
		message := fmt.Sprintf("cannot error out scan client for image %s, scan client not in progress (%s)", image.HumanReadableName(), results.ScanStatus)
		log.Errorf(message)
		panic(message)
	}
	results.ScanStatus = ScanStatusError
	// TODO get rid of these
	// for now, just readd the image to the queue upon error
	model.addImageToScanQueue(image)
}

func (model *Model) finishRunningScanClient(image common.Image) {
	results := model.safeGet(image)
	if results.ScanStatus != ScanStatusRunningScanClient {
		message := fmt.Sprintf("cannot finish running scan client for image %s, scan client not in progress (%s)", image.HumanReadableName(), results.ScanStatus)
		log.Errorf(message)
		panic(message) // TODO get rid of panic
	}
	results.ScanStatus = ScanStatusRunningHubScan
}

// func (model *Model) finishRunningHubScan(image common.Image) {
// 	results := model.safeGet(image)
// 	if results.ScanStatus != ScanStatusRunningHubScan {
// 		message := fmt.Sprintf("cannot finish running hub scan for image %s, scan not in progress (%s)", image.HumanReadableName(), results.ScanStatus)
// 		log.Errorf(message)
// 		panic(message)
// 	}
// 	results.ScanStatus = ScanStatusComplete
// }

// additional methods

func (model *Model) inProgressScanJobs() []common.Image {
	inProgressImages := []common.Image{}
	for image, results := range model.Images {
		switch results.ScanStatus {
		case ScanStatusRunningScanClient, ScanStatusRunningHubScan:
			inProgressImages = append(inProgressImages, image)
		default:
			break
		}
	}
	return inProgressImages
}

func (model *Model) inProgressScanCount() int {
	return len(model.inProgressScanJobs())
}

func (model *Model) inProgressHubScans() []common.Image {
	inProgressHubScans := []common.Image{}
	for image, results := range model.Images {
		switch results.ScanStatus {
		case ScanStatusRunningHubScan:
			inProgressHubScans = append(inProgressHubScans, image)
		}
	}
	return inProgressHubScans
}

func (model *Model) scanResults(podName string) (int, int, string, error) {
	pod, ok := model.Pods[podName]
	if !ok {
		return 0, 0, "", fmt.Errorf("could not find pod of name %s in cache", podName)
	}

	overallStatus := ""
	policyViolationCount := 0
	vulnerabilityCount := 0
	for _, container := range pod.Containers {
		imageScanResults, ok := model.Images[container.Image]
		if !ok {
			continue
		}
		if imageScanResults.ScanStatus != ScanStatusComplete {
			continue
		}
		if imageScanResults.ScanResults == nil {
			continue
		}
		policyViolationCount += imageScanResults.ScanResults.PolicyViolationCount()
		vulnerabilityCount += imageScanResults.ScanResults.VulnerabilityCount()
		// TODO what's the right way to combine all the 'OverallStatus' values
		//   from the individual image scans?
		if imageScanResults.ScanResults.OverallStatus() != "NOT_IN_VIOLATION" {
			overallStatus = imageScanResults.ScanResults.OverallStatus()
		}
	}
	return policyViolationCount, vulnerabilityCount, overallStatus, nil
}

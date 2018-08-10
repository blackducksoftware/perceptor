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
	"reflect"

	"github.com/blackducksoftware/perceptor/pkg/util"
	log "github.com/sirupsen/logrus"
)

// Model is the root of the core model
type Model struct {
	// Pods is a map of qualified name ("<namespace>/<name>") to pod
	Pods           map[string]Pod
	Images         map[DockerImageSha]*ImageInfo
	ImageScanQueue *util.PriorityQueue
	ImagePriority  map[DockerImageSha]int
}

// NewModel .....
func NewModel() *Model {
	return &Model{
		Pods:           make(map[string]Pod),
		Images:         make(map[DockerImageSha]*ImageInfo),
		ImageScanQueue: util.NewPriorityQueue(),
		ImagePriority:  map[DockerImageSha]int{},
	}
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
		model.AddImage(newCont.Image, 1)
	}
	log.Debugf("done adding containers+images from pod %s -- %s", newPod.UID, newPod.QualifiedName())
	model.Pods[newPod.QualifiedName()] = newPod
}

// AddImage adds an image to the model, adding it to the queue for hub checking.
func (model *Model) AddImage(image Image, priority int) {
	added := model.createImage(image)
	if added {
		model.ImagePriority[image.Sha] = priority
	}
}

// DeleteImage removes an image from the model.
// WARNING: It should ABSOLUTELY NOT be called for images that are still referenced by one or more pods.
// WARNING: It should *probably* not be called for images in the ScanStatusRunningScanClient
//   or ScanStatusRunningHubScan states.
func (model *Model) DeleteImage(sha DockerImageSha) error {
	if _, ok := model.Images[sha]; !ok {
		return fmt.Errorf("unable to delete image %s, not found", sha)
	}
	delete(model.Images, sha)
	return nil
}

// image state transitions

func (model *Model) leaveState(sha DockerImageSha, state ScanStatus) error {
	switch state {
	case ScanStatusInQueue:
		return model.removeImageFromScanQueue(sha)
	case ScanStatusUnknown, ScanStatusRunningScanClient, ScanStatusRunningHubScan, ScanStatusComplete:
		return nil
	default:
		return fmt.Errorf("leaveState: invalid ScanStatus %d", state)
	}
}

func (model *Model) enterState(sha DockerImageSha, state ScanStatus) error {
	switch state {
	case ScanStatusInQueue:
		return model.addImageToScanQueue(sha)
	case ScanStatusUnknown, ScanStatusRunningScanClient, ScanStatusRunningHubScan, ScanStatusComplete:
		return nil
	default:
		return fmt.Errorf("enterState: invalid ScanStatus %d", state)
	}
}

func (model *Model) setImageScanStatusForSha(sha DockerImageSha, newScanStatus ScanStatus) error {
	imageInfo, ok := model.Images[sha]
	if !ok {
		return fmt.Errorf("can not set scan status for sha %s, sha not found", string(sha))
	}

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
		newInfo := NewImageInfo(image.Sha, &RepoTag{Repository: image.Repository, Tag: image.Tag})
		model.Images[image.Sha] = newInfo
		log.Debugf("added image %s to model", image.PullSpec())
	} else {
		log.Debugf("not adding image %s to model, already have in cache", image.PullSpec())
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

func (model *Model) addImageToScanQueue(sha DockerImageSha) error {
	priority := model.ImagePriority[sha]
	return model.ImageScanQueue.Add(string(sha), priority, sha)
}

func (model *Model) removeImageFromScanQueue(sha DockerImageSha) error {
	_, err := model.ImageScanQueue.Remove(string(sha))
	return err
}

// "Public" methods

// SetImageScanStatus .....
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

// GetNextImageFromScanQueue .....
func (model *Model) GetNextImageFromScanQueue() *Image {
	if model.ImageScanQueue.IsEmpty() {
		log.Debug("scan queue empty, can't start a new scan")
		return nil
	}

	first, err := model.ImageScanQueue.Pop()
	if err != nil {
		log.Errorf("unable to get next image from scan queue: %s", err.Error())
		return nil
	}

	switch sha := first.(type) {
	case DockerImageSha:
		image := model.unsafeGet(sha).Image()
		model.SetImageScanStatus(sha, ScanStatusRunningScanClient)
		return &image
	default:
		log.Errorf("expected type DockerImageSha from priority queue, got %s", reflect.TypeOf(first))
		return nil
	}
}

// FinishRunningScanClient .....
func (model *Model) FinishRunningScanClient(image *Image, scanClientError error) {
	_, ok := model.Images[image.Sha]

	// if we don't have this sha already, let's add it to the model,
	// but *NOT* to the scan queue
	if !ok {
		log.Warnf("finish running scan client -- expected to already have image %s, but did not", string(image.Sha))
		_ = model.createImage(*image)
	}

	scanStatus := ScanStatusRunningHubScan
	if scanClientError != nil {
		scanStatus = ScanStatusInQueue
		log.Errorf("error running scan client -- %s", scanClientError.Error())
	}

	model.SetImageScanStatus(image.Sha, scanStatus)
}

// additional methods

// InProgressScans .....
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

// InProgressScanCount .....
func (model *Model) InProgressScanCount() int {
	return len(model.InProgressScans())
}

// InProgressHubScans .....
func (model *Model) InProgressHubScans() *([]Image) {
	inProgressHubScans := []Image{}
	for _, imageInfo := range model.Images {
		switch imageInfo.ScanStatus {
		case ScanStatusRunningHubScan:
			inProgressHubScans = append(inProgressHubScans, imageInfo.Image())
		}
	}
	return &inProgressHubScans
}

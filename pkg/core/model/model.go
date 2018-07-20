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

	util "github.com/blackducksoftware/perceptor/pkg/util"
	log "github.com/sirupsen/logrus"
)

// Model is the root of the core model
type Model struct {
	// Pods is a map of qualified name ("<namespace>/<name>") to pod
	Pods           map[string]Pod
	Images         map[DockerImageSha]*ImageInfo
	ImageScanQueue *util.PriorityQueue
	ImagePriority  map[DockerImageSha]int
	// map of HubURL to a set of image SHAs
	HubImageAssignments map[string]*Hub
	// ImageRefreshQueue    []DockerImageSha
	// ImageRefreshQueueSet map[DockerImageSha]bool
	Config  *Config
	Timings *Timings
	updates chan Update
}

// NewModel .....
func NewModel(config *Config, timings *Timings) *Model {
	return &Model{
		Pods:           make(map[string]Pod),
		Images:         make(map[DockerImageSha]*ImageInfo),
		ImageScanQueue: util.NewPriorityQueue(),
		ImagePriority:  map[DockerImageSha]int{},
		// ImageRefreshQueue:    []DockerImageSha{},
		// ImageRefreshQueueSet: make(map[DockerImageSha]bool),
		HubImageAssignments: map[string]*Hub{},
		Config:              config,
		Timings:             timings,
		updates:             make(chan Update),
	}
}

// Updates ...
func (model *Model) Updates() <-chan Update {
	return model.updates
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
func (model *Model) GetNextImageFromScanQueue() (*HubImageAssignment, error) {
	// 1. preliminaries
	if model.ImageScanQueue.IsEmpty() {
		log.Debug("scan queue empty, can't start a new scan")
		return nil, nil
	}

	// 2. find a hub

	var assignedHub *Hub
	for _, hub := range model.HubImageAssignments {
		if hub.InProgressScanCount() < model.Config.ConcurrentScanLimit {
			assignedHub = hub
			break
		}
	}

	if assignedHub == nil {
		log.Debugf("no available hub found -- %+v", model.HubImageAssignments)
		return nil, nil
	}

	// 3. find an image

	first, err := model.ImageScanQueue.Pop()
	if err != nil {
		log.Errorf("unable to get next image from scan queue: %s", err.Error())
		return nil, err
	}

	var imageInfo *ImageInfo
	switch sha := first.(type) {
	case DockerImageSha:
		imageInfo = model.unsafeGet(sha)
		model.SetImageScanStatus(sha, ScanStatusRunningScanClient)
	default:
		err := fmt.Errorf("expected type DockerImageSha from priority queue, got %s", reflect.TypeOf(first))
		log.Error(err.Error())
		return nil, err
	}
	sha := imageInfo.ImageSha

	err = model.assignImageToHub(sha, assignedHub.URL)
	if err != nil {
		return nil, err
	}

	err = assignedHub.StartScanningImage(sha)
	if err != nil {
		return nil, err
	}

	image := imageInfo.Image()
	assignment := &HubImageAssignment{HubURL: assignedHub.URL, Image: &image}
	go func() {
		model.updates <- &StartScan{Assignment: assignment}
	}()
	return assignment, nil
}

// // AddImageToRefreshQueue .....
// func (model *Model) AddImageToRefreshQueue(sha DockerImageSha) error {
// 	imageInfo, ok := model.Images[sha]
// 	if !ok {
// 		return fmt.Errorf("expected to already have image %s, but did not", string(sha))
// 	}
//
// 	if imageInfo.ScanStatus != ScanStatusComplete {
// 		return fmt.Errorf("unable to refresh image %s, scan status is %s", string(sha), imageInfo.ScanStatus.String())
// 	}
//
// 	// if it's already in the refresh queue, don't add it again
// 	_, ok = model.ImageRefreshQueueSet[sha]
// 	if ok {
// 		return fmt.Errorf("unable to add image %s to refresh queue, already in queue", string(sha))
// 	}
//
// 	model.ImageRefreshQueue = append(model.ImageRefreshQueue, sha)
// 	model.ImageRefreshQueueSet[sha] = false
// 	return nil
// }
//
// // GetNextImageFromRefreshQueue .....
// func (model *Model) GetNextImageFromRefreshQueue() *Image {
// 	if len(model.ImageRefreshQueue) == 0 {
// 		log.Debug("refresh queue empty")
// 		return nil
// 	}
//
// 	first := model.ImageRefreshQueue[0]
// 	image := model.unsafeGet(first).Image()
//
// 	return &image
// }
//
// // RemoveImageFromRefreshQueue .....
// func (model *Model) RemoveImageFromRefreshQueue(sha DockerImageSha) error {
// 	index := -1
// 	for i := 0; i < len(model.ImageRefreshQueue); i++ {
// 		if model.ImageRefreshQueue[i] == sha {
// 			index = i
// 			break
// 		}
// 	}
// 	if index < 0 {
// 		return fmt.Errorf("unable to remove sha %s from refresh queue, not found", string(sha))
// 	}
//
// 	model.ImageRefreshQueue = append(model.ImageRefreshQueue[:index], model.ImageRefreshQueue[index+1:]...)
// 	delete(model.ImageRefreshQueueSet, sha)
// 	return nil
// }

// FinishRunningScanClient .....
func (model *Model) FinishRunningScanClient(image *Image, hubURL string, scanClientError error) {
	_, ok := model.Images[image.Sha]

	// if we don't have this sha already, let's drop it.  Hub recovery will handle finding it again.
	if !ok {
		log.Warnf("finish running scan client -- expected to already have image %s, but did not", string(image.Sha))
		return
	}

	hub, ok := model.HubImageAssignments[hubURL]
	if !ok {
		log.Warnf("finish running scan client -- expected to already have hub %s, but did not", hubURL)
		return
	}

	scanStatus := ScanStatusRunningHubScan
	if scanClientError != nil {
		err := model.unassignImageFromHub(image.Sha, hubURL)
		if err != nil {
			log.Error(err.Error())
		}
		scanStatus = ScanStatusInQueue
		log.Errorf("error running scan client -- %s", scanClientError.Error())
	} else {
		err := hub.ScanDidFinish(image.Sha)
		if err != nil {
			log.Error(err.Error())
		}
	}
	model.SetImageScanStatus(image.Sha, scanStatus)
}

// additional methods

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

// hubs

func (model *Model) assignImageToHub(sha DockerImageSha, hubURL string) error {
	imageInfo, ok := model.Images[sha]
	if !ok {
		return fmt.Errorf("image %s not present", sha)
	}
	hub, ok := model.HubImageAssignments[hubURL]
	if !ok {
		return fmt.Errorf("hub URL %s not present", hubURL)
	}
	err := hub.AddImage(sha)
	if err != nil {
		return err
	}
	err = imageInfo.setHubURL(hubURL)
	if err != nil {
		return err
	}
	return nil
}

func (model *Model) unassignImageFromHub(sha DockerImageSha, hubURL string) error {
	imageInfo, ok := model.Images[sha]
	if !ok {
		return fmt.Errorf("image %s not present", sha)
	}
	hub, ok := model.HubImageAssignments[hubURL]
	if !ok {
		return fmt.Errorf("hub URL %s not present", hubURL)
	}
	err := hub.RemoveImage(sha)
	if err != nil {
		return err
	}
	return imageInfo.removeHubURL()
}

func (model *Model) addHub(hubURL string) error {
	_, ok := model.HubImageAssignments[hubURL]
	if ok {
		return fmt.Errorf("cannot add hub %s: already present", hubURL)
	}
	model.HubImageAssignments[hubURL] = NewHub(hubURL)
	return nil
}

func (model *Model) deleteHub(hubURL string) error {
	hub, ok := model.HubImageAssignments[hubURL]
	if !ok {
		return fmt.Errorf("cannot delete hub %s: not found", hubURL)
	}
	// 1. remove all image assignments to this hub
	for sha := range hub.Images {
		err := model.unassignImageFromHub(sha, hubURL)
		if err != nil {
			return err
		}
	}
	// 2. remove this hub
	delete(model.HubImageAssignments, hubURL)
	// 3. done
	return nil
}

// SetHubs ...
func (model *Model) SetHubs(hubURLs []string) error {
	newHubURLs := map[string]bool{}
	for _, hubURL := range hubURLs {
		newHubURLs[hubURL] = true
	}
	// delete hubs
	hubsToDelete := []string{}
	for hubURL := range model.HubImageAssignments {
		if _, ok := newHubURLs[hubURL]; !ok {
			hubsToDelete = append(hubsToDelete, hubURL)
		}
	}
	for _, hubURL := range hubsToDelete {
		err := model.deleteHub(hubURL)
		if err != nil {
			return err
		}
	}
	// create hubs
	for hubURL := range newHubURLs {
		if _, ok := model.HubImageAssignments[hubURL]; !ok {
			err := model.addHub(hubURL)
			if err != nil {
				return err
			}
		}
	}
	// updates
	go func() {
		for _, deleteHubURL := range hubsToDelete {
			model.updates <- &DeleteHub{HubURL: deleteHubURL}
		}
		for createHubURL := range newHubURLs {
			model.updates <- &CreateHub{HubURL: createHubURL}
		}
	}()
	return nil
}

func (model *Model) SetIsHubEnabled(hubURL string, isEnabled bool) error {
	hub, ok := model.HubImageAssignments[hubURL]
	if !ok {
		return fmt.Errorf("hub %s not found", hubURL)
	}
	hub.SetEnabled(isEnabled)
	return nil
}

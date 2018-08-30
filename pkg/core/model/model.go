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
	"time"

	"github.com/blackducksoftware/perceptor/pkg/api"
	"github.com/blackducksoftware/perceptor/pkg/hub"
	"github.com/blackducksoftware/perceptor/pkg/util"
	log "github.com/sirupsen/logrus"
)

const (
	actionChannelSize = 100
)

// Model is the root of the core model
type Model struct {
	// Pods is a map of qualified name ("<namespace>/<name>") to pod
	Pods           map[string]Pod
	Images         map[DockerImageSha]*ImageInfo
	ImageScanQueue *util.PriorityQueue
	ImagePriority  map[DockerImageSha]int
	//
	actions chan Action
}

// NewModel .....
func NewModel() *Model {
	model := &Model{
		Pods:           make(map[string]Pod),
		Images:         make(map[DockerImageSha]*ImageInfo),
		ImageScanQueue: util.NewPriorityQueue(),
		ImagePriority:  map[DockerImageSha]int{},
		actions:        make(chan Action, actionChannelSize),
	}
	go func() {
		stop := time.Now()
		for {
			select {
			case nextAction := <-model.actions:
				log.Debugf("processing model action of type %s", reflect.TypeOf(nextAction))

				// metrics: how many messages are waiting?
				recordNumberOfMessagesInQueue(len(model.actions))

				// metrics: log message type
				recordMessageType(fmt.Sprintf("%s", reflect.TypeOf(nextAction)))

				// metrics: how long idling since the last action finished processing?
				start := time.Now()
				recordReducerActivity(false, start.Sub(stop))

				// actually do the work
				nextAction.Apply(model)

				// metrics: how long did the work take?
				stop = time.Now()
				recordReducerActivity(true, stop.Sub(start))
			}
		}
	}()
	return model
}

// Public API

// AddPod ...
func (model *Model) AddPod(pod Pod) {
	model.actions <- &AddPod{Pod: pod}
}

// UpdatePod ...
func (model *Model) UpdatePod(pod Pod) {
	model.actions <- &UpdatePod{Pod: pod}
}

// DeletePod remove the record of a pod, but does not touch its images
func (model *Model) DeletePod(podName string) {
	model.actions <- &DeletePod{PodName: podName}
}

// SetPods ...
func (model *Model) SetPods(pods []Pod) {
	model.actions <- &AllPods{Pods: pods}
}

// AddImage ...
func (model *Model) AddImage(image Image) {
	model.actions <- &AddImage{Image: image}
}

// SetImages ...
func (model *Model) SetImages(images []Image) {
	model.actions <- &AllImages{Images: images}
}

// FinishScanJob should be called when the scan client has finished.
func (model *Model) FinishScanJob(image *Image, err error) {
	model.actions <- &FinishScanClient{Image: image, Err: err}
}

// ScanDidFinish should be called when:
// - the Hub scan finishes
// - upon startup, when scan results are first fetched
func (model *Model) ScanDidFinish(sha DockerImageSha, scanResults *hub.ScanResults) {
	model.actions <- &DidFetchScanResults{Sha: sha, ScanResults: scanResults}
}

// GetScanResults ...
func (model *Model) GetScanResults() api.ScanResults {
	get := NewGetScanResults()
	model.actions <- get
	return <-get.Done
}

// GetModel ...
func (model *Model) GetModel() api.CoreModel {
	get := NewGetModel()
	model.actions <- get
	return <-get.Done
}

// GetImages returns images in that status
func (model *Model) GetImages(status ScanStatus) []DockerImageSha {
	get := NewGetImages(status)
	model.actions <- get
	return <-get.Done
}

// GetMetrics calculates useful metrics for observing the progress of the model
// over time.
func (model *Model) GetMetrics() *Metrics {
	get := NewGetMetrics()
	model.actions <- get
	return <-get.Done
}

// GetNextImage ...
func (model *Model) GetNextImage() *Image {
	get := NewGetNextImage()
	model.actions <- get
	return <-get.Done
}

// StartScanClient ...
func (model *Model) StartScanClient(sha DockerImageSha) error {
	start := NewStartScanClient(sha)
	model.actions <- start
	return <-start.Error
}

// Package API

// AddPod adds a pod and all the images in a pod to the model.
// If the pod is already present in the model, it will be removed
// and a new one created in its place.
// The key is the combination of the pod's namespace and name.
// It extracts the containers and images from the pod,
// adding them into the cache.
func (model *Model) addPod(newPod Pod) {
	log.Debugf("about to add pod: UID %s, qualified name %s", newPod.UID, newPod.QualifiedName())
	if len(newPod.Containers) == 0 {
		recordEvent("adding pod with 0 containers")
		log.Warnf("adding pod %s with 0 containers: %+v", newPod.QualifiedName(), newPod)
	}
	for _, newCont := range newPod.Containers {
		model.addImage(newCont.Image, 1)
	}
	log.Debugf("done adding containers+images from pod %s -- %s", newPod.UID, newPod.QualifiedName())
	model.Pods[newPod.QualifiedName()] = newPod
}

// AddImage adds an image to the model, adding it to the queue for hub checking.
func (model *Model) addImage(image Image, priority int) {
	log.Debugf("about to add image %s, priority %d", image.Sha, priority)
	added := model.createImage(image)
	if added {
		model.ImagePriority[image.Sha] = priority
		return
	}
	if priority <= model.ImagePriority[image.Sha] {
		return
	}
	log.Debugf("upgrading priority for image %s to %d", image.Sha, priority)
	model.ImagePriority[image.Sha] = priority
	err := model.removeImageFromScanQueue(image.Sha)
	if err != nil {
		log.Errorf("unable to remove image %s from scan queue", image.Sha)
		return
	}
	err = model.addImageToScanQueue(image.Sha)
	if err != nil {
		log.Errorf("unable to re-add image %s to scan queue", image.Sha)
	}
}

func (model *Model) scanDidFinish(sha DockerImageSha, scanResults *hub.ScanResults) error {
	imageInfo, ok := model.Images[sha]
	if !ok {
		return fmt.Errorf("unable to handle scanDidFinish for %s: sha not found", sha)
	}
	if scanResults == nil {
		switch imageInfo.ScanStatus {
		case ScanStatusUnknown:
			model.setImageScanStatus(sha, ScanStatusInQueue)
		default:
			return fmt.Errorf("unexpectedly found nil ScanResults for image %s in state %s", sha, imageInfo.ScanStatus)
		}
	} else if scanResults.ScanSummaryStatus() == hub.ScanSummaryStatusSuccess {
		imageInfo.ScanResults = scanResults
		switch imageInfo.ScanStatus {
		case ScanStatusUnknown, ScanStatusInQueue, ScanStatusRunningScanClient, ScanStatusRunningHubScan:
			model.setImageScanStatus(sha, ScanStatusComplete)
		case ScanStatusComplete:
			// nothing to do
		}
	} else if scanResults.ScanSummaryStatus() == hub.ScanSummaryStatusInProgress {
		switch imageInfo.ScanStatus {
		case ScanStatusUnknown, ScanStatusInQueue:
			model.setImageScanStatus(sha, ScanStatusRunningHubScan)
		case ScanStatusRunningScanClient, ScanStatusRunningHubScan, ScanStatusComplete:
			// nothing to do
		}
	} else { // hub.ScanSummaryStatusFailure
		switch imageInfo.ScanStatus {
		case ScanStatusUnknown, ScanStatusRunningHubScan:
			model.setImageScanStatus(sha, ScanStatusInQueue)
		case ScanStatusInQueue, ScanStatusRunningScanClient, ScanStatusComplete:
			return fmt.Errorf("cannot handle scanDidFinish %s for image %s: cannot transition from state %s", imageInfo.ScanStatus, sha, imageInfo.ScanStatus.String())
		}
	}
	return nil
}

// DeleteImage removes an image from the model.
// WARNING: It should ABSOLUTELY NOT be called for images that are still referenced by one or more pods.
// WARNING: It should *probably* not be called for images in the ScanStatusRunningScanClient
//   or ScanStatusRunningHubScan states.
func (model *Model) deleteImage(sha DockerImageSha) error {
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
func (model *Model) setImageScanStatus(sha DockerImageSha, newScanStatus ScanStatus) {
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

// GetNextImageFromScanQueue simply returns the item at the front of the scan queue,
// non-destructively.
func (model *Model) getNextImageFromScanQueue() *Image {
	first := model.ImageScanQueue.Peek()
	switch sha := first.(type) {
	case DockerImageSha:
		image := model.unsafeGet(sha).Image()
		return &image
	case nil:
		return nil
	default:
		log.Errorf("expected type DockerImageSha from priority queue, got %s", reflect.TypeOf(first))
		log.Debugf("additional info of scan queue, values: %+v, %+v", model.ImageScanQueue, model.ImageScanQueue.Values())
		return nil
	}
}

// startScanClient attempts to move `sha` from state InQueue to state RunningScanClient,
// returning an error if the sha doesn't exist, or is not in state InQueue.
func (model *Model) startScanClient(sha DockerImageSha) error {
	imageInfo, ok := model.Images[sha]
	if !ok {
		return fmt.Errorf("unable to start scan client for image %s, not found", sha)
	}
	if imageInfo.ScanStatus != ScanStatusInQueue {
		return fmt.Errorf("unable to start scan client for image %s, not in state InQueue", sha)
	}
	model.setImageScanStatus(sha, ScanStatusRunningScanClient)
	return nil
}

// FinishRunningScanClient .....
func (model *Model) finishRunningScanClient(image *Image, scanClientError error) {
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

	model.setImageScanStatus(image.Sha, scanStatus)
}

// additional methods

// InProgressScans .....
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

// InProgressScanCount .....
func (model *Model) inProgressScanCount() int {
	return len(model.inProgressScans())
}

// InProgressHubScans .....
func (model *Model) inProgressHubScans() *([]Image) {
	inProgressHubScans := []Image{}
	for _, imageInfo := range model.Images {
		switch imageInfo.ScanStatus {
		case ScanStatusRunningHubScan:
			inProgressHubScans = append(inProgressHubScans, imageInfo.Image())
		}
	}
	return &inProgressHubScans
}

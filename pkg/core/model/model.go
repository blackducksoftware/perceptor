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
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
)

const (
	actionChannelSize = 100
)

// Model is the root of the core model
type Model struct {
	// Pods is a map of qualified name ("<namespace>/<name>") to pod
	Pods             map[string]Pod
	Images           map[DockerImageSha]*ImageInfo
	ImageScanQueue   *util.PriorityQueue
	ImageTransitions []*ImageTransition
	//
	actions chan *action
}

// NewModel .....
func NewModel() *Model {
	model := &Model{
		Pods:             make(map[string]Pod),
		Images:           make(map[DockerImageSha]*ImageInfo),
		ImageScanQueue:   util.NewPriorityQueue(),
		ImageTransitions: []*ImageTransition{},
		actions:          make(chan *action, actionChannelSize),
	}
	go func() {
		stop := time.Now()
		for {
			select {
			case nextAction := <-model.actions:
				actionName := nextAction.name
				log.Debugf("processing model action of type %s", actionName)

				// metrics: how many messages are waiting?
				recordNumberOfMessagesInQueue(len(model.actions))

				// metrics: log message type
				recordMessageType(actionName)

				// metrics: how long idling since the last action finished processing?
				start := time.Now()
				recordReducerActivity(false, start.Sub(stop))

				// actually do the work
				err := nextAction.apply()
				if err != nil {
					log.Errorf("problem processing action %s: %v", actionName, err)
					recordActionError(actionName)
				}

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
	model.actions <- &action{"addPod", func() error {
		return model.addPod(pod)
	}}
}

// UpdatePod ...
func (model *Model) UpdatePod(pod Pod) {
	model.actions <- &action{"updatePod", func() error {
		return model.addPod(pod)
	}}
}

// DeletePod removes the record of a pod, but does not touch its images
func (model *Model) DeletePod(podName string) {
	model.actions <- &action{"deletePod", func() error {
		return model.deletePod(podName)
	}}
}

// SetPods ...
func (model *Model) SetPods(pods []Pod) {
	model.actions <- &action{"allPods", func() error {
		return model.allPods(pods)
	}}
}

// AddImage ...
func (model *Model) AddImage(image Image) {
	model.actions <- &action{"addImage", func() error {
		return model.addImage(image)
	}}
}

// SetImages ...
func (model *Model) SetImages(images []Image) {
	model.actions <- &action{"allImages", func() error {
		return model.allImages(images)
	}}
}

// FinishScanJob should be called when the scan client has finished.
func (model *Model) FinishScanJob(image *Image, err error) {
	log.Infof("finish scan job: %+v, %v", image, err)
	model.actions <- &action{"finishScanJob", func() error {
		return model.finishRunningScanClient(image, err)
	}}
}

// ScanDidFinish should be called when:
// - the Hub scan finishes
// - upon startup, when scan results are first fetched
func (model *Model) ScanDidFinish(sha DockerImageSha, scanResults *hub.ScanResults) {
	model.actions <- &action{"scanDidFinish", func() error {
		return model.scanDidFinish(sha, scanResults)
	}}
}

// GetScanResults ...
func (model *Model) GetScanResults() api.ScanResults {
	done := make(chan api.ScanResults)
	model.actions <- &action{"getScanResults", func() error {
		scanResults, err := scanResults(model)
		go func() {
			done <- scanResults
		}()
		return err
	}}
	return <-done
}

// GetModel ...
func (model *Model) GetModel() *api.CoreModel {
	done := make(chan *api.CoreModel)
	model.actions <- &action{"getModel", func() error {
		apiModel := coreModelToAPIModel(model)
		go func() {
			done <- apiModel
		}()
		return nil
	}}
	return <-done
}

// GetImages returns images in that status
func (model *Model) GetImages(status ScanStatus) []DockerImageSha {
	done := make(chan []DockerImageSha)
	model.actions <- &action{"getImages", func() error {
		shas := model.getShas(status)
		go func() {
			done <- shas
		}()
		return nil
	}}
	return <-done
}

// GetMetrics calculates useful metrics for observing the progress of the model
// over time.
func (model *Model) GetMetrics() *Metrics {
	done := make(chan *Metrics)
	model.actions <- &action{"getMetrics", func() error {
		modelMetrics := metrics(model)
		go func() {
			done <- modelMetrics
		}()
		return nil
	}}
	return <-done
}

// GetNextImage ...
func (model *Model) GetNextImage() *Image {
	done := make(chan *Image)
	model.actions <- &action{"getNextImage", func() error {
		log.Debugf("looking for next image to scan")
		image, err := model.getNextImageFromScanQueue()
		go func() {
			done <- image
		}()
		return err
	}}
	return <-done
}

// StartScanClient ...
func (model *Model) StartScanClient(sha DockerImageSha) error {
	errCh := make(chan error)
	model.actions <- &action{"startScanClient", func() error {
		err := model.startScanClient(sha)
		go func() {
			errCh <- err
		}()
		return err
	}}
	return <-errCh
}

// Package API

// AddPod adds a pod and all the images in a pod to the model.
// If the pod is already present in the model, it will be removed
// and a new one created in its place.
// The key is the combination of the pod's namespace and name.
// It extracts the containers and images from the pod,
// adding them into the cache.
func (model *Model) addPod(newPod Pod) error {
	log.Debugf("about to add pod: UID %s, qualified name %s", newPod.UID, newPod.QualifiedName())
	if len(newPod.Containers) == 0 {
		recordEvent("adding pod with 0 containers")
		log.Warnf("adding pod %s with 0 containers: %+v", newPod.QualifiedName(), newPod)
	}
	errors := []error{}
	for _, newCont := range newPod.Containers {
		err := model.addImage(newCont.Image)
		if err != nil {
			errors = append(errors, err)
		}
	}
	log.Debugf("done adding containers+images from pod %s -- %s", newPod.UID, newPod.QualifiedName())
	model.Pods[newPod.QualifiedName()] = newPod
	return combineErrors("adding pod images", errors)
}

// AddImage adds an image to the model, adding it to the queue for hub checking.
func (model *Model) addImage(image Image) error {
	log.Debugf("about to add image %s, priority %d", image.Sha, image.Priority)
	added, err := model.createImage(image)
	log.Debugf("added image %s? %t", image.Sha, added)
	return err
}

func (model *Model) scanDidFinish(sha DockerImageSha, scanResults *hub.ScanResults) error {
	imageInfo, ok := model.Images[sha]
	if !ok {
		return fmt.Errorf("unable to handle scanDidFinish for %s: sha not found", sha)
	}
	if scanResults == nil {
		switch imageInfo.ScanStatus {
		case ScanStatusUnknown:
			return model.setImageScanStatus(sha, ScanStatusInQueue)
		default:
			return fmt.Errorf("unexpectedly found nil ScanResults for image %s in state %s", sha, imageInfo.ScanStatus)
		}
	} else if scanResults.ScanSummaryStatus() == hub.ScanSummaryStatusSuccess {
		imageInfo.ScanResults = scanResults
		switch imageInfo.ScanStatus {
		case ScanStatusUnknown, ScanStatusInQueue, ScanStatusRunningScanClient, ScanStatusRunningHubScan:
			return model.setImageScanStatus(sha, ScanStatusComplete)
		default: // case ScanStatusComplete:
			return nil // nothing to do
		}
	} else if scanResults.ScanSummaryStatus() == hub.ScanSummaryStatusInProgress {
		switch imageInfo.ScanStatus {
		case ScanStatusUnknown, ScanStatusInQueue:
			return model.setImageScanStatus(sha, ScanStatusRunningHubScan)
		default: // case ScanStatusRunningScanClient, ScanStatusRunningHubScan, ScanStatusComplete:
			return nil // nothing to do
		}
	} else { // hub.ScanSummaryStatusFailure
		switch imageInfo.ScanStatus {
		case ScanStatusUnknown, ScanStatusRunningHubScan:
			return model.setImageScanStatus(sha, ScanStatusInQueue)
		default: // case ScanStatusInQueue, ScanStatusRunningScanClient, ScanStatusComplete:
			return fmt.Errorf("cannot handle scanDidFinish %s for image %s: cannot transition from state %s", imageInfo.ScanStatus, sha, imageInfo.ScanStatus.String())
		}
	}
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

	err := model.leaveState(sha, imageInfo.ScanStatus)
	if err != nil {
		return errors.Annotatef(err, "unable to leaveState %s for sha %s", imageInfo.ScanStatus.String(), sha)
	}
	err = model.enterState(sha, newScanStatus)
	if err != nil {
		return errors.Annotatef(err, "unable to enter state %s for sha %s", newScanStatus, sha)
	}
	imageInfo.setScanStatus(newScanStatus)

	return nil
}

// createImage adds the image to the model, but not to the scan queue
func (model *Model) createImage(image Image) (bool, error) {
	imageInfo, ok := model.Images[image.Sha]
	added := !ok
	if ok {
		newPriority, oldPriority := image.Priority, imageInfo.Priority
		log.Debugf("not adding image %s to model, already have in cache", image.PullSpec())
		if newPriority <= oldPriority {
			log.Debugf("not decreasing priority for image %s", image.PullSpec())
			return added, nil
		}
		if oldPriority < 0 {
			log.Debugf("not increasing priority for image %s, old priority was %d", image.PullSpec(), oldPriority)
			return added, nil
		}
		log.Debugf("upgrading priority for image %s to %d", image.PullSpec(), image.Priority)
		imageInfo.SetPriority(image.Priority)
		if imageInfo.ScanStatus != ScanStatusInQueue {
			return added, nil
		}
		err := model.setImagePriority(image.Sha, image.Priority)
		if err != nil {
			return added, errors.Annotatef(err, "unable to set image %s priority in scan queue to %d", image.Sha, image.Priority)
		}
		return added, nil
	}
	newInfo := NewImageInfo(image, &RepoTag{Repository: image.Repository, Tag: image.Tag})
	model.Images[image.Sha] = newInfo
	log.Debugf("added image %s to model", image.PullSpec())
	return added, nil
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
	imageInfo, ok := model.Images[sha]
	if !ok {
		return fmt.Errorf("unable to add image %s to scan queue: not found", sha)
	}
	return model.ImageScanQueue.Add(string(sha), imageInfo.Priority, sha)
}

func (model *Model) setImagePriority(sha DockerImageSha, newPriority int) error {
	return model.ImageScanQueue.Set(string(sha), newPriority)
}

func (model *Model) removeImageFromScanQueue(sha DockerImageSha) error {
	_, err := model.ImageScanQueue.Remove(string(sha))
	return err
}

// "Public" methods

func (model *Model) setImageScanStatus(sha DockerImageSha, newScanStatus ScanStatus) error {
	log.Debugf("setImageScanStatus for %s to %s", sha, newScanStatus)
	imageInfo, ok := model.Images[sha]
	statusString := "sha not found"
	if ok {
		statusString = imageInfo.ScanStatus.String()
	}
	err := model.setImageScanStatusForSha(sha, newScanStatus)
	model.ImageTransitions = append(model.ImageTransitions, NewImageTransition(sha, statusString, newScanStatus, err))
	if err != nil {
		return errors.Annotatef(err, "unable to transition image state for sha %s from <%s> to %s", sha, statusString, newScanStatus)
	}
	log.Debugf("successfully transitioned image %s from <%s> to %s", sha, statusString, newScanStatus)
	return nil
}

// getNextImageFromScanQueue simply returns the item at the front of the scan queue,
// non-destructively.
func (model *Model) getNextImageFromScanQueue() (*Image, error) {
	first := model.ImageScanQueue.Peek()
	switch sha := first.(type) {
	case DockerImageSha:
		image := model.unsafeGet(sha).Image()
		return &image, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("expected type DockerImageSha from priority queue, got %s", reflect.TypeOf(first))
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
	return model.setImageScanStatus(sha, ScanStatusRunningScanClient)
}

func (model *Model) finishRunningScanClient(image *Image, scanClientError error) error {
	imageInfo, ok := model.Images[image.Sha]

	// if we don't have this sha already, we don't need to do anything
	if !ok {
		return fmt.Errorf("finish running scan client -- expected to already have image %s, but did not", string(image.Sha))
	}

	scanStatus := ScanStatusRunningHubScan
	if scanClientError != nil {
		imageInfo.SetPriority(-1)
		scanStatus = ScanStatusInQueue
	}

	return model.setImageScanStatus(image.Sha, scanStatus)
}

func (model *Model) getShas(status ScanStatus) []DockerImageSha {
	shas := []DockerImageSha{}
	for sha, imageInfo := range model.Images {
		if imageInfo.ScanStatus == status {
			shas = append(shas, sha)
		}
	}
	return shas
}

func (model *Model) deletePod(podName string) error {
	_, ok := model.Pods[podName]
	if !ok {
		return fmt.Errorf("unable to delete pod %s, pod not found", podName)
	}
	delete(model.Pods, podName)
	return nil
}

func (model *Model) allPods(pods []Pod) error {
	model.Pods = map[string]Pod{}
	errors := []error{}
	for _, pod := range pods {
		err := model.addPod(pod)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return combineErrors("allPods", errors)
}

func (model *Model) allImages(images []Image) error {
	errors := []error{}
	for _, image := range images {
		err := model.addImage(image)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return combineErrors("allImages", errors)
}

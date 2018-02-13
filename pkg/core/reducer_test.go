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
	"testing"

	"reflect"

	"github.com/blackducksoftware/perceptor/pkg/hub"
	log "github.com/sirupsen/logrus"
)

func TestReducer(t *testing.T) {
	concurrentScanLimit := 1
	initialModel := NewModel(concurrentScanLimit)
	actions := make(chan action)
	reducer := newReducer(*initialModel, actions)

	image1 := *NewImage("image1", DockerImageSha("fe67acf"))
	image2 := *NewImage("image2", DockerImageSha("89ca3ec"))

	// 1. add a pod
	//   this should add all the images in the pod to the hub check queue (if they haven't already been added),
	//   add them to the image dictionary, and set their status to HubCheck
	go func() {
		actions <- addPod{*NewPod("pod1", "uid1", "namespace1", []Container{
			*NewContainer(image1, "container1"),
			*NewContainer(image2, "container2"),
		})}
	}()
	newModel := <-reducer.model
	if len(newModel.ImageHubCheckQueue) != 2 {
		t.Logf("expected there to be 2 images in queue, found %d", len(newModel.ImageHubCheckQueue))
		t.Fail()
	}
	if len(newModel.ImageScanQueue) != 0 {
		t.Logf("expected there to be 0 images in queue, found %d", len(newModel.ImageScanQueue))
		t.Fail()
	}
	imageResults1, ok1 := newModel.Images[image1.Sha]
	if !ok1 {
		t.Logf("couldn't find image1 in image map")
		t.Fail()
	}
	if imageResults1.ScanStatus != ScanStatusInHubCheckQueue {
		t.Logf("expected image1 ScanStatus to be InHubCheckQueue, but instead is %s", imageResults1.ScanStatus)
		t.Fail()
	}

	// 1a. move image1 from unknown into the hub check queue
	var nextCheckImage *Image
	go func() {
		actions <- getNextImageForHubPolling{func(image *Image) {
			nextCheckImage = image
		}}
	}()
	newModel = <-reducer.model
	if nextCheckImage == nil {
		t.Logf("expected to get an image for hub checking, got nothing")
		t.Fail()
	} else if *nextCheckImage != image1 {
		t.Logf("expected to get image1, got %s", nextCheckImage.HumanReadableName())
		t.Fail()
	}

	// 1b. move image1 from hub check queue into scan queue
	go func() {
		actions <- hubCheckResults{HubImageScan{
			Sha:  image1.Sha,
			Scan: nil,
		}}
	}()
	newModel = <-reducer.model

	// 2. ask for the next image from the queue. this should:
	//   remove the first item from the queue
	//   change its status to InProgress
	var nextImage *Image
	go func() {
		actions <- getNextImage{func(image *Image) {
			nextImage = image
		}}
	}()

	newModel = <-reducer.model
	if nextImage == nil {
		t.Logf("expected to get an image, got nothing")
		t.Fail()
	} else if *nextImage != image1 {
		t.Logf("expected to get image1, got %s", nextImage.HumanReadableName())
		t.Fail()
	}
	if len(newModel.ImageScanQueue) != 0 {
		t.Logf("expected there to be 0 images left in queue, found %d", len(newModel.ImageScanQueue))
		t.Fail()
	}
	imageResults2, ok2 := newModel.Images[image1.Sha]
	if !ok2 {
		t.Logf("couldn't find image1 in image map")
		t.Fail()
	}
	if imageResults2.ScanStatus != ScanStatusRunningScanClient {
		t.Logf("expected image1 ScanStatus to be RunningScanClient, but instead is %d", imageResults2.ScanStatus)
		t.Fail()
	}

	// 3. finish a scan
	//   this should cause the image status to be set to running hub scan,
	//   and results to be added in the image dict
	log.Infof("is nil 1? %t", nextImage == nil)
	go func() {
		log.Infof("is nil 2? %t", nextImage == nil)
		actions <- finishScanClient{(*nextImage).Sha, ""}
	}()

	newModel = <-reducer.model
	imageResults3, ok3 := newModel.Images[image1.Sha]
	if !ok3 {
		t.Logf("couldn't find image1 in image map")
		t.Fail()
	}
	if imageResults3.ScanStatus != ScanStatusRunningHubScan {
		t.Logf("expected image1 ScanStatus to be RunningHubScan, but instead is %d", imageResults3.ScanStatus)
		t.Fail()
	}

	// 4. ask for the next image from the queue. this hits the concurrency limit,
	//    so it should not do anything
	go func() {
		actions <- getNextImage{func(image *Image) {
			nextImage = image
		}}
	}()
	newModel = <-reducer.model
	if nextImage != nil {
		t.Logf("expected to not get an image, got %s", nextImage.HumanReadableName())
		t.Fail()
	}

	// 5. finish the hub scan for image1. this should:
	//    change the ScanStatus to complete
	//    add scan results
	go func() {
		actions <- hubScanResults{HubImageScan{
			Sha: image1.Sha,
			Scan: &hub.ImageScan{
				ScanSummary: hub.ScanSummary{Status: "COMPLETE"},
			},
		}}
	}()
	newModel = <-reducer.model
	imageResults5, ok5 := newModel.Images[image1.Sha]
	if !ok5 {
		t.Logf("couldn't find image1 in image map")
		t.Fail()
	}
	if imageResults5.ScanStatus != ScanStatusComplete {
		t.Logf("expected image1 ScanStatus to be Complete, but instead is %s", imageResults5.ScanStatus)
		t.Fail()
	}
	expected5 := hub.ImageScan{}
	actual5 := *imageResults5.ScanResults

	// without using reflect, we get an error:
	//   invalid operation: expected5 != actual5 (struct containing hub.RiskProfile cannot be compared)
	if reflect.DeepEqual(expected5, actual5) {
		t.Logf("expected scan results to be %v, found %v", expected5, actual5)
		t.Fail()
	}

	// 6a. move image2 from unknown into the hub check queue
	go func() {
		actions <- getNextImageForHubPolling{func(image *Image) {
			nextCheckImage = image
		}}
	}()
	newModel = <-reducer.model
	if nextCheckImage == nil {
		t.Logf("expected to get an image for hub checking, got nothing")
		t.Fail()
	} else if *nextCheckImage != image2 {
		t.Logf("expected to get image2, got %s", nextCheckImage.HumanReadableName())
		t.Fail()
	}
	imageResults6a, ok6a := newModel.Images[image2.Sha]
	if !ok6a {
		t.Logf("couldn't find image2 in image map")
		t.Fail()
	}
	if imageResults6a.ScanStatus != ScanStatusCheckingHub {
		t.Logf("expected image2 ScanStatus to be CheckingHub, but instead is %d", imageResults6a.ScanStatus)
		t.Fail()
	}

	// 6b. move image2 from hub check queue into scan queue
	go func() {
		actions <- hubCheckResults{HubImageScan{
			Sha:  image2.Sha,
			Scan: nil,
		}}
	}()
	newModel = <-reducer.model
	imageResults6b, ok6b := newModel.Images[image2.Sha]
	if !ok6b {
		t.Logf("couldn't find image2 in image map")
		t.Fail()
	}
	if imageResults6b.ScanStatus != ScanStatusInQueue {
		t.Logf("expected image2 ScanStatus to be InQueue, but instead is %d", imageResults6b.ScanStatus)
		t.Fail()
	}

	// 6c. ask for the next image from the queue. this should:
	//   remove the first item from the queue
	//   change its status to InProgress
	go func() {
		actions <- getNextImage{func(image *Image) {
			nextImage = image
		}}
	}()
	newModel = <-reducer.model
	if nextImage == nil {
		t.Logf("expected to get an image, got nothing")
		t.Fail()
	} else if *nextImage != image2 {
		t.Logf("expected to get image2, got %s", nextImage.HumanReadableName())
		t.Fail()
	}
	if len(newModel.ImageScanQueue) != 0 {
		t.Logf("expected the queue to be empty, found %d", len(newModel.ImageScanQueue))
		t.Fail()
	}
	imageResults6, ok6 := newModel.Images[image2.Sha]
	if !ok6 {
		t.Logf("couldn't find image2 in image map")
		t.Fail()
	}
	if imageResults6.ScanStatus != ScanStatusRunningScanClient {
		t.Logf("expected image2 ScanStatus to be RunningScanClient, but instead is %d", imageResults6.ScanStatus)
		t.Fail()
	}

	// 7. finish a scan with an error
	//   this should cause the image to get put back in the queue,
	//   and the status set back to InQueue
	go func() {
		actions <- finishScanClient{(*nextImage).Sha, "oops"}
	}()

	newModel = <-reducer.model
	imageResults7, ok7 := newModel.Images[image2.Sha]
	if !ok7 {
		t.Logf("couldn't find image2 in image map")
		t.Fail()
	}
	if imageResults7.ScanStatus != ScanStatusInQueue {
		t.Logf("expected image7 ScanStatus to be InQueue, but instead is %d", imageResults7.ScanStatus)
		t.Fail()
	}

	// 8. ask for next image, get image2 again
	log.Info("about to run gofunc for message 8")
	go func() {
		log.Info("send message 8")
		actions <- getNextImage{func(image *Image) {
			nextImage = image
		}}
		log.Info("finished sending message 8")
	}()
	log.Info("get model 8")
	newModel = <-reducer.model
	log.Info("finished getting model 8")
	if nextImage == nil {
		t.Logf("expected to get an image, got nothing")
		t.Fail()
	}
	if *nextImage != image2 {
		t.Logf("expected image name to be image2, got %s", nextImage.HumanReadableName())
	}
	imageResults8, ok8 := newModel.Images[image2.Sha]
	log.Info("check results 8")
	if !ok8 {
		t.Logf("couldn't find image2 in image map")
		t.Fail()
	}
	if imageResults8.ScanStatus != ScanStatusRunningScanClient {
		t.Logf("expected image7 ScanStatus to be InQueue, but instead is %d", imageResults8.ScanStatus)
		t.Fail()
	}

	// 9. finish scan client with success
	log.Info("about to run gofunc for message 9")
	go func() {
		log.Info("send message 9")
		actions <- finishScanClient{(*nextImage).Sha, ""}
		log.Info("finished sending message 9")
	}()
	newModel = <-reducer.model

	// 10. finish hub scan with success
	go func() {
		actions <- hubScanResults{HubImageScan{
			Sha: image2.Sha,
			Scan: &hub.ImageScan{
				ScanSummary: hub.ScanSummary{Status: "Complete"},
			},
		}}
	}()
	newModel = <-reducer.model

	// 11. ask for next image, get nil because queue is empty
	go func() {
		actions <- getNextImage{func(image *Image) {
			log.Infof("image: %v, %t", image, image == nil)
			nextImage = image
		}}
	}()
	newModel = <-reducer.model
	if nextImage != nil {
		t.Logf("expected to get nothing, got %v", nextImage)
		t.Fail()
	}

	log.Info("done with all messages")
}

func TestScanClientFails(t *testing.T) {
	concurrentScanLimit := 1
	model := NewModel(concurrentScanLimit)
	image := *NewImage("abc", DockerImageSha("23bcf2dae3"))
	model.AddImage(image)
	model.Images[image.Sha].setScanStatus(ScanStatusRunningScanClient)
	model.errorRunningScanClient(image.Sha)

	if model.Images[image.Sha].ScanStatus != ScanStatusInQueue {
		t.Logf("expected ScanStatus of InQueue, got %s", model.Images[image.Sha].ScanStatus)
		t.Fail()
	}

	nextImage := model.getNextImageFromScanQueue()
	if image != *nextImage {
		t.Logf("expected nextImage of %v, got %v", image, nextImage)
		t.Fail()
	}
}

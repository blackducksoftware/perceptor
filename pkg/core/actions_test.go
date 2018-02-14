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
	"encoding/json"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
)

func assertEqual(t *testing.T, actual interface{}, expected interface{}) {
	if actual == nil && expected == nil {
		return
	}
	if reflect.DeepEqual(actual, expected) {
		return
	}
	if actual == expected {
		return
	}
	bs1, err := json.Marshal(actual)
	if err != nil {
		t.Errorf("json serialization error: %s", err.Error())
		return
	}
	bs2, err := json.Marshal(expected)
	if err != nil {
		t.Errorf("json serialization error: %s", err.Error())
		return
	}
	if string(bs1) == string(bs2) {
		return
	}
	// t.Errorf("expected \n%+v, got \n%+v", expected, actual)
	t.Errorf("expected \n%s, got \n%s", string(bs1), string(bs2))
}

func TestActionsImplementInterface(t *testing.T) {
	processAction(addPod{Pod{}})
	processAction(updatePod{Pod{}})
	processAction(deletePod{})
	processAction(addImage{})
	processAction(allPods{})
	processAction(getNextImage{})
	processAction(finishScanClient{})
	processAction(getNextImageForHubPolling{})
	processAction(hubCheckResults{})
	processAction(hubScanResults{})
	processAction(requeueStalledScan{})
	processAction(setConcurrentScanLimit{})
	processAction(allImages{})
}

func processAction(nextAction action) {
	log.Infof("received actions: %+v", nextAction)
}

var sha1 = DockerImageSha("sha1")
var image1 = Image{Name: "image1", Sha: sha1}
var cont1 = Container{Image: image1}
var pod1 = Pod{Namespace: "abc", Name: "def", UID: "fff", Containers: []Container{cont1}}

func TestAddPodAction(t *testing.T) {
	// actual
	actual := addPod{pod: pod1}.apply(*NewModel(3))
	// expected (a bit hacky to get the times set up):
	//  - pod gets added to .Pods
	//  - all images within pod get added to .Images
	//  - all new images get added to hub check queue
	expected := *NewModel(3)
	expected.Pods[pod1.QualifiedName()] = pod1
	imageInfo := NewImageInfo(sha1, "image1")
	imageInfo.ScanStatus = ScanStatusInHubCheckQueue
	imageInfo.TimeOfLastStatusChange = actual.Images[sha1].TimeOfLastStatusChange
	expected.Images[sha1] = imageInfo
	expected.ImageHubCheckQueue = append(expected.ImageHubCheckQueue, imageInfo.image())
	//
	assertEqual(t, actual, expected)
}

func TestAddImageAction(t *testing.T) {
	// actual
	actual := addImage{image: image1}.apply(*NewModel(3))
	// expected (a bit hacky to get the times set up):
	//  - image gets added to .Images
	//  - image gets added to hub check queue
	expected := *NewModel(3)
	imageInfo := NewImageInfo(sha1, "image1")
	imageInfo.ScanStatus = ScanStatusInHubCheckQueue
	imageInfo.TimeOfLastStatusChange = actual.Images[sha1].TimeOfLastStatusChange
	expected.Images[sha1] = imageInfo
	expected.ImageHubCheckQueue = append(expected.ImageHubCheckQueue, imageInfo.image())
	//
	assertEqual(t, actual, expected)
}

// TODO allPods

// TODO allImages

func TestGetNextImageForScanningActionNoImageAvailable(t *testing.T) {
	// actual
	var nextImage *Image
	actual := getNextImage{continuation: func(image *Image) {
		nextImage = image
	}}.apply(*NewModel(3))
	// expected: front image removed from scan queue, status and time of image changed
	expected := *NewModel(3)

	assertEqual(t, nextImage, nil)
	log.Infof("%+v, %+v", actual, expected)
	// assertEqual(t, actual, expected)
}

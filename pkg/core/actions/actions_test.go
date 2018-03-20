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

	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	"github.com/blackducksoftware/perceptor/pkg/hub"
	log "github.com/sirupsen/logrus"
)

var sha1 m.DockerImageSha
var sha2 m.DockerImageSha
var sha3 m.DockerImageSha
var image1 m.Image
var image2 m.Image
var image3 m.Image
var cont1 m.Container
var cont2 m.Container
var cont3 m.Container
var pod1 m.Pod
var pod2 m.Pod
var pod3 m.Pod

func init() {
	sha1 = m.DockerImageSha("sha1")
	image1 = *m.NewImage("image1", sha1)
	sha2 = m.DockerImageSha("sha2")
	image2 = *m.NewImage("image2", sha2)
	sha3 = m.DockerImageSha("sha3")
	image3 = *m.NewImage("image3", sha3)
	cont1 = *m.NewContainer(image1, "cont1")
	cont2 = *m.NewContainer(image2, "cont2")
	cont3 = *m.NewContainer(image3, "cont3")
	pod1 = *m.NewPod("pod1", "pod1uid", "ns1", []m.Container{cont1, cont2})
	pod2 = *m.NewPod("pod2", "pod2uid", "ns1", []m.Container{cont1})
	pod3 = *m.NewPod("pod3", "pod3uid", "ns3", []m.Container{cont3})
}

func createNewModel1() *m.Model {
	model := m.NewModel(&m.Config{ConcurrentScanLimit: 3}, "test version")
	model.AddPod(pod1)
	model.AddPod(pod2)
	model.Images[sha1].ScanStatus = m.ScanStatusComplete
	model.Images[sha1].ScanResults = &hub.ImageScan{
		PolicyStatus: hub.PolicyStatus{
			OverallStatus:                hub.PolicyStatusTypeInViolation,
			ComponentVersionStatusCounts: map[hub.PolicyStatusType]int{hub.PolicyStatusTypeInViolation: 3}}}
	return model
}

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
	processAction(&AddPod{m.Pod{}})
	processAction(&UpdatePod{m.Pod{}})
	processAction(&DeletePod{})
	processAction(&AddImage{})
	processAction(&AllPods{})
	processAction(&GetNextImage{})
	processAction(&FinishScanClient{})
	processAction(&GetInitialHubCheckImage{})
	processAction(&InitialHubCheckResults{})
	processAction(&HubCheckResults{})
	processAction(&RequeueStalledScans{})
	processAction(&SetConcurrentScanLimit{})
	processAction(&AllImages{})
	processAction(&GetModel{})
	processAction(&GetMetrics{})
	processAction(&GetScanResults{})
	processAction(&GetRunningHubScans{})
	processAction(&HubRecheckResults{})
	processAction(&GetCompletedScans{})
}

func processAction(nextAction Action) {
	log.Infof("received actions: %+v, %s", nextAction, reflect.TypeOf(nextAction))
}

var testSha = m.DockerImageSha("sha1")
var testImage = m.Image{Name: "image1", Sha: testSha}
var testCont = m.Container{Image: testImage}
var testPod = m.Pod{Namespace: "abc", Name: "def", UID: "fff", Containers: []m.Container{testCont}}

func TestAddPodAction(t *testing.T) {
	// actual
	actual := m.NewModel(&m.Config{}, "test version")
	(&AddPod{testPod}).Apply(actual)
	// expected (a bit hacky to get the times set up):
	//  - pod gets added to .Pods
	//  - all images within pod get added to .Images
	//  - all new images get added to hub check queue
	expected := *m.NewModel(&m.Config{}, "test version")
	expected.Pods[testPod.QualifiedName()] = testPod
	imageInfo := m.NewImageInfo(testSha, "image1")
	imageInfo.ScanStatus = m.ScanStatusInHubCheckQueue
	imageInfo.TimeOfLastStatusChange = actual.Images[testSha].TimeOfLastStatusChange
	expected.Images[testSha] = imageInfo
	expected.ImageHubCheckQueue = append(expected.ImageHubCheckQueue, imageInfo.Image())
	//
	assertEqual(t, actual, expected)
}

func TestAddImageAction(t *testing.T) {
	// actual
	actual := m.NewModel(&m.Config{ConcurrentScanLimit: 3}, "test version")
	(&AddImage{testImage}).Apply(actual)
	// expected (a bit hacky to get the times set up):
	//  - image gets added to .Images
	//  - image gets added to hub check queue
	expected := *m.NewModel(&m.Config{ConcurrentScanLimit: 3}, "test version")
	imageInfo := m.NewImageInfo(testSha, "image1")
	imageInfo.ScanStatus = m.ScanStatusInHubCheckQueue
	imageInfo.TimeOfLastStatusChange = actual.Images[testSha].TimeOfLastStatusChange
	expected.Images[testSha] = imageInfo
	expected.ImageHubCheckQueue = append(expected.ImageHubCheckQueue, imageInfo.Image())
	//
	assertEqual(t, actual, expected)
}

// AllPods does remove pre-existing pods
func TestAllPods(t *testing.T) {
	actual := createNewModel1()
	(&AllPods{}).Apply(actual)
	if len(actual.Pods) != 0 {
		t.Errorf("expected 0 pods, found %d", len(actual.Pods))
	}
}

// AllImages doesn't remove pre-existing images
func TestAllImages(t *testing.T) {
	actual := createNewModel1()
	(&AllImages{}).Apply(actual)
	if len(actual.Images) != 2 {
		t.Errorf("expected 2 images, found %d", len(actual.Images))
	}
}

func TestGetNextImageForScanningActionNoImageAvailable(t *testing.T) {
	// actual
	var nextImage *m.Image
	actual := m.NewModel(&m.Config{ConcurrentScanLimit: 3}, "test version")
	(&GetNextImage{func(image *m.Image) {
		nextImage = image
	}}).Apply(actual)
	// expected: front image removed from scan queue, status and time of image changed
	expected := *m.NewModel(&m.Config{ConcurrentScanLimit: 3}, "test version")

	assertEqual(t, nextImage, nil)
	log.Infof("%+v, %+v", actual, expected)
	// assertEqual(t, actual, expected)
}

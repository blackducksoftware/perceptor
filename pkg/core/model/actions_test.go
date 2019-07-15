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
	"encoding/json"
	"fmt"

	"github.com/blackducksoftware/perceptor/pkg/hub"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var (
	sha1   = DockerImageSha("sha1")
	image1 = *NewImage("image1", "1", sha1, 1, "Project Image1", "1.0")
	sha2   = DockerImageSha("sha2")
	image2 = *NewImage("image2", "2", sha2, 2, "Project Image2", "2.0")
	sha3   = DockerImageSha("sha3")
	image3 = *NewImage("image3", "3", sha3, 3, "Project Image3", "3.0")
	cont1  = *NewContainer(image1, "cont1")
	cont2  = *NewContainer(image2, "cont2")
	cont3  = *NewContainer(image3, "cont3")
	pod1   = *NewPod("pod1", "pod1uid", "ns1", []Container{cont1, cont2})
	pod2   = *NewPod("pod2", "pod2uid", "ns1", []Container{cont1})
	pod3   = *NewPod("pod3", "pod3uid", "ns3", []Container{cont3})
	// this is ridiculous, but let's create a pod with 0 containers
	pod4 = *NewPod("pod4", "pod4uid", "ns4", []Container{})
)

var (
	testSha   = DockerImageSha("sha1")
	testImage = Image{Repository: "image1", Tag: "", Sha: testSha, Priority: 1, BlackDuckProjectName: "Project Image1", BlackDuckProjectVersion: "1.0"}
	testCont  = Container{Image: testImage}
	testPod   = Pod{Namespace: "abc", Name: "def", UID: "fff", Containers: []Container{testCont}}
)

func checkModelEquality(m1 *Model, m2 *Model) {
	ji1, err := json.Marshal(m1.Images)
	Expect(err).To(BeNil())
	ji2, err := json.Marshal(m2.Images)
	Expect(err).To(BeNil())
	Expect(ji1).To(Equal(ji2))
	Expect(m1.ImageScanQueue).To(Equal(m2.ImageScanQueue))
	Expect(m1.Pods).To(Equal(m2.Pods))
}

func createNewModel1() *Model {
	model := NewModel()
	model.addPod(pod1)
	model.addPod(pod2)
	model.Images[sha1].ScanStatus = ScanStatusComplete
	model.Images[sha1].SetScanResults(&hub.ScanResults{
		PolicyStatus: hub.PolicyStatus{
			OverallStatus:                hub.PolicyStatusTypeInViolation,
			ComponentVersionStatusCounts: map[string]int{hub.PolicyStatusTypeInViolation: 3}}})
	return model
}

func createNewModel2() *Model {
	model := NewModel()
	model.addPod(pod1)
	model.addPod(pod2)
	model.addPod(pod3)
	model.addPod(pod4)
	model.Images[sha1].ScanStatus = ScanStatusComplete
	model.Images[sha1].SetScanResults(&hub.ScanResults{
		PolicyStatus: hub.PolicyStatus{
			OverallStatus:                hub.PolicyStatusTypeInViolation,
			ComponentVersionStatusCounts: map[string]int{hub.PolicyStatusTypeInViolation: 3}}})
	model.Images[sha3].ScanStatus = ScanStatusComplete
	model.Images[sha3].SetScanResults(&hub.ScanResults{
		PolicyStatus: hub.PolicyStatus{
			OverallStatus: hub.PolicyStatusTypeNotInViolation,
		},
	})
	return model
}

func RunActionTests() {
	Describe("addImage", func() {
		It("should add an image", func() {
			actual := NewModel()
			Expect(actual.addImage(testImage)).To(BeNil())
			// expected (a bit hacky to get the times set up):
			//  - image gets added to .Images
			//  - image gets added to hub check queue
			expected := *NewModel()
			imageInfo := NewImageInfo(testImage, &RepoTag{Repository: "image1", Tag: ""})
			imageInfo.ScanStatus = ScanStatusUnknown
			imageInfo.TimeOfLastStatusChange = actual.Images[testSha].TimeOfLastStatusChange
			expected.Images[testSha] = imageInfo
			//
			checkModelEquality(actual, &expected)
		})
	})
	Describe("addPod", func() {
		It("should add a pod and all the pod's containers' images", func() {
			actual := NewModel()
			Expect(actual.addPod(testPod)).To(BeNil())
			// expected (a bit hacky to get the times set up):
			//  - pod gets added to .Pods
			//  - all images within pod get added to .Images
			//  - all new images get added to hub check queue
			expected := *NewModel()
			expected.Pods[testPod.QualifiedName()] = testPod
			imageInfo := NewImageInfo(testImage, &RepoTag{Repository: "image1", Tag: ""})
			imageInfo.ScanStatus = ScanStatusUnknown
			imageInfo.TimeOfLastStatusChange = actual.Images[testSha].TimeOfLastStatusChange
			expected.Images[testSha] = imageInfo
			//
			checkModelEquality(actual, &expected)
		})
	})
	Describe("allImages", func() {
		It("should not remove pre-existing images", func() {
			actual := createNewModel1()
			Expect(actual.allImages([]Image{})).To(BeNil())
			Expect(len(actual.Images)).To(Equal(2))
		})
	})
	Describe("allPods", func() {
		It("should remove pre-existing pods", func() {
			actual := createNewModel1()
			Expect(actual.allPods([]Pod{})).To(BeNil())
			Expect(len(actual.Pods)).To(Equal(0))
		})
	})
	Describe("FinishScanClient", func() {
		It("handles failures", func() {
			model := NewModel()
			image := *NewImage("abc", "4.0", DockerImageSha("23bcf2dae3"), -1, "", "")
			model.setImageScanStatus(image.Sha, ScanStatusInQueue)
			model.setImageScanStatus(image.Sha, ScanStatusRunningScanClient)
			err := model.finishRunningScanClient(&image, fmt.Errorf("oops, unable to run scan client"))
			Expect(err).ToNot(BeNil())
		})
	})
	Describe("GetModel", func() {
		It("should get the right numbers of pods and images", func() {
			model := createNewModel2()
			apiModel := model.GetModel()
			Expect(len(apiModel.Images)).To(Equal(3))
			Expect(len(apiModel.Pods)).To(Equal(4))
		})
	})
	Describe("GetNextImage", func() {
		It("no image available", func() {
			// actual
			actual := NewModel()
			nextImage, err := actual.getNextImageFromScanQueue()
			Expect(err).To(BeNil())
			// expected: front image removed from scan queue, status and time of image changed
			expected := NewModel()

			Expect(nextImage).To(BeNil())
			log.Infof("%+v, %+v", actual, expected)
			checkModelEquality(actual, expected)
		})

		It("regular", func() {
			model := NewModel()
			model.addImage(image1)
			model.setImageScanStatus(image1.Sha, ScanStatusInQueue)

			nextImage, err := model.getNextImageFromScanQueue()
			Expect(err).To(BeNil())

			expected := NewModel()
			expected.addImage(image1)
			expected.setImageScanStatus(image1.Sha, ScanStatusInQueue)
			expected.Images[sha1].TimeOfLastStatusChange = model.Images[sha1].TimeOfLastStatusChange

			Expect(*nextImage).To(Equal(image1))
			checkModelEquality(model, expected)
			Expect(model.ImageScanQueue.Values()).To(Equal([]interface{}{sha1}))
			Expect(model.Images[image1.Sha].ScanStatus).To(Equal(ScanStatusInQueue))
			// TODO expected: time of image changed
		})
	})
	Describe("test get full scan results", func() {
		model := createNewModel1()
		scanResults, err := scanResults(model)
		It("should produce the right number of pods, images, data, and policy violations", func() {
			Expect(err).To(BeNil())
			Expect(len(scanResults.Pods)).To(Equal(1))
			Expect(scanResults.Pods[0].Name).To(Equal("pod2"))
			Expect(len(scanResults.Images)).To(Equal(1))
			Expect(scanResults.Images[0].PolicyViolations).To(Equal(3))
		})
	})

	Describe("test pod overall status", func() {
		model := createNewModel2()
		It("should get nil scan results for pod 1", func() {
			scan1, err := scanResultsForPod(model, pod1.QualifiedName())
			Expect(err).To(BeNil())
			Expect(scan1).To(BeNil())
		})

		It("should get the right scan results for pod 2", func() {
			scan2, err := scanResultsForPod(model, pod2.QualifiedName())
			Expect(err).To(BeNil())
			Expect(scan2.PolicyViolations).To(Equal(3))
			Expect(scan2.Vulnerabilities).To(Equal(0))
			Expect(scan2.OverallStatus).To(Equal(hub.PolicyStatusTypeInViolation))
		})

		It("should get the right results for pod 3", func() {
			scan3, err := scanResultsForPod(model, pod3.QualifiedName())
			Expect(err).To(BeNil())
			Expect(scan3.PolicyViolations).To(Equal(0))
			Expect(scan3.Vulnerabilities).To(Equal(0))
			Expect(scan3.OverallStatus).To(Equal(hub.PolicyStatusTypeNotInViolation))
		})

		It("should get the right results for pod 4", func() {
			scan4, err := scanResultsForPod(model, pod4.QualifiedName())
			Expect(err).To(BeNil())
			Expect(scan4.PolicyViolations).To(Equal(0))
			Expect(scan4.Vulnerabilities).To(Equal(0))
			Expect(scan4.OverallStatus).To(Equal(hub.PolicyStatusTypeNotInViolation))
		})

		It("should get the right results for image 1", func() {
			imageScan1, err := scanResultsForImage(model, image1.Sha)
			Expect(err).To(BeNil())
			Expect(imageScan1.PolicyViolations).To(Equal(3))
			Expect(imageScan1.Vulnerabilities).To(Equal(0))
			Expect(imageScan1.OverallStatus).To(Equal(hub.PolicyStatusTypeInViolation))
		})

		It("should get nil scan results for image 2", func() {
			imageScan2, err := scanResultsForImage(model, image2.Sha)
			Expect(err).To(BeNil())
			Expect(imageScan2).To(BeNil())
		})

		It("should get the right results for image 3", func() {
			imageScan3, err := scanResultsForImage(model, image3.Sha)
			Expect(err).To(BeNil())
			Expect(imageScan3.PolicyViolations).To(Equal(0))
			Expect(imageScan3.Vulnerabilities).To(Equal(0))
			Expect(imageScan3.OverallStatus).To(Equal(hub.PolicyStatusTypeNotInViolation))
		})
	})
	Describe("image", func() {
		It("should unmarshal from JSON correctly", func() {
			jsonString := `{"Repository":"docker.io/mfenwickbd/perceptor","Sha":"04bb619150cd99cfb21e76429c7a5c2f4545775b07456cb6b9c866c8aff9f9e5","Tag":"latest"}`
			var image Image
			err := json.Unmarshal([]byte(jsonString), &image)
			Expect(err).To(BeNil())
			Expect(image.Repository).To(Equal("docker.io/mfenwickbd/perceptor"))
		})

		It("default hub data", func() {
			sha := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
			image := NewImage("abc", "latest", DockerImageSha(sha), 0, "", "")
			Expect(image.GetBlackDuckProjectName()).To(Equal("abc"))
			Expect(image.GetBlackDuckProjectVersionName()).To(Equal("latest-" + sha[:20]))
			Expect(image.GetBlackDuckScanName()).To(Equal(sha))
		})
		It("missing tag", func() {
			sha := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
			image := NewImage("abc", "", DockerImageSha(sha), 0, "", "")
			Expect(image.GetBlackDuckProjectName()).To(Equal("abc"))
			Expect(image.GetBlackDuckProjectVersionName()).To(Equal(sha[:20]))
			Expect(image.GetBlackDuckScanName()).To(Equal(sha))
		})
		It("specific hub data", func() {
			sha := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
			image := NewImage("abc", "", DockerImageSha(sha), 0, "def", "ghi")
			Expect(image.GetBlackDuckProjectName()).To(Equal("def"))
			Expect(image.GetBlackDuckProjectVersionName()).To(Equal("ghi"))
			Expect(image.GetBlackDuckScanName()).To(Equal(sha))
		})
	})
	Describe("metrics", func() {
		It("should handle metrics without panicing", func() {
			recordStateTransition(ScanStatusUnknown, ScanStatusComplete, false)
			recordEvent("abc")
			recordActionError("def")
			recordSetImagePriority(0, 1)
			Expect(1).To(Equal(1))
		})
	})
}

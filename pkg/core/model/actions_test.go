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
	"reflect"

	"github.com/blackducksoftware/perceptor/pkg/hub"
	. "github.com/onsi/ginkgo"
	log "github.com/sirupsen/logrus"
)

var (
	sha1   = DockerImageSha("sha1")
	image1 = *NewImage("image1", "1", sha1)
	sha2   = DockerImageSha("sha2")
	image2 = *NewImage("image2", "2", sha2)
	sha3   = DockerImageSha("sha3")
	image3 = *NewImage("image3", "3", sha3)
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
	testImage = Image{Repository: "image1", Tag: "", Sha: testSha}
	testCont  = Container{Image: testImage}
	testPod   = Pod{Namespace: "abc", Name: "def", UID: "fff", Containers: []Container{testCont}}
)

func createNewModel1() *Model {
	model := NewModel()
	model.AddPod(pod1)
	model.AddPod(pod2)
	model.Images[sha1].ScanStatus = ScanStatusComplete
	model.Images[sha1].SetScanResults(&hub.ScanResults{
		PolicyStatus: hub.PolicyStatus{
			OverallStatus:                hub.PolicyStatusTypeInViolation,
			ComponentVersionStatusCounts: map[hub.PolicyStatusType]int{hub.PolicyStatusTypeInViolation: 3}}})
	return model
}

func createNewModel2() *Model {
	model := NewModel()
	model.AddPod(pod1)
	model.AddPod(pod2)
	model.AddPod(pod3)
	model.AddPod(pod4)
	model.Images[sha1].ScanStatus = ScanStatusComplete
	model.Images[sha1].SetScanResults(&hub.ScanResults{
		PolicyStatus: hub.PolicyStatus{
			OverallStatus:                hub.PolicyStatusTypeInViolation,
			ComponentVersionStatusCounts: map[hub.PolicyStatusType]int{hub.PolicyStatusTypeInViolation: 3}}})
	model.Images[sha3].ScanStatus = ScanStatusComplete
	model.Images[sha3].SetScanResults(&hub.ScanResults{
		PolicyStatus: hub.PolicyStatus{
			OverallStatus: hub.PolicyStatusTypeNotInViolation,
		},
	})
	return model
}

func RunActionTests() {
	Describe("Actions", func() {
		It("implement interface", func() {
			processAction(&AddPod{Pod{}})
			processAction(&UpdatePod{Pod{}})
			processAction(&DeletePod{})
			processAction(&AddImage{})
			processAction(&AllPods{})
			processAction(&GetNextImage{})
			processAction(&FinishScanClient{})
			processAction(&AllImages{})
			processAction(&GetModel{})
			processAction(&GetMetrics{})
			processAction(&GetScanResults{})
		})
	})
}

func processAction(nextAction Action) {
	log.Infof("received actions: %+v, %s", nextAction, reflect.TypeOf(nextAction))
}

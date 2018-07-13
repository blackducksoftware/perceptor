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

package actions

import (
	"reflect"

	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	"github.com/blackducksoftware/perceptor/pkg/hub"
	. "github.com/onsi/ginkgo"
	log "github.com/sirupsen/logrus"
)

var (
	layer1    = "abcdef1234567890"
	layer2    = "0987654321fedcba"
	layers1   = []string{layer1}
	layers2   = []string{layer2, layer1}
	sha1      = m.DockerImageSha("sha1")
	image1    = *m.NewImage("image1", sha1)
	sha2      = m.DockerImageSha("sha2")
	image2    = *m.NewImage("image2", sha2)
	sha3      = m.DockerImageSha("sha3")
	image3    = *m.NewImage("image3", sha3)
	cont1     = *m.NewContainer(image1, "cont1")
	cont2     = *m.NewContainer(image2, "cont2")
	cont3     = *m.NewContainer(image3, "cont3")
	pod1      = *m.NewPod("pod1", "pod1uid", "ns1", []m.Container{cont1, cont2})
	pod2      = *m.NewPod("pod2", "pod2uid", "ns1", []m.Container{cont1})
	pod3      = *m.NewPod("pod3", "pod3uid", "ns3", []m.Container{cont3})
	testSha   = m.DockerImageSha("sha1")
	testImage = m.Image{Name: "image1", Sha: testSha}
	testCont  = m.Container{Image: testImage}
	testPod   = m.Pod{Namespace: "abc", Name: "def", UID: "fff", Containers: []m.Container{testCont}}
)

func createNewModel1() *m.Model {
	model := m.NewModel("test version", &m.Config{ConcurrentScanLimit: 3}, nil)
	model.AddPod(pod1)
	model.AddPod(pod2)
	model.SetLayersForImage(image1.Sha, layers1)
	model.SetLayerScanStatus(layer1, m.ScanStatusRunningHubScan)
	model.SetLayerScanStatus(layer1, m.ScanStatusComplete)
	model.Layers[layer1].SetScanResults(&hub.ScanResults{
		PolicyStatus: hub.PolicyStatus{
			OverallStatus:                hub.PolicyStatusTypeInViolation,
			ComponentVersionStatusCounts: map[hub.PolicyStatusType]int{hub.PolicyStatusTypeInViolation: 3}}})
	return model
}

func RunActionTests() {
	Describe("Actions", func() {
		It("implement interface", func() {
			processAction(&AddPod{m.Pod{}})
			processAction(&UpdatePod{m.Pod{}})
			processAction(&DeletePod{})
			processAction(&AddImage{})
			processAction(&AllPods{})
			processAction(&GetNextImage{})
			processAction(&FinishScanClient{})
			processAction(&CheckScanInitial{})
			processAction(&FetchScanInitial{})
			processAction(&FetchScanCompletion{})
			processAction(&RequeueStalledScans{})
			processAction(&SetConfig{})
			processAction(&AllImages{})
			processAction(&GetModel{})
			processAction(&GetMetrics{})
			processAction(&GetScanResults{})
			processAction(&CheckScansCompletion{})
			processAction(&FetchScanRefresh{})
			processAction(&CheckScanRefresh{})
			processAction(&SetIsHubEnabled{})
		})
	})
}

func processAction(nextAction Action) {
	log.Infof("received actions: %+v, %s", nextAction, reflect.TypeOf(nextAction))
}

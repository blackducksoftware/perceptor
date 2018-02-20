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
	"testing"

	"github.com/blackducksoftware/perceptor/pkg/hub"
	"github.com/prometheus/common/log"
)

var sha1 DockerImageSha
var sha2 DockerImageSha
var sha3 DockerImageSha
var image1 Image
var image2 Image
var image3 Image
var cont1 Container
var cont2 Container
var cont3 Container
var pod1 Pod
var pod2 Pod
var pod3 Pod

func init() {
	sha1 = DockerImageSha("sha1")
	image1 = *NewImage("image1", sha1)
	sha2 = DockerImageSha("sha2")
	image2 = *NewImage("image2", sha2)
	sha3 = DockerImageSha("sha3")
	image3 = *NewImage("image3", sha3)
	cont1 = *NewContainer(image1, "cont1")
	cont2 = *NewContainer(image2, "cont2")
	cont3 = *NewContainer(image3, "cont3")
	pod1 = *NewPod("pod1", "pod1uid", "ns1", []Container{cont1, cont2})
	pod2 = *NewPod("pod2", "pod2uid", "ns1", []Container{cont1})
	pod3 = *NewPod("pod3", "pod3uid", "ns3", []Container{cont3})
}

func createNewModel1() *Model {
	model := NewModel(3, PerceptorConfig{})
	model.AddPod(pod1)
	model.AddPod(pod2)
	model.Images[sha1].ScanStatus = ScanStatusComplete
	model.Images[sha1].ScanResults = &hub.ImageScan{
		PolicyStatus: hub.PolicyStatus{
			OverallStatus:                hub.PolicyStatusTypeInViolation,
			ComponentVersionStatusCounts: map[hub.PolicyStatusType]int{hub.PolicyStatusTypeInViolation: 3}}}
	return model
}

func createNewModel2() *Model {
	model := NewModel(3, PerceptorConfig{})
	model.AddPod(pod1)
	model.AddPod(pod2)
	model.AddPod(pod3)
	model.Images[sha1].ScanStatus = ScanStatusComplete
	model.Images[sha1].ScanResults = &hub.ImageScan{
		PolicyStatus: hub.PolicyStatus{
			OverallStatus:                hub.PolicyStatusTypeInViolation,
			ComponentVersionStatusCounts: map[hub.PolicyStatusType]int{hub.PolicyStatusTypeInViolation: 3}}}
	model.Images[sha3].ScanStatus = ScanStatusComplete
	model.Images[sha3].ScanResults = &hub.ImageScan{
		PolicyStatus: hub.PolicyStatus{
			OverallStatus: hub.PolicyStatusTypeNotInViolation,
		},
	}
	return model
}

func TestGetFullScanResults(t *testing.T) {
	model := createNewModel1()
	scanResults := model.scanResults()
	if len(scanResults.Pods) != 1 {
		t.Errorf("expected 1 finished pod, found %d", len(scanResults.Pods))
	}
	actualPodName := scanResults.Pods[0].Name
	expectedPodName := "pod2"
	if actualPodName != expectedPodName {
		t.Errorf("expected pod name of %s, found %s", expectedPodName, actualPodName)
	}
	if len(scanResults.Images) != 1 {
		t.Errorf("expected 1 finished image, found %d", len(scanResults.Images))
	}
	actualPolicyViolations := scanResults.Images[0].PolicyViolations
	if actualPolicyViolations != 3 {
		t.Errorf("expected 3 policy violations, found %d", actualPolicyViolations)
	}
}

func TestPodOverallStatus(t *testing.T) {
	model := createNewModel2()
	pv1, vc1, os1, err := model.scanResultsForPod("ns1/pod1")
	if err != nil {
		jsonBytes, _ := json.Marshal(model)
		log.Infof("model: %s", string(jsonBytes))
		panic(err)
	}
	if pv1 != 3 {
		t.Errorf("expected 3 policy violations, found %d", pv1)
	}
	if vc1 != 0 {
		t.Errorf("expected 0 vulnerabilities, found %d", vc1)
	}
	if os1 != "IN_VIOLATION" {
		t.Errorf("expected overall status of IN_VIOLATION, found <%s>", os1)
	}
	pv2, vc2, os2, err := model.scanResultsForPod("ns1/pod2")
	if err != nil {
		panic(err)
	}
	if pv2 != 3 {
		t.Errorf("expected 0 policy violations, found %d", pv2)
	}
	if vc2 != 0 {
		t.Errorf("expected 0 vulnerabilities, found %d", vc2)
	}
	if os2 != "IN_VIOLATION" {
		t.Errorf("expected overall status of IN_VIOLATION, found <%s>", os2)
	}
}

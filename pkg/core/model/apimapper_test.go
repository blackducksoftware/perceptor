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
	"testing"

	"github.com/blackducksoftware/perceptor/pkg/hub"
	log "github.com/sirupsen/logrus"
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
var pod4 Pod

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
	// this is ridiculous, but let's create a pod with 0 containers
	pod4 = *NewPod("pod4", "pod4uid", "ns4", []Container{})
}

func createNewModel1() *Model {
	model := NewModel(&Config{ConcurrentScanLimit: 3}, "test version")
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
	model := NewModel(&Config{ConcurrentScanLimit: 3}, "test version")
	model.AddPod(pod1)
	model.AddPod(pod2)
	model.AddPod(pod3)
	model.AddPod(pod4)
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
	scanResults := model.ScanResults()
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
	scan1, err := model.ScanResultsForPod(pod1.QualifiedName())
	if err != nil {
		jsonBytes, _ := json.Marshal(model)
		log.Infof("model: %s", string(jsonBytes))
		panic(err)
	}
	if scan1 != nil {
		t.Errorf("expected nil scan results for pod %s, found %+v", pod1.QualifiedName(), scan1)
	}

	scan2, err := model.ScanResultsForPod(pod2.QualifiedName())
	if err != nil {
		panic(err)
	}
	if scan2.PolicyViolations != 3 {
		t.Errorf("expected 0 policy violations, found %d", scan2.PolicyViolations)
	}
	if scan2.Vulnerabilities != 0 {
		t.Errorf("expected 0 vulnerabilities, found %d", scan2.Vulnerabilities)
	}
	if scan2.OverallStatus != "IN_VIOLATION" {
		t.Errorf("expected overall status of IN_VIOLATION, found <%s>", scan2.OverallStatus)
	}

	scan3, err := model.ScanResultsForPod(pod3.QualifiedName())
	if err != nil {
		panic(err)
	}
	if scan3.PolicyViolations != 0 {
		t.Errorf("expected 0 policy violations, found %d", scan3.PolicyViolations)
	}
	if scan3.Vulnerabilities != 0 {
		t.Errorf("expected 0 vulnerabilities, found %d", scan3.Vulnerabilities)
	}
	if scan3.OverallStatus != "NOT_IN_VIOLATION" {
		t.Errorf("expected overall status of NOT_IN_VIOLATION, found <%s>", scan3.OverallStatus)
	}

	scan4, err := model.ScanResultsForPod(pod4.QualifiedName())
	if err != nil {
		panic(err)
	}
	if scan4.PolicyViolations != 0 {
		t.Errorf("expected 0 policy violations, found %d", scan4.PolicyViolations)
	}
	if scan4.Vulnerabilities != 0 {
		t.Errorf("expected 0 vulnerabilities, found %d", scan4.Vulnerabilities)
	}
	if scan4.OverallStatus != "NOT_IN_VIOLATION" {
		t.Errorf("expected overall status of NOT_IN_VIOLATION, found <%s>", scan4.OverallStatus)
	}

	imageScan1, err := model.ScanResultsForImage(image1.Sha)
	if err != nil {
		panic(err)
	}
	if imageScan1.PolicyViolations != 3 {
		t.Errorf("expected 0 policy violations, found %d", imageScan1.PolicyViolations)
	}
	if imageScan1.Vulnerabilities != 0 {
		t.Errorf("expected 0 vulnerabilities, found %d", imageScan1.Vulnerabilities)
	}
	if imageScan1.OverallStatus.String() != "IN_VIOLATION" {
		t.Errorf("expected overall status of IN_VIOLATION, found <%s>", imageScan1.OverallStatus)
	}

	imageScan2, err := model.ScanResultsForImage(image2.Sha)
	if err != nil {
		panic(err)
	}
	if imageScan2 != nil {
		t.Errorf("expected nil scan results, got %+v", imageScan2)
	}

	imageScan3, err := model.ScanResultsForImage(image3.Sha)
	if err != nil {
		panic(err)
	}
	if imageScan3.PolicyViolations != 0 {
		t.Errorf("expected 0 policy violations, found %d", imageScan3.PolicyViolations)
	}
	if imageScan3.Vulnerabilities != 0 {
		t.Errorf("expected 0 vulnerabilities, found %d", imageScan3.Vulnerabilities)
	}
	if imageScan3.OverallStatus.String() != "NOT_IN_VIOLATION" {
		t.Errorf("expected overall status of NOT_IN_VIOLATION, found <%s>", imageScan3.OverallStatus)
	}
}

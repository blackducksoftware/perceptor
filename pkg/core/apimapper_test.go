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

	"github.com/blackducksoftware/perceptor/pkg/hub"
)

func TestGetFullScanResults(t *testing.T) {
	model := NewModel(3)
	sha1 := DockerImageSha("sha1")
	image1 := *NewImage("image1", sha1)
	sha2 := DockerImageSha("sha2")
	image2 := *NewImage("image2", sha2)
	cont1 := *NewContainer(image1, "cont1")
	cont2 := *NewContainer(image2, "cont2")
	pod1 := *NewPod("pod1", "pod1uid", "ns1", []Container{cont1, cont2})
	pod2 := *NewPod("pod2", "pod2uid", "ns1", []Container{cont1})
	model.AddPod(pod1)
	model.AddPod(pod2)
	model.Images[sha1].ScanStatus = ScanStatusComplete
	model.Images[sha1].ScanResults = &hub.ImageScan{
		PolicyStatus: hub.PolicyStatus{
			ComponentVersionStatusCounts: map[string]int{"IN_VIOLATION": 3}}}

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

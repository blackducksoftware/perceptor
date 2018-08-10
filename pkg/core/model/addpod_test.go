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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func RunTestAddPodAction() {
	It("should add a pod and all the pod's containers' images", func() {
		actual := NewModel()
		(&AddPod{testPod}).Apply(actual)
		// expected (a bit hacky to get the times set up):
		//  - pod gets added to .Pods
		//  - all images within pod get added to .Images
		//  - all new images get added to hub check queue
		expected := *NewModel()
		expected.ImagePriority[testImage.Sha] = 1
		expected.Pods[testPod.QualifiedName()] = testPod
		imageInfo := NewImageInfo(testSha, &RepoTag{Repository: "image1", Tag: ""})
		imageInfo.ScanStatus = ScanStatusUnknown
		imageInfo.TimeOfLastStatusChange = actual.Images[testSha].TimeOfLastStatusChange
		expected.Images[testSha] = imageInfo
		//
		Expect(actual).To(Equal(&expected))
	})
}
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
	"testing"

	m "github.com/blackducksoftware/perceptor/pkg/core/model"
)

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
	expected.ImageHubCheckQueue = append(expected.ImageHubCheckQueue, imageInfo.ImageSha)
	//
	assertEqual(t, actual, expected)
}

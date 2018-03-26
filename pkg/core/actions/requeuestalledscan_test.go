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
	"time"

	m "github.com/blackducksoftware/perceptor/pkg/core/model"
)

func requeueTestModel() *m.Model {
	model := m.NewModel(&m.Config{ConcurrentScanLimit: 3}, "test version")
	model.AddImage(image1)
	model.SetImageScanStatus(image1.Sha, m.ScanStatusInQueue)
	model.SetImageScanStatus(image1.Sha, m.ScanStatusRunningScanClient)
	model.AddImage(image2)
	model.SetImageScanStatus(image2.Sha, m.ScanStatusRunningHubScan)
	return model
}

func TestRequeueStalledScanClientAndHubScans(t *testing.T) {
	model := requeueTestModel()

	if model.Images[image1.Sha].ScanStatus != m.ScanStatusRunningScanClient {
		t.Errorf("expected scan to be in hub, is actually %s", model.Images[image1.Sha].ScanStatus)
		return
	}

	r := RequeueStalledScans{StalledHubScanTimeout: 1 * time.Nanosecond, StalledScanClientTimeout: 1 * time.Nanosecond}
	r.Apply(model)

	for _, sha := range []m.DockerImageSha{image1.Sha, image2.Sha} {
		actual := model.Images[sha].ScanStatus
		if actual != m.ScanStatusInQueue {
			t.Errorf("expected scan to be in queue, is actually %s", actual)
		}
	}
}

func TestDoesntRequeueRunningHubScan(t *testing.T) {
	model := requeueTestModel()

	if model.Images[image1.Sha].ScanStatus != m.ScanStatusRunningScanClient {
		t.Errorf("expected scan to be in hub, is actually %s", model.Images[image1.Sha].ScanStatus)
		return
	}

	r := RequeueStalledScans{StalledHubScanTimeout: 1 * time.Minute, StalledScanClientTimeout: 1 * time.Minute}
	r.Apply(model)

	actual1 := model.Images[image1.Sha].ScanStatus
	if actual1 != m.ScanStatusRunningScanClient {
		t.Errorf("expected scan to be running scan client, is actually %s", actual1)
	}

	actual2 := model.Images[image1.Sha].ScanStatus
	if actual2 != m.ScanStatusRunningScanClient {
		t.Errorf("expected scan to be running in hub, is actually %s", actual2)
	}
}

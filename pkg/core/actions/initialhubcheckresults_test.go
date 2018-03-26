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
	"fmt"
	"testing"

	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	"github.com/blackducksoftware/perceptor/pkg/hub"
)

func initialCheckModel() *m.Model {
	model := m.NewModel(&m.Config{ConcurrentScanLimit: 3}, "abc")
	model.AddImage(image1)
	return model
}

func TestInitialHubCheckResultsError(t *testing.T) {
	model := initialCheckModel()
	ihc := InitialHubCheckResults{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: nil, Err: fmt.Errorf("")}}
	ihc.Apply(model)

	actual := model.Images[image1.Sha].ScanStatus
	expected := m.ScanStatusInHubCheckQueue
	if actual != expected {
		t.Errorf("expected %s, found %s", expected, actual)
	}
}

func TestInitialHubCheckResultsNotFound(t *testing.T) {
	model := initialCheckModel()
	ihc := InitialHubCheckResults{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: nil, Err: nil}}
	ihc.Apply(model)

	actual := model.Images[image1.Sha].ScanStatus
	expected := m.ScanStatusInQueue
	if actual != expected {
		t.Errorf("expected %s, found %s", expected, actual)
	}
}

func TestInitialHubCheckResultsInProgress(t *testing.T) {
	model := initialCheckModel()
	imageScan := &hub.ImageScan{ScanSummary: hub.ScanSummary{Status: hub.ScanSummaryStatusInProgress}}
	ihc := InitialHubCheckResults{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: imageScan, Err: nil}}
	ihc.Apply(model)

	actual := model.Images[image1.Sha].ScanStatus
	expected := m.ScanStatusRunningHubScan
	if actual != expected {
		t.Errorf("expected %s, found %s", expected, actual)
	}
}

func TestInitialHubCheckResultsFailed(t *testing.T) {
	model := initialCheckModel()
	imageScan := &hub.ImageScan{ScanSummary: hub.ScanSummary{Status: hub.ScanSummaryStatusFailure}}
	ihc := InitialHubCheckResults{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: imageScan, Err: nil}}
	ihc.Apply(model)

	actual := model.Images[image1.Sha].ScanStatus
	expected := m.ScanStatusInQueue
	if actual != expected {
		t.Errorf("expected %s, found %s", expected, actual)
	}
}

func TestInitialHubCheckResultsSuccess(t *testing.T) {
	model := initialCheckModel()
	imageScan := &hub.ImageScan{ScanSummary: hub.ScanSummary{Status: hub.ScanSummaryStatusSuccess}}
	ihc := InitialHubCheckResults{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: imageScan, Err: nil}}
	ihc.Apply(model)

	actual := model.Images[image1.Sha].ScanStatus
	expected := m.ScanStatusComplete
	if actual != expected {
		t.Errorf("expected %s, found %s", expected, actual)
	}
}

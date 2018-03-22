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

func hubCheckModel() *m.Model {
	model := m.NewModel(&m.Config{ConcurrentScanLimit: 2}, "abc")
	model.AddImage(image1)
	model.SetImageScanStatus(image1.Sha, m.ScanStatusRunningHubScan)
	return model
}

func TestHubCheckResultsError(t *testing.T) {
	model := hubCheckModel()
	hc := HubCheckResults{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: nil, Err: fmt.Errorf("")}}
	hc.Apply(model)

	actual := model.Images[image1.Sha].ScanStatus
	expected := m.ScanStatusRunningHubScan
	if actual != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}
}

func TestHubCheckResultsNotFound(t *testing.T) {
	model := hubCheckModel()
	hc := HubCheckResults{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: nil, Err: nil}}
	hc.Apply(model)

	actual := model.Images[image1.Sha].ScanStatus
	expected := m.ScanStatusRunningHubScan
	if actual != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}
}

func TestHubCheckResultsInProgress(t *testing.T) {
	model := hubCheckModel()
	hc := HubCheckResults{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: imageScan(0, hub.ScanSummaryStatusInProgress), Err: nil}}
	hc.Apply(model)

	actual := model.Images[image1.Sha].ScanStatus
	expected := m.ScanStatusRunningHubScan
	if actual != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}
}

func TestHubCheckResultsFailed(t *testing.T) {
	model := hubCheckModel()
	hc := HubCheckResults{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: imageScan(0, hub.ScanSummaryStatusFailure), Err: nil}}
	hc.Apply(model)

	actual := model.Images[image1.Sha].ScanStatus
	expected := m.ScanStatusInQueue
	if actual != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}
}

func TestHubCheckResultsSuccess(t *testing.T) {
	model := hubCheckModel()
	hc := HubCheckResults{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: imageScan(8, hub.ScanSummaryStatusSuccess), Err: nil}}
	hc.Apply(model)

	actual := model.Images[image1.Sha].ScanStatus
	expected := m.ScanStatusComplete
	if actual != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}
	assertEqual(t, model.Images[image1.Sha].ScanResults, imageScan(8, hub.ScanSummaryStatusSuccess))
}

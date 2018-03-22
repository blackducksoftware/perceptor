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

func imageScan(vulnerabilityCount int, status hub.ScanSummaryStatus) *hub.ImageScan {
	return &hub.ImageScan{
		ScanSummary: hub.ScanSummary{
			Status: status,
		},
		RiskProfile: hub.RiskProfile{
			Categories: map[hub.RiskProfileCategory]hub.RiskProfileStatusCounts{
				hub.RiskProfileCategoryVulnerability: hub.RiskProfileStatusCounts{
					StatusCounts: map[hub.RiskProfileStatus]int{
						hub.RiskProfileStatusHigh: vulnerabilityCount,
					},
				},
			},
		},
	}
}

func recheckModel(vulnCount int) *m.Model {
	model := m.NewModel(&m.Config{ConcurrentScanLimit: 3}, "abc")
	model.AddImage(image1)
	model.SetImageScanStatus(image1.Sha, m.ScanStatusComplete)
	model.Images[image1.Sha].ScanResults = imageScan(vulnCount, hub.ScanSummaryStatusSuccess)
	return model
}

func TestHubRecheckResultsError(t *testing.T) {
	vulnCount := 3
	model := recheckModel(vulnCount)
	hrr := HubRecheckResults{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: nil, Err: fmt.Errorf("")}}
	hrr.Apply(model)

	actual := model.Images[image1.Sha].ScanResults
	expected := imageScan(vulnCount, hub.ScanSummaryStatusSuccess)
	assertEqual(t, actual, expected)
}

func TestHubRecheckResultsNotFound(t *testing.T) {
	vulnCount := 3
	model := recheckModel(vulnCount)
	hrr := HubRecheckResults{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: nil, Err: nil}}
	hrr.Apply(model)

	actual := model.Images[image1.Sha].ScanResults
	expected := imageScan(vulnCount, hub.ScanSummaryStatusSuccess)
	assertEqual(t, actual, expected)
}

func TestHubRecheckResultsInProgress(t *testing.T) {
	vulnCount := 3
	model := recheckModel(vulnCount)
	hrr := HubRecheckResults{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: imageScan(8, hub.ScanSummaryStatusInProgress), Err: nil}}
	hrr.Apply(model)

	actual := model.Images[image1.Sha].ScanResults
	expected := imageScan(vulnCount, hub.ScanSummaryStatusSuccess)
	assertEqual(t, actual, expected)
}

func TestHubRecheckResultsFailed(t *testing.T) {
	vulnCount := 3
	model := recheckModel(vulnCount)
	hrr := HubRecheckResults{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: imageScan(17, hub.ScanSummaryStatusFailure), Err: nil}}
	hrr.Apply(model)

	actual := model.Images[image1.Sha].ScanResults
	expected := imageScan(vulnCount, hub.ScanSummaryStatusSuccess)
	assertEqual(t, actual, expected)
}

func TestHubRecheckResultsSuccess(t *testing.T) {
	vulnCount := 3
	model := recheckModel(vulnCount)
	hrr := HubRecheckResults{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: imageScan(18, hub.ScanSummaryStatusSuccess), Err: nil}}
	hrr.Apply(model)

	actual := model.Images[image1.Sha].ScanResults
	expected := imageScan(18, hub.ScanSummaryStatusSuccess)
	assertEqual(t, actual, expected)
}

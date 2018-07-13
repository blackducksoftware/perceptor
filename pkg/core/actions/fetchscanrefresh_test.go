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

	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	hub "github.com/blackducksoftware/perceptor/pkg/hub"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func scan(vulnerabilityCount int, status hub.ScanSummaryStatus) *hub.ScanResults {
	return &hub.ScanResults{
		ScanSummaries: []hub.ScanSummary{{
			Status: status,
		}},
		RiskProfile: hub.RiskProfile{
			Categories: map[hub.RiskProfileCategory]hub.RiskProfileStatusCounts{
				hub.RiskProfileCategoryVulnerability: {
					StatusCounts: map[hub.RiskProfileStatus]int{
						hub.RiskProfileStatusHigh: vulnerabilityCount,
					},
				},
			},
		},
	}
}

func recheckModel(vulnCount int) *m.Model {
	model := m.NewModel("abc", &m.Config{ConcurrentScanLimit: 3}, nil)
	model.AddImage(image1, 0)
	model.SetLayersForImage(image1.Sha, layers1)
	model.SetLayerScanStatus(layer1, m.ScanStatusComplete)
	err := model.AddLayerToRefreshQueue(layer1)
	if err != nil {
		panic(err)
	}
	model.Layers[layer1].SetScanResults(scan(vulnCount, hub.ScanSummaryStatusSuccess))
	return model
}

func RunFetchScanRefresh() {
	Describe("FetchScanRefresh", func() {
		It("handles errors", func() {
			vulnCount := 3
			model := recheckModel(vulnCount)
			hrr := FetchScanRefresh{Scan: &m.HubScan{Sha: layer1, Scan: nil, Err: fmt.Errorf("")}}
			hrr.Apply(model)

			actual := model.Layers[layer1].ScanResults
			expected := scan(vulnCount, hub.ScanSummaryStatusSuccess)
			Expect(actual).To(Equal(expected))
		})

		It("handles layer not found", func() {
			vulnCount := 3
			model := recheckModel(vulnCount)
			hrr := FetchScanRefresh{Scan: &m.HubScan{Sha: layer1, Scan: nil, Err: nil}}
			hrr.Apply(model)

			actual := model.Layers[layer1].ScanResults
			expected := scan(vulnCount, hub.ScanSummaryStatusSuccess)
			Expect(actual).To(Equal(expected))
		})

		It("handles layer scan in progress", func() {
			vulnCount := 3
			model := recheckModel(vulnCount)
			hrr := FetchScanRefresh{Scan: &m.HubScan{Sha: layer1, Scan: scan(8, hub.ScanSummaryStatusInProgress), Err: nil}}
			hrr.Apply(model)

			actual := model.Layers[layer1].ScanResults
			expected := scan(vulnCount, hub.ScanSummaryStatusSuccess)
			Expect(actual).To(Equal(expected))
		})

		It("handles layer scan failed", func() {
			vulnCount := 3
			model := recheckModel(vulnCount)
			hrr := FetchScanRefresh{Scan: &m.HubScan{Sha: layer1, Scan: scan(17, hub.ScanSummaryStatusFailure), Err: nil}}
			hrr.Apply(model)

			actual := model.Layers[layer1].ScanResults
			expected := scan(vulnCount, hub.ScanSummaryStatusSuccess)
			Expect(actual).To(Equal(expected))
		})

		It("handles layer scan success", func() {
			model := recheckModel(3)
			expected := scan(18, hub.ScanSummaryStatusSuccess)
			hrr := FetchScanRefresh{Scan: &m.HubScan{Sha: layer1, Scan: expected, Err: nil}}
			hrr.Apply(model)

			actual := model.Layers[layer1].ScanResults
			Expect(actual).To(Equal(expected))
		})
	})
}

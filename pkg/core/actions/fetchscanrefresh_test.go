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

// import (
// 	"fmt"
//
// 	m "github.com/blackducksoftware/perceptor/pkg/core/model"
// 	"github.com/blackducksoftware/perceptor/pkg/hub"
// 	. "github.com/onsi/ginkgo"
// 	. "github.com/onsi/gomega"
// )
//
// func imageScan(vulnerabilityCount int, status hub.ScanSummaryStatus) *hub.ScanResults {
// 	return &hub.ScanResults{
// 		ScanSummaries: []hub.ScanSummary{{
// 			Status: status,
// 		}},
// 		RiskProfile: hub.RiskProfile{
// 			Categories: map[hub.RiskProfileCategory]hub.RiskProfileStatusCounts{
// 				hub.RiskProfileCategoryVulnerability: {
// 					StatusCounts: map[hub.RiskProfileStatus]int{
// 						hub.RiskProfileStatusHigh: vulnerabilityCount,
// 					},
// 				},
// 			},
// 		},
// 	}
// }
//
// func recheckModel(vulnCount int) *m.Model {
// 	model := m.NewModel("abc", &m.Config{ConcurrentScanLimit: 3}, nil)
// 	model.AddImage(image1, 0)
// 	model.SetImageScanStatus(image1.Sha, m.ScanStatusComplete)
// 	model.Images[image1.Sha].SetScanResults(imageScan(vulnCount, hub.ScanSummaryStatusSuccess))
// 	return model
// }
//
// func RunFetchScanRefresh() {
// 	Describe("FetchScanRefresh", func() {
// 		It("error", func() {
// 			vulnCount := 3
// 			model := recheckModel(vulnCount)
// 			hrr := FetchScanRefresh{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: nil, Err: fmt.Errorf("")}}
// 			hrr.Apply(model)
//
// 			actual := model.Images[image1.Sha].ScanResults
// 			expected := imageScan(vulnCount, hub.ScanSummaryStatusSuccess)
// 			Expect(actual).To(Equal(expected))
// 		})
//
// 		It("not found", func() {
// 			vulnCount := 3
// 			model := recheckModel(vulnCount)
// 			hrr := FetchScanRefresh{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: nil, Err: nil}}
// 			hrr.Apply(model)
//
// 			actual := model.Images[image1.Sha].ScanResults
// 			expected := imageScan(vulnCount, hub.ScanSummaryStatusSuccess)
// 			Expect(actual).To(Equal(expected))
// 		})
//
// 		It("in progress", func() {
// 			vulnCount := 3
// 			model := recheckModel(vulnCount)
// 			hrr := FetchScanRefresh{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: imageScan(8, hub.ScanSummaryStatusInProgress), Err: nil}}
// 			hrr.Apply(model)
//
// 			actual := model.Images[image1.Sha].ScanResults
// 			expected := imageScan(vulnCount, hub.ScanSummaryStatusSuccess)
// 			Expect(actual).To(Equal(expected))
// 		})
//
// 		It("failed", func() {
// 			vulnCount := 3
// 			model := recheckModel(vulnCount)
// 			hrr := FetchScanRefresh{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: imageScan(17, hub.ScanSummaryStatusFailure), Err: nil}}
// 			hrr.Apply(model)
//
// 			actual := model.Images[image1.Sha].ScanResults
// 			expected := imageScan(vulnCount, hub.ScanSummaryStatusSuccess)
// 			Expect(actual).To(Equal(expected))
// 		})
//
// 		It("success", func() {
// 			vulnCount := 3
// 			model := recheckModel(vulnCount)
// 			hrr := FetchScanRefresh{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: imageScan(18, hub.ScanSummaryStatusSuccess), Err: nil}}
// 			hrr.Apply(model)
//
// 			actual := model.Images[image1.Sha].ScanResults
// 			expected := imageScan(18, hub.ScanSummaryStatusSuccess)
// 			Expect(actual).To(Equal(expected))
// 		})
// 	})
// }

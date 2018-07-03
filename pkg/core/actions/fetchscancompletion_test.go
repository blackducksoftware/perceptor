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
	"github.com/blackducksoftware/perceptor/pkg/hub"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func hubCheckModel() *m.Model {
	model := m.NewModel("abc", &m.Config{ConcurrentScanLimit: 2}, nil)
	model.AddImage(image1, 0)
	model.SetImageScanStatus(image1.Sha, m.ScanStatusRunningHubScan)
	return model
}

func RunFetchScanCompletionTests() {
	Describe("FetchScanCompletion", func() {
		It("error handling", func() {
			model := hubCheckModel()
			hc := FetchScanCompletion{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: nil, Err: fmt.Errorf("")}}
			hc.Apply(model)

			actual := model.Images[image1.Sha].ScanStatus
			expected := m.ScanStatusRunningHubScan
			Expect(actual).To(Equal(expected))
		})

		It("not found", func() {
			model := hubCheckModel()
			hc := FetchScanCompletion{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: nil, Err: nil}}
			hc.Apply(model)

			actual := model.Images[image1.Sha].ScanStatus
			expected := m.ScanStatusRunningHubScan
			Expect(actual).To(Equal(expected))
		})

		It("in progress", func() {
			model := hubCheckModel()
			hc := FetchScanCompletion{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: imageScan(0, hub.ScanSummaryStatusInProgress), Err: nil}}
			hc.Apply(model)

			actual := model.Images[image1.Sha].ScanStatus
			expected := m.ScanStatusRunningHubScan
			Expect(actual).To(Equal(expected))
		})

		It("failed", func() {
			model := hubCheckModel()
			hc := FetchScanCompletion{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: imageScan(0, hub.ScanSummaryStatusFailure), Err: nil}}
			hc.Apply(model)

			actual := model.Images[image1.Sha].ScanStatus
			expected := m.ScanStatusInQueue
			Expect(actual).To(Equal(expected))
		})

		It("success", func() {
			model := hubCheckModel()
			hc := FetchScanCompletion{Scan: &m.HubImageScan{Sha: image1.Sha, Scan: imageScan(8, hub.ScanSummaryStatusSuccess), Err: nil}}
			hc.Apply(model)

			actual := model.Images[image1.Sha].ScanStatus
			expected := m.ScanStatusComplete
			Expect(actual).To(Equal(expected))
			Expect(model.Images[image1.Sha].ScanResults).To(Equal(imageScan(8, hub.ScanSummaryStatusSuccess)))
		})
	})
}

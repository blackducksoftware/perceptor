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

func initialCheckModel() *m.Model {
	model := m.NewModel("abc", &m.Config{ConcurrentScanLimit: 3}, nil)
	model.AddImage(image1, 0)
	model.SetLayersForImage(image1.Sha, layers1)
	return model
}

func RunFetchScanInitialTests() {
	Describe("fetch scan initial", func() {
		It("error", func() {
			model := initialCheckModel()
			ihc := FetchScanInitial{Scan: &m.HubScan{Sha: layer1, Scan: nil, Err: fmt.Errorf("")}}
			ihc.Apply(model)

			Expect(model.Layers[layer1].ScanStatus).To(Equal(m.ScanStatusUnknown))
		})

		It("not found", func() {
			model := initialCheckModel()
			ihc := FetchScanInitial{Scan: &m.HubScan{Sha: layer1, Scan: nil, Err: nil}}
			ihc.Apply(model)

			Expect(model.Layers[layer1].ScanStatus).To(Equal(m.ScanStatusNotScanned))
		})

		It("in progress", func() {
			model := initialCheckModel()
			imageScan := &hub.ScanResults{ScanSummaries: []hub.ScanSummary{{Status: hub.ScanSummaryStatusInProgress}}}
			ihc := FetchScanInitial{Scan: &m.HubScan{Sha: layer1, Scan: imageScan, Err: nil}}
			ihc.Apply(model)

			Expect(model.Layers[layer1].ScanStatus).To(Equal(m.ScanStatusRunningHubScan))
		})

		It("failed", func() {
			model := initialCheckModel()
			imageScan := &hub.ScanResults{ScanSummaries: []hub.ScanSummary{{Status: hub.ScanSummaryStatusFailure}}}
			ihc := FetchScanInitial{Scan: &m.HubScan{Sha: layer1, Scan: imageScan, Err: nil}}
			ihc.Apply(model)

			Expect(model.Layers[layer1].ScanStatus).To(Equal(m.ScanStatusNotScanned))
		})

		It("success", func() {
			model := initialCheckModel()
			imageScan := &hub.ScanResults{ScanSummaries: []hub.ScanSummary{{Status: hub.ScanSummaryStatusSuccess}}}
			ihc := FetchScanInitial{Scan: &m.HubScan{Sha: layer1, Scan: imageScan, Err: nil}}
			ihc.Apply(model)

			Expect(model.Layers[layer1].ScanStatus).To(Equal(m.ScanStatusComplete))
		})
	})
}

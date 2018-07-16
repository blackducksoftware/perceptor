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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	m "github.com/blackducksoftware/perceptor/pkg/core/model"
)

func RunFinishScanClientTests() {
	Describe("finish scan client", func() {
		It("should handle failure", func() {
			model := m.NewModel("test version", &m.Config{ConcurrentScanLimit: 1}, nil)
			model.AddImage(image1, 0)
			err := model.SetLayersForImage(image1.Sha, layers1)
			Expect(err).To(BeNil())
			err = model.SetLayerScanStatus(layer1, m.ScanStatusNotScanned)
			Expect(err).To(BeNil())
			err = model.SetLayerScanStatus(layer1, m.ScanStatusRunningScanClient)
			Expect(err).To(BeNil())
			model.FinishRunningScanClient(layer1, fmt.Errorf("oops, unable to run scan client"))
			Expect(model.Layers[layer1].ScanStatus).To(Equal(m.ScanStatusNotScanned))
		})
		It("should handle success", func() {
			// TODO
		})
	})
}

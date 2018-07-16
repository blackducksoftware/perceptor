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
	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func RunShouldScanLayerTests() {
	Describe("should scan layer", func() {
		action := NewShouldScanLayer(layer1)

		It("should say yes if the layer has not been scanned", func() {
			actual := createNewModel1()
			actual.SetLayersForImage(image1.Sha, []string{layer1})
			actual.Layers[layer1].ScanStatus = m.ScanStatusNotScanned
			go func() {
				action.Apply(actual)
			}()
			var err error
			var s m.ShouldScanLayer
			select {
			case e := <-action.Err:
				err = e
			case shouldScan := <-action.Success:
				s = shouldScan
			}
			Expect(err).To(BeNil())
			Expect(s).To(Equal(m.ShouldScanLayerYes))
		})

		It("should say no if the layer has already been scanned", func() {
			actual := createNewModel1()
			actual.SetLayersForImage(image1.Sha, []string{layer1})
			actual.Layers[layer1].ScanStatus = m.ScanStatusComplete
			go func() {
				action.Apply(actual)
			}()
			var err error
			var s m.ShouldScanLayer
			select {
			case e := <-action.Err:
				err = e
			case shouldScan := <-action.Success:
				s = shouldScan
			}
			Expect(err).To(BeNil())
			Expect(s).To(Equal(m.ShouldScanLayerNo))
		})

		It("should say wait if the number of concurrent scans already equals the limit", func() {
			actual := createNewModel1()
			actual.Config.ConcurrentScanLimit = 0
			actual.SetLayersForImage(image1.Sha, []string{layer1})
			actual.Layers[layer1].ScanStatus = m.ScanStatusNotScanned
			go func() {
				action.Apply(actual)
			}()
			var err error
			var s m.ShouldScanLayer
			select {
			case e := <-action.Err:
				err = e
			case shouldScan := <-action.Success:
				s = shouldScan
			}
			Expect(err).To(BeNil())
			Expect(s).To(Equal(m.ShouldScanLayerWait))
		})

		It("should say 'wait' if the layer has not been checked in the hub", func() {
			actual := createNewModel1()
			actual.SetLayersForImage(image1.Sha, []string{layer1})
			actual.Layers[layer1].ScanStatus = m.ScanStatusUnknown
			go func() {
				action.Apply(actual)
			}()
			var err error
			var s m.ShouldScanLayer
			select {
			case e := <-action.Err:
				err = e
			case shouldScan := <-action.Success:
				s = shouldScan
			}
			Expect(err).To(BeNil())
			Expect(s).To(Equal(m.ShouldScanLayerWait))
		})

		It("should report an error if the layer is not present", func() {
			actual := createNewModel1()
			a := NewShouldScanLayer(layer2)
			go func() {
				a.Apply(actual)
			}()
			var err error
			// var s m.ShouldScanLayer
			select {
			case e := <-a.Err:
				err = e
			case _ = <-a.Success:
				// s = shouldScan
			}
			Expect(err).ToNot(BeNil())
			// Expect(s).To(Equal(""))
		})
	})
}

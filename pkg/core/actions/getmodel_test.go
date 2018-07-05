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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func RunTestGetFullScanResults() {
	Describe("test get full scan results", func() {
		model := createNewModel1()
		scanResults := ScanResults(model)
		It("should produce the right number of pods", func() {
			Expect(len(scanResults.Pods)).To(Equal(1))
		})
		It("should produce pods with the right data", func() {
			Expect(scanResults.Pods[0].Name).To(Equal("pod2"))
		})
		It("should produce the right number of images", func() {
			Expect(len(scanResults.Images)).To(Equal(1))
		})
		It("should produce the right number of policy violations", func() {
			Expect(scanResults.Images[0].PolicyViolations).To(Equal(3))
		})
	})
}

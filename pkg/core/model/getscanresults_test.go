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

package model

import (
	"github.com/blackducksoftware/perceptor/pkg/hub"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func RunTestPodOverallStatus() {
	Describe("test pod overall status", func() {
		model := createNewModel2()
		It("should get nil scan results for pod 1", func() {
			scan1, err := scanResultsForPod(model, pod1.QualifiedName())
			Expect(err).To(BeNil())
			Expect(scan1).To(BeNil())
		})

		It("should get the right scan results for pod 2", func() {
			scan2, err := scanResultsForPod(model, pod2.QualifiedName())
			Expect(err).To(BeNil())
			Expect(scan2.PolicyViolations).To(Equal(3))
			Expect(scan2.Vulnerabilities).To(Equal(0))
			Expect(scan2.OverallStatus).To(Equal(hub.PolicyStatusTypeInViolation))
		})

		It("should get the right results for pod 3", func() {
			scan3, err := scanResultsForPod(model, pod3.QualifiedName())
			Expect(err).To(BeNil())
			Expect(scan3.PolicyViolations).To(Equal(0))
			Expect(scan3.Vulnerabilities).To(Equal(0))
			Expect(scan3.OverallStatus).To(Equal(hub.PolicyStatusTypeNotInViolation))
		})

		It("should get the right results for pod 4", func() {
			scan4, err := scanResultsForPod(model, pod4.QualifiedName())
			Expect(err).To(BeNil())
			Expect(scan4.PolicyViolations).To(Equal(0))
			Expect(scan4.Vulnerabilities).To(Equal(0))
			Expect(scan4.OverallStatus).To(Equal(hub.PolicyStatusTypeNotInViolation))
		})

		It("should get the right results for image 1", func() {
			imageScan1, err := scanResultsForImage(model, image1.Sha)
			Expect(err).To(BeNil())
			Expect(imageScan1.PolicyViolations).To(Equal(3))
			Expect(imageScan1.Vulnerabilities).To(Equal(0))
			Expect(imageScan1.OverallStatus).To(Equal(hub.PolicyStatusTypeInViolation))
		})

		It("should get nil scan results for image 2", func() {
			imageScan2, err := scanResultsForImage(model, image2.Sha)
			Expect(err).To(BeNil())
			Expect(imageScan2).To(BeNil())
		})

		It("should get the right results for image 3", func() {
			imageScan3, err := scanResultsForImage(model, image3.Sha)
			Expect(err).To(BeNil())
			Expect(imageScan3.PolicyViolations).To(Equal(0))
			Expect(imageScan3.Vulnerabilities).To(Equal(0))
			Expect(imageScan3.OverallStatus).To(Equal(hub.PolicyStatusTypeNotInViolation))
		})
	})
}

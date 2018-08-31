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

package core

import (
	"time"

	"github.com/blackducksoftware/perceptor/pkg/api"
	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func RunTestPerceptor() {
	Describe("Perceptor", func() {
		It("should experience unblocked channel communication", func() {
			manager := &MockHubCreater{}
			timings := &Timings{
				CheckForStalledScansPause: 9999 * time.Second,
				ModelMetricsPause:         15 * time.Second,
				StalledScanClientTimeout:  9999 * time.Second,
				UnknownImagePause:         2 * time.Second,
			}
			pcp, err := NewPerceptor(timings, &ScanScheduler{HubManager: manager, ConcurrentScanLimit: 2, TotalScanLimit: 5}, manager)
			Expect(err).To(BeNil())
			image1 := api.Image{
				Sha:        "sha1sha8sh12sh16sha1sha8sh12sh16sha1sha8sh12sh16sha1sha8sh12sh16",
				Repository: "repo1",
				Tag:        "tag1"}
			imageSpec := api.ImageSpec{
				HubProjectName:        "proj1",
				HubProjectVersionName: "ver1",
				HubScanName:           "scan1",
				HubURL:                "hub1",
				Repository:            image1.Repository,
				Sha:                   image1.Sha,
				Tag:                   image1.Tag}
			nextImage := api.NextImage{
				ImageSpec: &imageSpec,
			}
			Expect(pcp.AddImage(image1)).To(BeNil())
			pcp.PutHubs(&api.PutHubs{HubURLs: []string{"hub1"}})
			time.Sleep(3 * time.Second)
			Expect(pcp.GetNextImage()).To(Equal(nextImage))
			Expect(pcp.PostFinishScan(api.FinishedScanClientJob{ImageSpec: imageSpec, Err: ""})).To(BeNil())
			Expect(pcp.model.Images["sha1"].ScanStatus).To(Equal(m.ScanStatusRunningHubScan))
			//			Expect(pcp.PostFinishScan(api.FinishedScanClientJob{})).To(BeNil())
		})
	})
}

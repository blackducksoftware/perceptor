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
	"fmt"
	"time"

	"github.com/blackducksoftware/perceptor/pkg/api"
	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func newPerceptor(concurrentScanLimit int, totalScanLimit int) *Perceptor {
	stop := make(chan struct{})
	manager := NewHubManager(createMockHub, stop)
	timings := &Timings{
		CheckForStalledScansPauseHours: 9999,
		ModelMetricsPauseSeconds:       15,
		StalledScanClientTimeoutHours:  9999,
		UnknownImagePauseMilliseconds:  500,
	}
	config := &Config{}
	pcp, err := NewPerceptor(config, timings,
		&ScanScheduler{
			HubManager:          manager,
			ConcurrentScanLimit: concurrentScanLimit,
			TotalScanLimit:      totalScanLimit},
		manager)
	Expect(err).To(BeNil())
	return pcp
}

func makeImageSpec(image *api.Image, hub string) *api.ImageSpec {
	return &api.ImageSpec{
		HubProjectName:        image.Repository,
		HubProjectVersionName: fmt.Sprintf("%s-%s", image.Tag, image.Sha[:20]),
		HubScanName:           image.Sha,
		HubURL:                hub,
		Repository:            image.Repository,
		Sha:                   image.Sha,
		Tag:                   image.Tag,
		Priority:              *image.Priority,
	}
}

func RunTestPerceptor() {
	one := 1
	image1 := api.Image{Sha: "sha1abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzeightchs", Repository: "repo1", Tag: "tag1", Priority: &one}
	two := 2
	image2 := api.Image{Sha: "sha2222222222222222222222222222222222222222222222222222222222222", Repository: "repo2", Tag: "tag2", Priority: &two}
	three := 3
	image3 := api.Image{Sha: "sha1333333333333333333333333333333333333333333333333333333333333", Repository: "repo3", Tag: "tag3", Priority: &three}
	four := 4
	image4 := api.Image{Sha: "sha1444444444444444444444444444444444444444444444444444444444444", Repository: "repo4", Tag: "tag4", Priority: &four}
	five := 5
	image5 := api.Image{Sha: "sha1555555555555555555555555555555555555555555555555555555555555", Repository: "repo5", Tag: "tag5", Priority: &five}
	Describe("Perceptor", func() {
		It("should experience unblocked channel communication", func() {
			pcp := newPerceptor(2, 5)
			sha1, err := m.NewDockerImageSha(image1.Sha)
			Expect(err).To(BeNil())
			imageSpec := makeImageSpec(&image1, "hub1")
			nextImage := api.NextImage{ImageSpec: imageSpec}
			Expect(pcp.AddImage(image1)).To(BeNil())
			time.Sleep(500 * time.Millisecond)
			Expect(len(pcp.model.Images)).To(Equal(1))

			pcp.PutHubs(&api.PutHubs{HubURLs: []string{"hub1"}})
			Expect(pcp.model.Images[sha1].ScanStatus).To(Equal(m.ScanStatusUnknown))
			time.Sleep(1 * time.Second)

			Expect(pcp.model.Images[sha1].ScanStatus).To(Equal(m.ScanStatusInQueue))
			Expect(pcp.GetNextImage()).To(Equal(nextImage))
			Expect(pcp.PostFinishScan(api.FinishedScanClientJob{ImageSpec: *imageSpec, Err: ""})).To(BeNil())
			time.Sleep(500 * time.Millisecond)

			Expect(pcp.model.Images[sha1].ScanStatus).To(Equal(m.ScanStatusRunningHubScan))
			//			Expect(pcp.PostFinishScan(api.FinishedScanClientJob{})).To(BeNil())
		})

		It("should not assign scans when there are no hubs", func() {
			pcp := newPerceptor(2, 5)
			pcp.UpdateAllImages(api.AllImages{
				Images: []api.Image{image1},
			})
			Expect(pcp.GetNextImage()).To(Equal(api.NextImage{}))
		})

		It("should not assign scans when the concurrent scan limit is 0", func() {
			pcp := newPerceptor(0, 5)
			pcp.UpdateAllImages(api.AllImages{
				Images: []api.Image{image1},
			})
			pcp.PutHubs(&api.PutHubs{HubURLs: []string{"hub1", "hub2", "hub3"}})
			time.Sleep(1 * time.Second)
			Expect(pcp.GetNextImage()).To(Equal(api.NextImage{}))
		})

		It("should assign scans to different hubs, not exceeding the concurrent scan limit of any hub", func() {
			pcp := newPerceptor(1, 5)
			pcp.UpdateAllImages(api.AllImages{
				Images: []api.Image{image1, image2, image3, image4, image5},
			})
			pcp.PutHubs(&api.PutHubs{HubURLs: []string{"hub1", "hub2", "hub3"}})
			time.Sleep(1 * time.Second)

			Expect(pcp.model.ImageScanQueue.Size()).To(Equal(5))

			next1 := pcp.GetNextImage()
			Expect(next1).To(Equal(*api.NewNextImage(makeImageSpec(&image5, next1.ImageSpec.HubURL))))
			time.Sleep(500 * time.Millisecond)
			Expect(pcp.model.ImageScanQueue.Size()).To(Equal(4))

			next2 := pcp.GetNextImage()
			Expect(next2).To(Equal(*api.NewNextImage(makeImageSpec(&image4, next2.ImageSpec.HubURL))))
			time.Sleep(500 * time.Millisecond)
			Expect(pcp.model.ImageScanQueue.Size()).To(Equal(3))

			next3 := pcp.GetNextImage()
			Expect(next3).To(Equal(*api.NewNextImage(makeImageSpec(&image3, next3.ImageSpec.HubURL))))
			time.Sleep(500 * time.Millisecond)
			Expect(pcp.model.ImageScanQueue.Size()).To(Equal(2))

			Expect(pcp.GetNextImage()).To(Equal(api.NextImage{}))
			Expect(pcp.model.ImageScanQueue.Size()).To(Equal(2))
		})
	})
}

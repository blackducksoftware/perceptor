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
	"sync"
	"time"

	"github.com/blackducksoftware/perceptor/pkg/api"
	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	"github.com/blackducksoftware/perceptor/pkg/hub"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	one      = 1
	image1   = api.Image{Sha: "sha1abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzeightchs", Repository: "repo1", Tag: "tag1", Priority: &one}
	two      = 2
	image2   = api.Image{Sha: "sha2222222222222222222222222222222222222222222222222222222222222", Repository: "repo2", Tag: "tag2", Priority: &two}
	three    = 3
	image3   = api.Image{Sha: "sha1333333333333333333333333333333333333333333333333333333333333", Repository: "repo3", Tag: "tag3", Priority: &three}
	four     = 4
	image4   = api.Image{Sha: "sha1444444444444444444444444444444444444444444444444444444444444", Repository: "repo4", Tag: "tag4", Priority: &four}
	five     = 5
	image5   = api.Image{Sha: "sha1555555555555555555555555555555555555555555555555555555555555", Repository: "repo5", Tag: "tag5", Priority: &five}
	hub1Host = "hub1"
	hub2Host = "hub2"
	hub3Host = "hub3"
)

func newPerceptor(concurrentScanLimit int, totalScanLimit int) *Perceptor {
	stop := make(chan struct{})
	manager := NewHubManager(createMockHubClient, stop)
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

func newPerceptorPrepopulatedClients(fetchUnknownScansPause time.Duration) *Perceptor {
	concurrentScanLimit := 2
	totalScanLimit := 5
	scans := map[string][]string{
		hub1Host: {image1.Sha, image2.Sha},
		hub2Host: {image3.Sha},
		hub3Host: {},
	}
	createClient := func(hubURL string) (hub.ClientInterface, error) {
		mockRawClient := hub.NewMockRawClient(false, scans[hubURL])
		return hub.NewClient("mock-username", "mock-password", hubURL, mockRawClient, 1*time.Minute, fetchUnknownScansPause, 999999*time.Hour), nil
	}

	stop := make(chan struct{})
	manager := NewHubManager(createClient, stop)
	timings := &Timings{
		CheckForStalledScansPauseHours: 9999,
		ModelMetricsPauseSeconds:       15,
		StalledScanClientTimeoutHours:  9999,
		UnknownImagePauseMilliseconds:  500,
	}
	config := &Config{
		Timings: &Timings{},
		Hub:     &HubConfig{},
	}
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
			Expect(pcp.model.Images[sha1].ScanStatus).To(Equal(m.ScanStatusUnknown))

			pcp.hubManager.SetHubs([]string{"hub1"})
			time.Sleep(1 * time.Second)

			Expect(pcp.model.Images[sha1].ScanStatus).To(Equal(m.ScanStatusInQueue))
			Expect(pcp.GetNextImage()).To(Equal(nextImage))
			Expect(pcp.PostFinishScan(api.FinishedScanClientJob{ImageSpec: *imageSpec, Err: ""})).To(BeNil())
			time.Sleep(500 * time.Millisecond)

			Expect(pcp.model.Images[sha1].ScanStatus).To(Equal(m.ScanStatusRunningHubScan))
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
			pcp.hubManager.SetHubs([]string{"hub1", "hub2", "hub3"})
			time.Sleep(1 * time.Second)
			Expect(pcp.GetNextImage()).To(Equal(api.NextImage{}))
		})

		It("should assign scans to different hubs, not exceeding the concurrent scan limit of any hub", func() {
			pcp := newPerceptor(1, 5)
			pcp.UpdateAllImages(api.AllImages{
				Images: []api.Image{image1, image2, image3, image4, image5},
			})
			pcp.hubManager.SetHubs([]string{"hub1", "hub2", "hub3"})
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

		It("should handle scan client failure", func() {
			pcp := newPerceptor(2, 5)
			pcp.UpdateAllImages(api.AllImages{
				Images: []api.Image{image1, image2},
			})
			pcp.hubManager.SetHubs([]string{"hub1"})
			time.Sleep(1 * time.Second)

			Expect(pcp.model.ImageScanQueue.Size()).To(Equal(2))

			next1 := pcp.GetNextImage()
			Expect(next1.ImageSpec.Sha).To(Equal(image2.Sha))
			time.Sleep(500 * time.Millisecond)
			Expect(pcp.model.ImageScanQueue.Size()).To(Equal(1))

			pcp.PostFinishScan(api.FinishedScanClientJob{Err: "planned error", ImageSpec: *next1.ImageSpec})
			time.Sleep(500 * time.Millisecond)

			Expect(pcp.model.ImageScanQueue.Size()).To(Equal(2))
			Expect(pcp.model.Images[m.DockerImageSha(image1.Sha)].ScanStatus).To(Equal(m.ScanStatusInQueue))

			Expect(<-pcp.hubManager.HubClients()["hub1"].ScansCount()).To(Equal(0))
		})

		It("should recognize scan status of scans already in hubs when first starting up, or after a restart", func() {
			pcp := newPerceptorPrepopulatedClients(500 * time.Millisecond)
			pcp.UpdateAllImages(api.AllImages{
				Images: []api.Image{image1, image2, image3, image4, image5},
			})
			pcp.hubManager.SetHubs([]string{hub1Host, hub2Host, hub3Host})
			time.Sleep(1 * time.Second)

			// jbs, _ := json.MarshalIndent(pcp.GetModel(), "", "  ")
			// fmt.Printf("%s\n", string(jbs))

			Expect(pcp.model.ImageScanQueue.Size()).To(Equal(2))
			Expect(pcp.model.Images[m.DockerImageSha(image1.Sha)].ScanStatus).To(Equal(m.ScanStatusComplete))
			Expect(pcp.model.Images[m.DockerImageSha(image2.Sha)].ScanStatus).To(Equal(m.ScanStatusComplete))
			Expect(pcp.model.Images[m.DockerImageSha(image3.Sha)].ScanStatus).To(Equal(m.ScanStatusComplete))
			Expect(pcp.model.Images[m.DockerImageSha(image4.Sha)].ScanStatus).To(Equal(m.ScanStatusInQueue))
			Expect(pcp.model.Images[m.DockerImageSha(image5.Sha)].ScanStatus).To(Equal(m.ScanStatusInQueue))
		})

		It("should not return the same image to scan twice", func() {
			pcp := newPerceptor(2, 5)
			pcp.UpdateAllImages(api.AllImages{
				Images: []api.Image{image1, image2, image3, image4, image5},
			})
			pcp.hubManager.SetHubs([]string{"hub1"})
			time.Sleep(1 * time.Second)

			var i1 *api.NextImage
			var i2 *api.NextImage
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				i := pcp.GetNextImage()
				i1 = &i
				wg.Done()
			}()
			go func() {
				i := pcp.GetNextImage()
				i2 = &i
				wg.Done()
			}()
			wg.Wait()
			Expect(i1).NotTo(Equal(i2))
		})
	})
}

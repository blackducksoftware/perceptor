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
	"encoding/json"
	"sort"

	ds "github.com/blackducksoftware/perceptor/pkg/datastructures"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var (
	sha1   = DockerImageSha("sha1")
	image1 = *NewImage("image1", sha1)
	sha2   = DockerImageSha("sha2")
	image2 = *NewImage("image2", sha2)
	sha3   = DockerImageSha("sha3")
	image3 = *NewImage("image3", sha3)
	cont1  = *NewContainer(image1, "cont1")
	cont2  = *NewContainer(image2, "cont2")
	cont3  = *NewContainer(image3, "cont3")
	pod1   = *NewPod("pod1", "pod1uid", "ns1", []Container{cont1, cont2})
	pod2   = *NewPod("pod2", "pod2uid", "ns1", []Container{cont1})
	pod3   = *NewPod("pod3", "pod3uid", "ns3", []Container{cont3})
	// this is ridiculous, but let's create a pod with 0 containers
	pod4 = *NewPod("pod4", "pod4uid", "ns4", []Container{})
)

func sortedValues(pq *ds.PriorityQueue) []interface{} {
	vals := pq.Values()
	sort.Slice(vals, func(i int, j int) bool {
		return vals[i].(DockerImageSha) < vals[j].(DockerImageSha)
	})
	return vals
}

func RunModelTests() {
	Describe("Model", func() {

		removeItemModel := func() *Model {
			model := NewModel("zzz", &Config{ConcurrentScanLimit: 1}, nil)
			model.AddImage(image1, 0)
			model.AddImage(image2, 0)
			model.AddImage(image3, 0)
			return model
		}

		removeScanItemModel := func() *Model {
			model := NewModel("zzz", &Config{ConcurrentScanLimit: 1}, nil)
			model.AddImage(image1, 0)
			model.AddImage(image2, 0)
			model.AddImage(image3, 0)
			model.SetImageScanStatus(image1.Sha, ScanStatusInQueue)
			model.SetImageScanStatus(image2.Sha, ScanStatusInQueue)
			model.SetImageScanStatus(image3.Sha, ScanStatusInQueue)
			return model
		}

		It("Model JSON Serialization", func() {
			m := NewModel("test version", &Config{ConcurrentScanLimit: 3}, nil)
			jsonBytes, err := json.Marshal(m)
			Expect(err).To(BeNil())
			log.Infof("json bytes: %s", string(jsonBytes))
		})

		Describe("Hub check queue operations", func() {
			It("TestModelRemoveItemFromFrontOfHubCheckQueue", func() {
				model := removeItemModel()
				model.removeImageFromHubCheckQueue(image1.Sha)
				// "remove item from front of hub check queue"
				Expect(model.ImageHubCheckQueue).To(Equal([]DockerImageSha{image2.Sha, image3.Sha}))
			})

			It("TestModelRemoveItemFromMiddleOfHubCheckQueue", func() {
				model := removeItemModel()
				err := model.removeImageFromHubCheckQueue(image2.Sha)
				Expect(err).To(BeNil())
				Expect(model.ImageHubCheckQueue).To(Equal([]DockerImageSha{image1.Sha, image3.Sha}))
			})

			It("TestModelRemoveItemFromEndOfHubCheckQueue", func() {
				model := removeItemModel()
				model.removeImageFromHubCheckQueue(image3.Sha)
				Expect(model.ImageHubCheckQueue).To(Equal([]DockerImageSha{image1.Sha, image2.Sha}))
			})

			It("TestModelRemoveAllItemsFromHubCheckQueue", func() {
				model := removeItemModel()
				model.removeImageFromHubCheckQueue(image1.Sha)
				model.removeImageFromHubCheckQueue(image2.Sha)
				model.removeImageFromHubCheckQueue(image3.Sha)
				Expect(model.ImageHubCheckQueue).To(Equal([]DockerImageSha{}))
			})
		})

		Describe("Image scan queue operations", func() {
			It("TestModelRemoveItemFromFrontOfScanQueue", func() {
				model := removeScanItemModel()
				model.SetImageScanStatus(image1.Sha, ScanStatusRunningScanClient)
				Expect(sortedValues(model.ImageScanQueue)).To(Equal([]interface{}{image2.Sha, image3.Sha}))
			})

			It("TestModelRemoveItemFromMiddleOfScanQueue", func() {
				model := removeScanItemModel()
				model.SetImageScanStatus(image2.Sha, ScanStatusRunningScanClient)
				Expect(sortedValues(model.ImageScanQueue)).To(Equal([]interface{}{image1.Sha, image3.Sha}))
			})

			It("TestModelRemoveItemFromEndOfScanQueue", func() {
				model := removeScanItemModel()
				model.SetImageScanStatus(image3.Sha, ScanStatusRunningScanClient)
				Expect(sortedValues(model.ImageScanQueue)).To(Equal([]interface{}{image1.Sha, image2.Sha}))
			})

			It("TestModelRemoveAllItemsFromScanQueue", func() {
				model := removeScanItemModel()
				model.SetImageScanStatus(image1.Sha, ScanStatusRunningScanClient)
				model.SetImageScanStatus(image2.Sha, ScanStatusRunningScanClient)
				model.SetImageScanStatus(image3.Sha, ScanStatusRunningScanClient)
				Expect(sortedValues(model.ImageScanQueue)).To(Equal([]interface{}{}))
			})
		})

		Describe("Refresh queue operations", func() {
			model := removeItemModel()
			It("should add all 3 images to the refresh queue", func() {
				for _, image := range []Image{image1, image2, image3} {
					model.SetImageScanStatus(image.Sha, ScanStatusComplete)
					err := model.AddImageToRefreshQueue(image.Sha)
					Expect(err).To(BeNil())
				}
			})

			It("should start out with all 3 images", func() {
				Expect(model.ImageRefreshQueue).To(Equal([]DockerImageSha{image1.Sha, image2.Sha, image3.Sha}))
			})

			It("should produce image1 next, but still leave all 3 in the queue", func() {
				Expect(*model.GetNextImageFromRefreshQueue()).To(Equal(image1))
				Expect(model.ImageRefreshQueue).To(Equal([]DockerImageSha{image1.Sha, image2.Sha, image3.Sha}))
			})

			It("should remove 2 from the queue, leaving behind 1 and 3", func() {
				err := model.RemoveImageFromRefreshQueue(image2.Sha)
				Expect(err).To(BeNil())
				Expect(model.ImageRefreshQueue).To(Equal([]DockerImageSha{image1.Sha, image3.Sha}))
				Expect(*model.GetNextImageFromRefreshQueue()).To(Equal(image1))
			})

			It("should remove 1 from the queue, leaving behind 3", func() {
				err := model.RemoveImageFromRefreshQueue(image1.Sha)
				Expect(err).To(BeNil())
				Expect(model.ImageRefreshQueue).To(Equal([]DockerImageSha{image3.Sha}))
				Expect(*model.GetNextImageFromRefreshQueue()).To(Equal(image3))
			})

			It("should remove 3 from the queue, leaving behind nothing", func() {
				err := model.RemoveImageFromRefreshQueue(image3.Sha)
				Expect(err).To(BeNil())
				Expect(model.ImageRefreshQueue).To(Equal([]DockerImageSha{}))
				Expect(model.GetNextImageFromRefreshQueue()).To(BeNil())
			})
		})
	})
}

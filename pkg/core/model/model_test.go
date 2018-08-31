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

	"github.com/blackducksoftware/perceptor/pkg/hub"
	"github.com/blackducksoftware/perceptor/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

func sortedValues(pq *util.PriorityQueue) []interface{} {
	vals := pq.Values()
	sort.Slice(vals, func(i int, j int) bool {
		return vals[i].(DockerImageSha) < vals[j].(DockerImageSha)
	})
	return vals
}

func RunModelTests() {
	Describe("Model", func() {
		It("add image without pod, then same image in pod", func() {
			model := NewModel()

			model.addImage(image1, -2)
			model.addImage(image3, -1)
			model.setImageScanStatus(sha1, ScanStatusInQueue)
			model.setImageScanStatus(sha3, ScanStatusInQueue)
			Expect(model.ImagePriority[sha1]).To(Equal(-2))
			Expect(model.ImagePriority[sha3]).To(Equal(-1))
			Expect(model.ImageScanQueue.Values()[0]).To(Equal(sha3))

			model.addPod(pod1)
			log.Infof("priorities: %+v", model.ImagePriority)
			model.setImageScanStatus(sha2, ScanStatusInQueue)
			Expect(model.ImagePriority[sha1]).To(Equal(1))
			Expect(model.ImagePriority[sha2]).To(Equal(1))
			Expect(model.ImagePriority[sha3]).To(Equal(-1))

			// This is destructive!
			values := []interface{}{}
			for {
				next, err := model.ImageScanQueue.Pop()
				if err != nil {
					break
				}
				values = append(values, next)
			}
			Expect(values).To(Equal([]interface{}{sha1, sha2, sha3}))
		})

		removeScanItemModel := func() *Model {
			model := NewModel()
			model.addImage(image1, 0)
			model.addImage(image2, 0)
			model.addImage(image3, 0)
			model.setImageScanStatus(image1.Sha, ScanStatusInQueue)
			model.setImageScanStatus(image2.Sha, ScanStatusInQueue)
			model.setImageScanStatus(image3.Sha, ScanStatusInQueue)
			return model
		}

		It("Model JSON Serialization", func() {
			m := NewModel()
			jsonBytes, err := json.Marshal(m)
			Expect(err).To(BeNil())
			log.Infof("json bytes: %s", string(jsonBytes))
		})

		Describe("Image scan queue operations", func() {
			It("TestModelRemoveItemFromFrontOfScanQueue", func() {
				model := removeScanItemModel()
				model.setImageScanStatus(image1.Sha, ScanStatusRunningScanClient)
				Expect(sortedValues(model.ImageScanQueue)).To(Equal([]interface{}{image2.Sha, image3.Sha}))
			})

			It("TestModelRemoveItemFromMiddleOfScanQueue", func() {
				model := removeScanItemModel()
				model.setImageScanStatus(image2.Sha, ScanStatusRunningScanClient)
				Expect(sortedValues(model.ImageScanQueue)).To(Equal([]interface{}{image1.Sha, image3.Sha}))
			})

			It("TestModelRemoveItemFromEndOfScanQueue", func() {
				model := removeScanItemModel()
				model.setImageScanStatus(image3.Sha, ScanStatusRunningScanClient)
				Expect(sortedValues(model.ImageScanQueue)).To(Equal([]interface{}{image1.Sha, image2.Sha}))
			})

			It("TestModelRemoveAllItemsFromScanQueue", func() {
				model := removeScanItemModel()
				model.setImageScanStatus(image1.Sha, ScanStatusRunningScanClient)
				model.setImageScanStatus(image2.Sha, ScanStatusRunningScanClient)
				model.setImageScanStatus(image3.Sha, ScanStatusRunningScanClient)
				Expect(sortedValues(model.ImageScanQueue)).To(Equal([]interface{}{}))
			})
		})

		Describe("Image status operations", func() {
			It("moves an image Unknown->InQueue->RunningScanClient->RunningHubScan->Complete", func() {
				model := NewModel()
				model.addImage(image1, 0)
				model.addImage(image2, 0)
				// 1. Unknown
				Expect(model.Images[sha1].ScanStatus).To(Equal(ScanStatusUnknown))
				// 2. InQueue
				Expect(model.scanDidFinish(sha1, nil)).To(BeNil())
				Expect(model.Images[sha1].ScanStatus).To(Equal(ScanStatusInQueue))
				// 3. RunningScanClient
				Expect(model.StartScanClient(sha1)).To(BeNil())
				Expect(model.Images[sha1].ScanStatus).To(Equal(ScanStatusRunningScanClient))
				// 4. RunningHubScan
				model.finishRunningScanClient(&image1, nil)
				Expect(model.Images[sha1].ScanStatus).To(Equal(ScanStatusRunningHubScan))
				// 5. Complete
				results := &hub.ScanResults{
					ScanSummaries: []hub.ScanSummary{
						{
							CreatedAt: "maintenant",
							Status:    hub.ScanSummaryStatusSuccess,
							UpdatedAt: "demain",
						},
					},
				}
				Expect(model.scanDidFinish(sha1, results)).To(BeNil())
				Expect(model.Images[sha1].ScanStatus).To(Equal(ScanStatusComplete))
			})
		})
	})
}

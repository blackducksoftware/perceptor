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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

func RunGetNextImage() {
	Describe("GetNextImage", func() {
		It("no image available", func() {
			// actual
			actual := NewModel()
			get := NewGetNextImage()
			go func() {
				get.Apply(actual)
			}()
			nextImage := <-get.Done
			// expected: front image removed from scan queue, status and time of image changed
			expected := *NewModel()

			Expect(nextImage).To(BeNil())
			log.Infof("%+v, %+v", actual, expected)
			// assertEqual(t, actual, expected)
		})

		It("regular", func() {
			model := NewModel()
			model.addImage(image1, 0)
			model.setImageScanStatus(image1.Sha, ScanStatusInQueue)

			get := NewGetNextImage()
			go func() { get.Apply(model) }()
			nextImage := <-get.Done

			Expect(nextImage).To(Equal(image1))
			Expect(model.ImageScanQueue.Values()).To(Equal([]interface{}{}))
			Expect(model.Images[image1.Sha].ScanStatus).To(Equal(ScanStatusRunningScanClient))
			// TODO expected: time of image changed
		})
	})
}

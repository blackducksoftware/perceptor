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
	log "github.com/sirupsen/logrus"
)

func RunGetNextImageTests() {
	Describe("GetNextImage", func() {
		It("no image available", func() {
			// actual
			actual := m.NewModel(&m.Config{ConcurrentScanLimit: 3}, nil)
			get := NewGetNextImage()
			go func() {
				get.Apply(actual)
			}()
			nextImage := <-get.Done
			// expected: front image removed from scan queue, status and time of image changed
			expected := *m.NewModel(&m.Config{ConcurrentScanLimit: 3}, nil)

			Expect(nextImage).To(BeNil())
			log.Infof("%+v, %+v", actual, expected)
			// assertEqual(t, actual, expected)
		})

		It("regular", func() {
			model := m.NewModel(&m.Config{ConcurrentScanLimit: 3}, nil)
			model.AddImage(image1, 0)
			Expect(model.SetHubs([]string{"abc"})).To(BeNil())
			model.SetImageScanStatus(image1.Sha, m.ScanStatusInQueue)
			assignment := &m.HubImageAssignment{HubURL: "abc", Image: &image1}
			get := NewGetNextImage()
			go func() { get.Apply(model) }()
			var nextAssignment *m.HubImageAssignment
			var err error
			select {
			case a := <-get.Done:
				nextAssignment = a
			case e := <-get.Error:
				err = e
			}

			Expect(err).To(BeNil())
			Expect(nextAssignment).To(Equal(assignment))
			Expect(model.ImageScanQueue.Values()).To(Equal([]interface{}{}))
			Expect(model.Images[image1.Sha].ScanStatus).To(Equal(m.ScanStatusRunningScanClient))
			// TODO expected: time of image changed
		})

		It("no hubs, or all hubs inacessible", func() {
			model := m.NewModel(&m.Config{ConcurrentScanLimit: 3}, nil)
			model.AddImage(image1, 0)
			model.SetImageScanStatus(image1.Sha, m.ScanStatusInQueue)

			get := NewGetNextImage()
			go func() { get.Apply(model) }()
			nextImage := <-get.Done

			Expect(nextImage).To(BeNil())

			get = NewGetNextImage()
			go func() { get.Apply(model) }()
			nextImage = <-get.Done
			Expect(nextImage).To(BeNil())
		})
	})
}

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
	"sync"
	"testing"

	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	log "github.com/sirupsen/logrus"
)

// TestGetNextImageForScanningActionNoImageAvailable .....
func TestGetNextImageForScanningActionNoImageAvailable(t *testing.T) {
	// actual
	var nextImage *m.Image
	actual := m.NewModel(&m.Config{ConcurrentScanLimit: 3}, "test version")
	(&GetNextImage{func(image *m.Image) {
		nextImage = image
	}}).Apply(actual)
	// expected: front image removed from scan queue, status and time of image changed
	expected := *m.NewModel(&m.Config{ConcurrentScanLimit: 3}, "test version")

	assertEqual(t, nextImage, nil)
	log.Infof("%+v, %+v", actual, expected)
	// assertEqual(t, actual, expected)
}

// TestGetNextImage .....
func TestGetNextImage(t *testing.T) {
	model := m.NewModel(&m.Config{ConcurrentScanLimit: 3}, "test version")
	model.AddImage(image1)
	model.SetImageScanStatus(image1.Sha, m.ScanStatusInQueue)

	var nextImage *m.Image
	var wg sync.WaitGroup
	wg.Add(1)
	(&GetNextImage{func(image *m.Image) {
		nextImage = image
		wg.Done()
	}}).Apply(model)
	wg.Wait()

	if nextImage == nil {
		t.Errorf("expected %+v, got nil", image1)
	} else if *nextImage != image1 {
		t.Errorf("expected %+v, got %+v", nextImage, image1)
	}

	assertEqual(t, model.ImageScanQueue, []m.DockerImageSha{})
	assertEqual(t, model.Images[image1.Sha].ScanStatus, m.ScanStatusRunningScanClient)
	// TODO expected: time of image changed
}

// TestGetNextImageHubInaccessible .....
func TestGetNextImageHubInaccessible(t *testing.T) {
	model := m.NewModel(&m.Config{ConcurrentScanLimit: 3}, "test version")
	model.AddImage(image1)
	model.SetImageScanStatus(image1.Sha, m.ScanStatusInQueue)
	model.HubCircuitBreaker.HubFailure()

	var nextImage *m.Image
	var wg sync.WaitGroup
	wg.Add(1)
	(&GetNextImage{func(image *m.Image) {
		nextImage = image
		wg.Done()
	}}).Apply(model)
	wg.Wait()

	if nextImage != nil {
		t.Errorf("expected nil, got %+v", nextImage)
	}
}

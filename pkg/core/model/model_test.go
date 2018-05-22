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
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
)

func assertEqual(t *testing.T, message string, actual interface{}, expected interface{}) {
	if actual == nil && expected == nil {
		return
	}
	if reflect.DeepEqual(actual, expected) {
		return
	}
	// ?? can't compare []DockerImageSha with this ??
	// if actual == expected {
	// 	return
	// }
	actualBytes, err := json.Marshal(actual)
	if err != nil {
		t.Errorf("json serialization error: %s", err.Error())
		return
	}
	expectedBytes, err := json.Marshal(expected)
	if err != nil {
		t.Errorf("json serialization error: %s", err.Error())
		return
	}
	if string(actualBytes) == string(expectedBytes) {
		return
	}
	t.Errorf("%s: expected \n%s, got \n%s", message, string(expectedBytes), string(actualBytes))
}

// TestModelJSONSerialization .....
func TestModelJSONSerialization(t *testing.T) {
	m := NewModel(&Config{ConcurrentScanLimit: 3}, "test version")
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		t.Errorf("unabled to serialize model to json: %s", err.Error())
	}
	log.Infof("json bytes: %s", string(jsonBytes))
}

func removeItemModel() *Model {
	model := NewModel(&Config{ConcurrentScanLimit: 1}, "zzz")
	model.AddImage(image1)
	model.AddImage(image2)
	model.AddImage(image3)
	return model
}

func removeScanItemModel() *Model {
	model := NewModel(&Config{ConcurrentScanLimit: 1}, "zzz")
	model.AddImage(image1)
	model.AddImage(image2)
	model.AddImage(image3)
	model.SetImageScanStatus(image1.Sha, ScanStatusInQueue)
	model.SetImageScanStatus(image2.Sha, ScanStatusInQueue)
	model.SetImageScanStatus(image3.Sha, ScanStatusInQueue)
	return model
}

// TestModelRemoveItemFromFrontOfHubCheckQueue .....
func TestModelRemoveItemFromFrontOfHubCheckQueue(t *testing.T) {
	model := removeItemModel()
	model.removeImageFromHubCheckQueue(image1.Sha)
	assertEqual(t, "remove item from front of hub check queue", model.ImageHubCheckQueue, []DockerImageSha{image2.Sha, image3.Sha})
}

// TestModelRemoveItemFromMiddleOfHubCheckQueue .....
func TestModelRemoveItemFromMiddleOfHubCheckQueue(t *testing.T) {
	model := removeItemModel()
	model.removeImageFromHubCheckQueue(image2.Sha)
	assertEqual(t, "", model.ImageHubCheckQueue, []DockerImageSha{image1.Sha, image3.Sha})
}

// TestModelRemoveItemFromEndOfHubCheckQueue .....
func TestModelRemoveItemFromEndOfHubCheckQueue(t *testing.T) {
	model := removeItemModel()
	model.removeImageFromHubCheckQueue(image3.Sha)
	assertEqual(t, "", model.ImageHubCheckQueue, []DockerImageSha{image1.Sha, image2.Sha})
}

// TestModelRemoveAllItemsFromHubCheckQueue .....
func TestModelRemoveAllItemsFromHubCheckQueue(t *testing.T) {
	model := removeItemModel()
	model.removeImageFromHubCheckQueue(image1.Sha)
	model.removeImageFromHubCheckQueue(image2.Sha)
	model.removeImageFromHubCheckQueue(image3.Sha)
	assertEqual(t, "", model.ImageHubCheckQueue, []DockerImageSha{})
}

// TestModelRemoveItemFromFrontOfScanQueue .....
func TestModelRemoveItemFromFrontOfScanQueue(t *testing.T) {
	model := removeScanItemModel()
	model.SetImageScanStatus(image1.Sha, ScanStatusRunningScanClient)
	assertEqual(t, "remove from front of queue", model.ImageScanQueue, []DockerImageSha{image2.Sha, image3.Sha})
}

// TestModelRemoveItemFromMiddleOfScanQueue .....
func TestModelRemoveItemFromMiddleOfScanQueue(t *testing.T) {
	model := removeScanItemModel()
	model.SetImageScanStatus(image2.Sha, ScanStatusRunningScanClient)
	assertEqual(t, "remove from middle of queue", model.ImageScanQueue, []DockerImageSha{image1.Sha, image3.Sha})
}

// TestModelRemoveItemFromEndOfScanQueue .....
func TestModelRemoveItemFromEndOfScanQueue(t *testing.T) {
	model := removeScanItemModel()
	model.SetImageScanStatus(image3.Sha, ScanStatusRunningScanClient)
	assertEqual(t, "remove from end of queue", model.ImageScanQueue, []DockerImageSha{image1.Sha, image2.Sha})
}

// TestModelRemoveAllItemsFromScanQueue .....
func TestModelRemoveAllItemsFromScanQueue(t *testing.T) {
	model := removeScanItemModel()
	model.SetImageScanStatus(image1.Sha, ScanStatusRunningScanClient)
	model.SetImageScanStatus(image2.Sha, ScanStatusRunningScanClient)
	model.SetImageScanStatus(image3.Sha, ScanStatusRunningScanClient)
	assertEqual(t, "remove all items", model.ImageScanQueue, []DockerImageSha{})
}

// TestModelRefreshQueueOperations .....
func TestModelRefreshQueueOperations(t *testing.T) {
	model := removeItemModel()
	for _, image := range []Image{image1, image2, image3} {
		model.SetImageScanStatus(image.Sha, ScanStatusComplete)
		err := model.AddImageToRefreshQueue(image.Sha)
		if err != nil {
			t.Errorf("unable to add sha %s to image refresh queue: %s", image.Sha, err.Error())
		}
	}
	assertEqual(t, "refresh queue: all items", model.ImageRefreshQueue, []DockerImageSha{image1.Sha, image2.Sha, image3.Sha})

	image := model.GetNextImageFromRefreshQueue()
	assertEqual(t, "next Image from refresh queue", image, image1)

	err := model.RemoveImageFromRefreshQueue(image2.Sha)
	if err != nil {
		t.Errorf("unable to remove sha %s from refresh queue: %s", image2.Sha, err.Error())
	}
	assertEqual(t, "refresh queue: 1 and 3", model.ImageRefreshQueue, []DockerImageSha{image1.Sha, image3.Sha})

	image = model.GetNextImageFromRefreshQueue()
	assertEqual(t, "next Image from refresh queue", image, image1)

	err = model.RemoveImageFromRefreshQueue(image1.Sha)
	if err != nil {
		t.Errorf("unable to remove sha %s from refresh queue: %s", image2.Sha, err.Error())
	}
	assertEqual(t, "refresh queue: 3", model.ImageRefreshQueue, []DockerImageSha{image3.Sha})

	image = model.GetNextImageFromRefreshQueue()
	assertEqual(t, "next Image from refresh queue", image, image3)

	err = model.RemoveImageFromRefreshQueue(image3.Sha)
	if err != nil {
		t.Errorf("unable to remove sha %s from refresh queue: %s", image2.Sha, err.Error())
	}
	assertEqual(t, "refresh queue: 1 and 3", model.ImageRefreshQueue, []DockerImageSha{})

	image = model.GetNextImageFromRefreshQueue()
	assertEqual(t, "next Image from refresh queue", image, nil)
}

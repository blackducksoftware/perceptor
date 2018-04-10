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
	"testing"
)

func TestImageJSON(t *testing.T) {
	jsonString := `{"Name":"docker.io/mfenwickbd/perceptor","Sha":"04bb619150cd99cfb21e76429c7a5c2f4545775b07456cb6b9c866c8aff9f9e5","DockerImage":"docker.io/mfenwickbd/perceptor:latest"}`
	var image Image
	err := json.Unmarshal([]byte(jsonString), &image)
	if err != nil {
		t.Errorf("unable to parse %s as JSON: %v", jsonString, err)
		t.Fail()
		panic("a")
	}
	expectedName := "docker.io/mfenwickbd/perceptor"
	if image.Name != expectedName {
		t.Errorf("expected name of %s, got %s", expectedName, image.Name)
	}
}

func TestImageHubData(t *testing.T) {
	image := NewImage("abc", DockerImageSha("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"))
	actualProject := image.HubProjectName()
	expectedProject := "abc-abcdefghijklmnopqrst"
	if actualProject != expectedProject {
		t.Errorf("hub project name: expected %s, got %s", expectedProject, actualProject)
	}
	actualProjectVersion := image.HubProjectVersionName()
	expectedProjectVersion := "abcdefghijklmnopqrst"
	if image.HubProjectVersionName() != expectedProjectVersion {
		t.Errorf("hub project version name: expected %s, got %s", expectedProjectVersion, actualProjectVersion)
	}
	actualScan := image.HubScanName()
	expectedScan := "abcdefghijklmnopqrst"
	if image.HubScanName() != expectedScan {
		t.Errorf("hub scan name: expected %s, got %s", expectedScan, actualScan)
	}
	actualProjectSearch := image.HubProjectNameSearchString()
	expectedProjectSearch := "abcdefghijklmnopqrst"
	if image.HubProjectNameSearchString() != expectedProjectSearch {
		t.Errorf("hub project search string: expected %s, got %s", expectedProjectSearch, actualProjectSearch)
	}
	actualVersionSearch := image.HubProjectVersionNameSearchString()
	expectedVersionSearch := "abcdefghijklmnopqrst"
	if image.HubProjectVersionNameSearchString() != expectedVersionSearch {
		t.Errorf("hub project version search string: expected %s, got %s", expectedVersionSearch, actualVersionSearch)
	}
	actualScanSearch := image.HubScanNameSearchString()
	expectedScanSearch := "abcdefghijklmnopqrst"
	if image.HubScanNameSearchString() != expectedScanSearch {
		t.Errorf("hub scan search string: expected %s, got %s", expectedScanSearch, actualScanSearch)
	}
}

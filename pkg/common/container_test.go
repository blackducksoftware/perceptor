/*
Copyright (C) 2018 Black Duck Software, Inc.

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

package common

import "testing"

func TestNewContainer(t *testing.T) {
	c1 := NewContainer(*NewImage("imgName", "sha", "dockerImage"), "name")
	c2 := NewContainer(*NewImage("imgName", "sha", "dockerImage"), "name")

	if c1.Name != "name" {
		t.Errorf("Expected name to be set to 'name")
	}

	if c1.Image.Name != "imgName" {
		t.Errorf("Expected container's image to have the name 'imgName'")
	}

	if c1.Image.Sha != "sha" {
		t.Errorf("Expected container's Sha to have the value 'sha'")
	}

	if c1.Image.DockerImage != "dockerImage" {
		t.Errorf("Expected container's DockerImage to have the value 'dockerImage'")
	}

	if &c1 == &c2 {
		t.Errorf("We should create unique container instances")
	}

	if c1 == c2 {
		t.Errorf("Two containers with the same internal values should be equal")
	}
}
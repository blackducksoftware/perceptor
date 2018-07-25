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
	"fmt"
)

// Image .....
type Image struct {
	// Name combines Host, User, and Project
	Name string
	Sha  DockerImageSha
}

// NewImage .....
func NewImage(name string, sha DockerImageSha) *Image {
	return &Image{Name: name, Sha: sha}
}

func (image Image) shaPrefix() string {
	return string(image.Sha)[:20]
}

// These strings are for the scanner

// HubProjectName .....
func (image Image) HubProjectName() string {
	return image.Name
}

// HubProjectVersionName .....
func (image Image) HubProjectVersionName() string {
	// TODO add tag if available
	return image.shaPrefix()
}

// HubScanName .....
func (image Image) HubScanName() string {
	return string(image.Sha)
}

// PullSpec combines Name with the image sha and should be pullable by Docker
func (image *Image) PullSpec() string {
	return fmt.Sprintf("%s@sha256:%s", image.Name, image.Sha)
}

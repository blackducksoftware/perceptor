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
	Repository              string
	Tag                     string
	Sha                     DockerImageSha
	Priority                int
	BlackDuckProjectName    string
	BlackDuckProjectVersion string
}

// NewImage .....
func NewImage(repository string, tag string, sha DockerImageSha, priority int, blackDuckProjectName string, blackDuckProjectVersion string) *Image {
	return &Image{Repository: repository, Tag: tag, Sha: sha, Priority: priority, BlackDuckProjectName: blackDuckProjectName, BlackDuckProjectVersion: blackDuckProjectVersion}
}

func (image Image) shaPrefix() string {
	return string(image.Sha)[:20]
}

// These strings are for the scanner

// HubProjectName .....
func (image Image) HubProjectName() string {
	if image.BlackDuckProjectName != "" {
		return image.BlackDuckProjectName
	}
	return image.Repository
}

// HubProjectVersionName .....
func (image Image) HubProjectVersionName() string {
	if image.BlackDuckProjectVersion != "" {
		return image.BlackDuckProjectVersion
	}

	tag := ""
	if image.Tag != "" {
		tag = image.Tag + "-"
	}
	return fmt.Sprintf("%s%s", tag, image.shaPrefix())
}

// HubScanName .....
func (image Image) HubScanName() string {
	return string(image.Sha)
}

// PullSpec combines repository with sha and should be pullable by Docker
func (image *Image) PullSpec() string {
	return fmt.Sprintf("%s@sha256:%s", image.Repository, image.Sha)
}

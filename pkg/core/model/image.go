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

type Image struct {
	// Name combines Host, User, and Project
	Name string
	Sha  DockerImageSha
}

func NewImage(name string, sha DockerImageSha) *Image {
	return &Image{Name: name, Sha: sha}
}

func (image Image) shaPrefix() string {
	return string(image.Sha)[:20]
}

// These strings are for the scanner

func (image Image) HubProjectName() string {
	return fmt.Sprintf("%s-%s", image.Name, image.shaPrefix())
}

func (image Image) HubProjectVersionName() string {
	return image.shaPrefix()
}

func (image Image) HubScanName() string {
	return image.shaPrefix()
}

// These strings are for the hub fetcher
// For the hub project name, we want to include a meaningful, human-readable
//   string -- so we add the docker image name of the first image to have this
//   sha.
// But when we search for the project, we *only* want to search by sha --
//   in case the docker image name is different.
// This is weird, but allows to both:
//  - have our hub projects be searchable by sha, regardless of docker image name
//  - have a meaningful, human-readable hub project name

func (image Image) HubProjectNameSearchString() string {
	return image.shaPrefix()
}

func (image Image) HubProjectVersionNameSearchString() string {
	return image.shaPrefix()
}

func (image Image) HubScanNameSearchString() string {
	return image.shaPrefix()
}

// HumanReadableName returns a nice, easy to read string
func (image *Image) HumanReadableName() string {
	return image.Name
}

// PullSpec combines Name with the image sha and should be pullable by Docker
func (image *Image) PullSpec() string {
	return fmt.Sprintf("%s@sha256:%s", image.Name, image.Sha)
}

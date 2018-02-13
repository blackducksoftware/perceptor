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

package core

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

func (image Image) HubProjectName() string {
	return fmt.Sprintf("%s-%s", image.Name, string(image.Sha))
}

func (image Image) HubProjectVersionName() string {
	return string(image.Sha)
}

func (image Image) HubScanName() string {
	return string(image.Sha)
}

// HumanReadableName returns a nice, easy to read string
func (image *Image) HumanReadableName() string {
	return image.Name
}

// PullSpec combines Name with the image sha and should be pullable by Docker
func (image *Image) PullSpec() string {
	return fmt.Sprintf("%s@sha256:%s", image.Name, image.Sha)
}

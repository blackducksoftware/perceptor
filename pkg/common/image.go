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

import (
	"fmt"
	"net/url"
	"strings"
)

type Image struct {
	// Name combines Host, User, and Project
	// DockerImage is the kubernetes .Image string, which may or may not include the registry, user, tag, and sha
	//   DockerImage should probably only be used as a human-readable string, not for storing or organizing
	//   data, because it is so nebulous and ambiguous.
	Name        string
	Sha         string
	DockerImage string
}

func NewImage(name string, sha string, dockerImage string) *Image {
	return &Image{Name: name, Sha: sha, DockerImage: dockerImage}
}

func (image *Image) HubProjectName() string {
	return image.Name
}

func (image *Image) HubVersionName() string {
	return image.Sha
}

// HubScanName has to be unique; otherwise, multiple
// code locations could be mapped to the same scan,
// which would be confusing
func (image *Image) HubScanName() string {
	return image.Sha
}

// Name returns a nice, easy to read string
func (image *Image) HumanReadableName() string {
	return image.DockerImage
}

// FullName combines Name with the image sha
func (image *Image) ShaName() string {
	return fmt.Sprintf("%s@sha256:%s", image.Name, image.Sha)
}

func (image *Image) TarFilePath() string {
	filePath := strings.Replace(image.ShaName(), "/", "_", -1)
	return fmt.Sprintf("./tmp/%s.tar", filePath)
}

func (image *Image) URLEncodedName() string {
	return url.QueryEscape(image.ShaName())
}

// CreateURL returns the URL used for hitting the docker daemon's create endpoint
func (image *Image) CreateURL() string {
	// TODO v1.24 refers to the docker version.  figure out how to avoid hard-coding this
	// TODO can probably use the docker api code for this
	return fmt.Sprintf("http://localhost/v1.24/images/create?fromImage=%s", image.URLEncodedName())
	//	return fmt.Sprintf("http://localhost/v1.24/images/create?fromImage=%s&tag=%s", image.name, image.tag)
}

// GetURL returns the URL used for hitting the docker daemon's get endpoint
func (image *Image) GetURL() string {
	return fmt.Sprintf("http://localhost/v1.24/images/%s/get", image.URLEncodedName())
}

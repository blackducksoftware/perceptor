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
	"time"

	"github.com/blackducksoftware/perceptor/pkg/hub"
)

// ImageInfo .....
type ImageInfo struct {
	ScanStatus             ScanStatus
	TimeOfLastStatusChange time.Time
	TimeOfLastRefresh      time.Time
	ScanResults            *hub.ScanResults
	ImageSha               DockerImageSha
	RepoTags               []*RepoTag
}

// NewImageInfo .....
func NewImageInfo(sha DockerImageSha, repoTag *RepoTag) *ImageInfo {
	imageInfo := &ImageInfo{
		ScanResults: nil,
		ImageSha:    sha,
		RepoTags:    []*RepoTag{repoTag},
	}
	imageInfo.setScanStatus(ScanStatusUnknown)
	return imageInfo
}

func (imageInfo *ImageInfo) setScanStatus(newStatus ScanStatus) {
	imageInfo.ScanStatus = newStatus
	imageInfo.TimeOfLastStatusChange = time.Now()
}

// SetScanResults .....
func (imageInfo *ImageInfo) SetScanResults(results *hub.ScanResults) {
	imageInfo.ScanResults = results
	imageInfo.TimeOfLastRefresh = time.Now()
}

// TimeInCurrentScanStatus .....
func (imageInfo *ImageInfo) TimeInCurrentScanStatus() time.Duration {
	return time.Now().Sub(imageInfo.TimeOfLastStatusChange)
}

// Image .....
func (imageInfo *ImageInfo) Image() Image {
	repoTag := imageInfo.FirstRepoTag()
	return *NewImage(repoTag.Repository, repoTag.Tag, imageInfo.ImageSha)
}

// AddImageName .....
func (imageInfo *ImageInfo) AddRepoTag(repoTag *RepoTag) {
	if !arrayContains(imageInfo.RepoTags, repoTag) {
		imageInfo.RepoTags = append(imageInfo.RepoTags, repoTag)
	}
}

// FirstRepoTag .....
func (imageInfo *ImageInfo) FirstRepoTag() *RepoTag {
	if len(imageInfo.RepoTags) == 0 {
		panic(fmt.Errorf("expected at least 1 RepoTag, found 0"))
	}
	return imageInfo.RepoTags[0]
}

func arrayContains(array []*RepoTag, value *RepoTag) bool {
	for _, item := range array {
		if item == value {
			return true
		}
	}
	return false
}

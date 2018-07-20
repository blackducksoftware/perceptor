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
	ImageNames             []string
	HubURL                 string
}

// NewImageInfo .....
func NewImageInfo(sha DockerImageSha, imageName string) *ImageInfo {
	imageInfo := &ImageInfo{
		ScanResults: nil,
		ImageSha:    sha,
		ImageNames:  []string{imageName},
	}
	imageInfo.setScanStatus(ScanStatusUnknown)
	return imageInfo
}

func (imageInfo *ImageInfo) setHubURL(hubURL string) error {
	if imageInfo.HubURL != "" {
		return fmt.Errorf("Hub URL already set to %s", imageInfo.HubURL)
	}
	imageInfo.HubURL = hubURL
	return nil
}

func (imageInfo *ImageInfo) removeHubURL() error {
	if imageInfo.HubURL == "" {
		return fmt.Errorf("hub URL not set")
	}
	imageInfo.HubURL = ""
	return nil
}

func (imageInfo *ImageInfo) isAssignedHub() bool {
	return imageInfo.HubURL != ""
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
	return *NewImage(imageInfo.FirstImageName(), imageInfo.ImageSha)
}

// AddImageName .....
func (imageInfo *ImageInfo) AddImageName(imageName string) {
	if !arrayContains(imageInfo.ImageNames, imageName) {
		imageInfo.ImageNames = append(imageInfo.ImageNames, imageName)
	}
}

// FirstImageName .....
func (imageInfo *ImageInfo) FirstImageName() string {
	if len(imageInfo.ImageNames) == 0 {
		panic(fmt.Errorf("expected at least 1 imageName, found 0"))
	}
	return imageInfo.ImageNames[0]
}

func arrayContains(array []string, value string) bool {
	for _, item := range array {
		if item == value {
			return true
		}
	}
	return false
}

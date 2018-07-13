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

	log "github.com/sirupsen/logrus"
)

// ImageInfo .....
type ImageInfo struct {
	ImageSha   DockerImageSha
	ImageNames []string
	Layers     map[string]bool
}

// NewImageInfo .....
func NewImageInfo(sha DockerImageSha, imageName string) *ImageInfo {
	imageInfo := &ImageInfo{
		ImageSha:   sha,
		ImageNames: []string{imageName},
		Layers:     nil,
	}
	return imageInfo
}

// SetLayers returns an error if layers have already been set, and succeeds otherwise.
// It ignores duplicate layers.
func (imageInfo *ImageInfo) SetLayers(layers []string) error {
	if imageInfo.Layers != nil {
		return fmt.Errorf("cannot set layers for image %s, already set (have %d, %d attempted to be added)", imageInfo.ImageSha, len(imageInfo.Layers), len(layers))
	}
	layerSet := map[string]bool{}
	for _, layer := range layers {
		if layerSet[layer] {
			log.Warnf("ignoring duplicate layer %s from %+v", layer, layers)
		}
		layerSet[layer] = true
	}
	imageInfo.Layers = layerSet
	return nil
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

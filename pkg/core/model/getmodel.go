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
	"github.com/blackducksoftware/perceptor/pkg/api"
)

// GetModel .....
type GetModel struct {
	Done chan api.CoreModel
}

// NewGetModel .....
func NewGetModel() *GetModel {
	return &GetModel{Done: make(chan api.CoreModel)}
}

// Apply .....
func (g *GetModel) Apply(model *Model) error {
	apiModel := CoreModelToAPIModel(model)
	go func() {
		g.Done <- apiModel
	}()
	return nil
}

// CoreContainerToAPIContainer .....
func CoreContainerToAPIContainer(coreContainer Container) *api.Container {
	image := coreContainer.Image
	priority := image.Priority
	return &api.Container{
		Image: *api.NewImage(image.Repository, image.Tag, string(image.Sha), &priority),
		Name:  coreContainer.Name,
	}
}

// CorePodToAPIPod .....
func CorePodToAPIPod(corePod Pod) *api.Pod {
	containers := []api.Container{}
	for _, coreContainer := range corePod.Containers {
		containers = append(containers, *CoreContainerToAPIContainer(coreContainer))
	}
	return &api.Pod{
		Containers: containers,
		Name:       corePod.Name,
		Namespace:  corePod.Namespace,
		UID:        corePod.UID,
	}
}

// CoreModelToAPIModel .....
func CoreModelToAPIModel(model *Model) api.CoreModel {
	// pods
	pods := map[string]*api.Pod{}
	for podName, pod := range model.Pods {
		pods[podName] = CorePodToAPIPod(pod)
	}
	// images
	images := map[string]*api.ModelImageInfo{}
	for imageSha, imageInfo := range model.Images {
		repoTags := []*api.ModelRepoTag{}
		for _, repoTag := range imageInfo.RepoTags {
			repoTags = append(repoTags, &api.ModelRepoTag{Repository: repoTag.Repository, Tag: repoTag.Tag})
		}
		images[string(imageSha)] = &api.ModelImageInfo{
			RepoTags:               repoTags,
			ImageSha:               string(imageInfo.ImageSha),
			ScanResults:            imageInfo.ScanResults,
			ScanStatus:             imageInfo.ScanStatus.String(),
			TimeOfLastStatusChange: imageInfo.TimeOfLastStatusChange.String(),
			Priority:               imageInfo.Priority,
		}
	}

	// return value
	return api.CoreModel{
		Pods:           pods,
		Images:         images,
		ImageScanQueue: model.ImageScanQueue.Dump(),
	}
}

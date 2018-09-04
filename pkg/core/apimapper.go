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

package core

import (
	"github.com/blackducksoftware/perceptor/pkg/api"
	"github.com/blackducksoftware/perceptor/pkg/core/model"
)

// api -> model

// APIImageToCoreImage .....
func APIImageToCoreImage(apiImage api.Image) (*model.Image, error) {
	sha, err := model.NewDockerImageSha(apiImage.Sha)
	if err != nil {
		return nil, err
	}
	priority := 0
	if apiImage.Priority != nil {
		priority = *apiImage.Priority
	}
	return model.NewImage(apiImage.Repository, apiImage.Tag, sha, priority), nil
}

// APIContainerToCoreContainer .....
func APIContainerToCoreContainer(apiContainer api.Container) (*model.Container, error) {
	image, err := APIImageToCoreImage(apiContainer.Image)
	if err != nil {
		return nil, err
	}
	return model.NewContainer(*image, apiContainer.Name), nil
}

// APIPodToCorePod .....
func APIPodToCorePod(apiPod api.Pod) (*model.Pod, error) {
	containers := []model.Container{}
	for _, apiContainer := range apiPod.Containers {
		container, err := APIContainerToCoreContainer(apiContainer)
		if err != nil {
			return nil, err
		}
		if apiContainer.Image.Priority == nil {
			container.Image.Priority = 1
		}
		containers = append(containers, *container)
	}
	return model.NewPod(apiPod.Name, apiPod.UID, apiPod.Namespace, containers), nil
}

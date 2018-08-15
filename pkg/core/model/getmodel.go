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
	log "github.com/sirupsen/logrus"
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
func (g *GetModel) Apply(model *Model) {
	apiModel := CoreModelToAPIModel(model)
	go func() {
		g.Done <- apiModel
	}()
}

// CoreContainerToAPIContainer .....
func CoreContainerToAPIContainer(coreContainer Container) *api.Container {
	image := coreContainer.Image
	return &api.Container{
		Image: *api.NewImage(image.Repository, image.Tag, string(image.Sha)),
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
		}
	}

	// return value
	return api.CoreModel{
		Pods:   pods,
		Images: images,
		// TODO
		// ImagePriority: model.ImagePriority,
		// HubVersion:     model.HubVersion,
		// ImageScanQueue: model.ImageScanQueue.Dump(),
		// Config: &api.ModelConfig{
		// 	HubHost:             model.Config.HubHost,
		// 	HubUser:             model.Config.HubUser,
		// 	HubPort:             model.Config.HubPort,
		// 	LogLevel:            model.Config.LogLevel,
		// 	Port:                model.Config.Port,
		// 	ConcurrentScanLimit: model.Config.ConcurrentScanLimit,
		// },
		// Timings: &api.ModelTimings{
		// 	CheckForStalledScansPause:      *api.NewModelTime(model.Timings.CheckForStalledScansPause),
		// 	CheckHubForCompletedScansPause: *api.NewModelTime(model.Timings.CheckHubForCompletedScansPause),
		// 	CheckHubThrottle:               *api.NewModelTime(model.Timings.CheckHubThrottle),
		// 	EnqueueImagesForRefreshPause:   *api.NewModelTime(model.Timings.EnqueueImagesForRefreshPause),
		// 	HubClientTimeout:               *api.NewModelTime(model.Timings.HubClientTimeout),
		// 	HubReloginPause:                *api.NewModelTime(model.Timings.HubReloginPause),
		// 	ModelMetricsPause:              *api.NewModelTime(model.Timings.ModelMetricsPause),
		// 	RefreshImagePause:              *api.NewModelTime(model.Timings.RefreshImagePause),
		// 	RefreshThresholdDuration:       *api.NewModelTime(model.Timings.RefreshThresholdDuration),
		// 	StalledScanClientTimeout:       *api.NewModelTime(model.Timings.StalledScanClientTimeout),
	}
}

// ScanResults .....
func ScanResults(model *Model) api.ScanResults {
	// pods
	pods := []api.ScannedPod{}
	for podName, pod := range model.Pods {
		podScan, err := scanResultsForPod(model, podName)
		if err != nil {
			log.Errorf("unable to retrieve scan results for Pod %s: %s", podName, err.Error())
			continue
		}
		if podScan == nil {
			log.Debugf("image scans not complete for pod %s, skipping (pod info: %+v)", podName, pod)
			continue
		}
		pods = append(pods, api.ScannedPod{
			Namespace:        pod.Namespace,
			Name:             pod.Name,
			PolicyViolations: podScan.PolicyViolations,
			Vulnerabilities:  podScan.Vulnerabilities,
			OverallStatus:    podScan.OverallStatus.String()})
	}

	// images
	images := []api.ScannedImage{}
	for sha, imageInfo := range model.Images {
		if imageInfo.ScanStatus != ScanStatusComplete {
			continue
		}
		if imageInfo.ScanResults == nil {
			log.Errorf("model inconsistency: found ScanStatusComplete for image %s, but nil ScanResults (imageInfo %+v)", sha, imageInfo)
			continue
		}
		image := imageInfo.Image()
		apiImage := api.ScannedImage{
			Repository:       image.Repository,
			Tag:              image.Tag,
			Sha:              string(image.Sha),
			PolicyViolations: imageInfo.ScanResults.PolicyViolationCount(),
			Vulnerabilities:  imageInfo.ScanResults.VulnerabilityCount(),
			OverallStatus:    imageInfo.ScanResults.OverallStatus().String(),
			ComponentsURL:    imageInfo.ScanResults.ComponentsHref}
		images = append(images, apiImage)
	}

	return *api.NewScanResults(pods, images)
}

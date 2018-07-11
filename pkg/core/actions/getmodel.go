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

package actions

import (
	"github.com/blackducksoftware/perceptor/pkg/api"
	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	log "github.com/sirupsen/logrus"
)

// GetModel .....
type GetModel struct {
	HubCircuitBreaker *api.ModelCircuitBreaker
	Done              chan *api.Model
}

// NewGetModel .....
func NewGetModel() *GetModel {
	return &GetModel{Done: make(chan *api.Model)}
}

// Apply .....
func (g *GetModel) Apply(model *m.Model) {
	apiModel := CoreModelToAPIModel(model)
	apiModel.HubCircuitBreaker = g.HubCircuitBreaker
	go func() {
		g.Done <- apiModel
	}()
}

// CoreContainerToAPIContainer .....
func CoreContainerToAPIContainer(coreContainer m.Container) *api.Container {
	return &api.Container{
		Image: api.Image{
			Name: coreContainer.Image.Name,
			Sha:  string(coreContainer.Image.Sha),
		},
		Name: coreContainer.Name,
	}
}

// CorePodToAPIPod .....
func CorePodToAPIPod(corePod m.Pod) *api.Pod {
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
func CoreModelToAPIModel(model *m.Model) *api.Model {
	// pods
	pods := map[string]*api.Pod{}
	for podName, pod := range model.Pods {
		pods[podName] = CorePodToAPIPod(pod)
	}
	// images
	images := map[string]*api.ModelImageInfo{}
	for imageSha, imageInfo := range model.Images {
		layers := []string{}
		for layerSha := range imageInfo.Layers {
			layers = append(layers, layerSha)
		}
		images[string(imageSha)] = &api.ModelImageInfo{
			ImageNames: imageInfo.ImageNames,
			ImageSha:   string(imageInfo.ImageSha),
			Layers:     layers,
		}
	}
	layers := map[string]*api.ModelLayerInfo{}
	for layerSha, layerInfo := range model.Layers {
		layers[layerSha] = &api.ModelLayerInfo{
			ImageSha:               string(layerInfo.ImageSha),
			ScanResults:            layerInfo.ScanResults,
			ScanStatus:             layerInfo.ScanStatus.String(),
			TimeOfLastStatusChange: layerInfo.TimeOfLastStatusChange.String(),
		}
	}
	// hub check queue
	hubQueue := make([]string, len(model.LayerHubCheckQueue))
	for i, image := range model.LayerHubCheckQueue {
		hubQueue[i] = string(image)
	}
	// return value
	return &api.Model{
		Pods:               pods,
		Images:             images,
		Layers:             layers,
		HubVersion:         model.HubVersion,
		LayerHubCheckQueue: hubQueue,
		ImageScanQueue:     model.ImageScanQueue.Dump(),
		Config: &api.ModelConfig{
			HubHost:             model.Config.HubHost,
			HubUser:             model.Config.HubUser,
			HubPort:             model.Config.HubPort,
			LogLevel:            model.Config.LogLevel,
			Port:                model.Config.Port,
			ConcurrentScanLimit: model.Config.ConcurrentScanLimit,
		},
		Timings: &api.ModelTimings{
			CheckForStalledScansPause:      *api.NewModelTime(model.Timings.CheckForStalledScansPause),
			CheckHubForCompletedScansPause: *api.NewModelTime(model.Timings.CheckHubForCompletedScansPause),
			CheckHubThrottle:               *api.NewModelTime(model.Timings.CheckHubThrottle),
			EnqueueImagesForRefreshPause:   *api.NewModelTime(model.Timings.EnqueueImagesForRefreshPause),
			HubClientTimeout:               *api.NewModelTime(model.Timings.HubClientTimeout),
			HubReloginPause:                *api.NewModelTime(model.Timings.HubReloginPause),
			ModelMetricsPause:              *api.NewModelTime(model.Timings.ModelMetricsPause),
			RefreshImagePause:              *api.NewModelTime(model.Timings.RefreshImagePause),
			RefreshThresholdDuration:       *api.NewModelTime(model.Timings.RefreshThresholdDuration),
			StalledScanClientTimeout:       *api.NewModelTime(model.Timings.StalledScanClientTimeout),
		},
	}
}

// ScanResults .....
func ScanResults(model *m.Model) api.ScanResults {
	// pods
	pods := []api.ScannedPod{}
	for podName, pod := range model.Pods {
		podScan, err := model.ScanResultsForPod(podName)
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
			OverallStatus:    podScan.OverallStatus})
	}

	// images
	images := []api.ScannedImage{}
	// for sha, imageInfo := range model.Images {
	// 	TODO
	// 	if imageInfo.ScanStatus != m.ScanStatusComplete {
	// 		continue
	// 	}
	// 	if imageInfo.ScanResults == nil {
	// 		log.Errorf("model inconsistency: found ScanStatusComplete for image %s, but nil ScanResults (imageInfo %+v)", sha, imageInfo)
	// 		continue
	// 	}
	// 	image := imageInfo.Image()
	// 	apiImage := api.ScannedImage{
	// 		Name:             image.HumanReadableName(),
	// 		Sha:              string(image.Sha),
	// 		PolicyViolations: imageInfo.ScanResults.PolicyViolationCount(),
	// 		Vulnerabilities:  imageInfo.ScanResults.VulnerabilityCount(),
	// 		OverallStatus:    imageInfo.ScanResults.OverallStatus().String(),
	// 		ComponentsURL:    imageInfo.ScanResults.ComponentsHref}
	// 	images = append(images, apiImage)
	// }

	return *api.NewScanResults(model.HubVersion, model.HubVersion, pods, images)
}

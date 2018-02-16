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
	"fmt"

	"github.com/blackducksoftware/perceptor/pkg/api"
	"github.com/prometheus/common/log"
)

// api -> model

func newImage(apiImage api.Image) *Image {
	return NewImage(apiImage.Name, DockerImageSha(apiImage.Sha))
}

func newContainer(apiContainer api.Container) *Container {
	return NewContainer(*newImage(apiContainer.Image), apiContainer.Name)
}

func newPod(apiPod api.Pod) *Pod {
	containers := []Container{}
	for _, apiContainer := range apiPod.Containers {
		containers = append(containers, *newContainer(apiContainer))
	}
	return NewPod(apiPod.Name, apiPod.UID, apiPod.Namespace, containers)
}

// model -> api

func (model *Model) scanResultsForPod(podName string) (int, int, string, error) {
	pod, ok := model.Pods[podName]
	if !ok {
		return 0, 0, "", fmt.Errorf("could not find pod of name %s in cache", podName)
	}

	overallStatus := "NOT_IN_VIOLATION"
	policyViolationCount := 0
	vulnerabilityCount := 0
	for _, container := range pod.Containers {
		imageInfo, ok := model.Images[container.Image.Sha]
		if !ok {
			continue
		}
		if imageInfo.ScanStatus != ScanStatusComplete {
			continue
		}
		if imageInfo.ScanResults == nil {
			continue
		}
		policyViolationCount += imageInfo.ScanResults.PolicyViolationCount()
		vulnerabilityCount += imageInfo.ScanResults.VulnerabilityCount()
		imageScanOverallStatus := imageInfo.ScanResults.OverallStatus()
		if imageScanOverallStatus != "NOT_IN_VIOLATION" && imageScanOverallStatus != "" {
			overallStatus = imageScanOverallStatus
		}
	}
	return policyViolationCount, vulnerabilityCount, overallStatus, nil
}

func (model *Model) scanResults() api.ScanResults {
	pods := []api.ScannedPod{}
	images := []api.ScannedImage{}
	// pods
	for podName, pod := range model.Pods {
		skipPod := false
		for _, cont := range pod.Containers {
			imageSha := cont.Image.Sha
			imageInfo, ok := model.Images[imageSha]
			if !ok {
				log.Errorf("expected to find Image %s, but did not", string(imageSha))
				continue
			}
			if imageInfo.ScanStatus != ScanStatusComplete {
				skipPod = true
				break
			}
		}
		if skipPod {
			continue
		}
		policyViolationCount, vulnerabilityCount, overallStatus, err := model.scanResultsForPod(podName)
		if err != nil {
			log.Errorf("unable to retrieve scan results for Pod %s: %s", podName, err.Error())
			continue
		}
		pods = append(pods, api.ScannedPod{
			Namespace:        pod.Namespace,
			Name:             pod.Name,
			PolicyViolations: policyViolationCount,
			Vulnerabilities:  vulnerabilityCount,
			OverallStatus:    overallStatus})
	}
	// images
	for _, imageInfo := range model.Images {
		if imageInfo.ScanStatus != ScanStatusComplete {
			continue
		}
		componentsURL := ""
		overallStatus := ""
		policyViolations := 0
		vulnerabilities := 0
		if imageInfo.ScanResults != nil {
			policyViolations = imageInfo.ScanResults.PolicyViolationCount()
			vulnerabilities = imageInfo.ScanResults.VulnerabilityCount()
			componentsURL = imageInfo.ScanResults.ComponentsHref
			overallStatus = imageInfo.ScanResults.OverallStatus()
		}
		image := imageInfo.image()
		apiImage := api.ScannedImage{
			Name:             image.HumanReadableName(),
			Sha:              string(image.Sha),
			PolicyViolations: policyViolations,
			Vulnerabilities:  vulnerabilities,
			OverallStatus:    overallStatus,
			ComponentsURL:    componentsURL}
		images = append(images, apiImage)
	}
	return *api.NewScanResults(model.Config.HubScanClientVersion, model.Config.HubVersion, pods, images)
}

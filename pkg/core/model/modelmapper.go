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

	"github.com/blackducksoftware/perceptor/pkg/hub"
	log "github.com/sirupsen/logrus"
)

// ScanResultsForPod .....
func (model *Model) ScanResultsForPod(podName string) (*ScanResults, error) {
	pod, ok := model.Pods[podName]
	if !ok {
		return nil, fmt.Errorf("could not find pod of name %s in cache", podName)
	}

	overallStatus := hub.PolicyStatusTypeNotInViolation
	policyViolationCount := 0
	vulnerabilityCount := 0
	for _, container := range pod.Containers {
		imageScan, err := model.ScanResultsForImage(container.Image.Sha)
		if err != nil {
			log.Errorf("unable to get scan results for image %s: %s", container.Image.Sha, err.Error())
			return nil, err
		}
		if imageScan == nil {
			return nil, nil
		}
		policyViolationCount += imageScan.PolicyViolations
		vulnerabilityCount += imageScan.Vulnerabilities
		imageScanOverallStatus := imageScan.OverallStatus
		if imageScanOverallStatus != hub.PolicyStatusTypeNotInViolation {
			overallStatus = imageScanOverallStatus
		}
	}
	podScan := &ScanResults{
		OverallStatus:    overallStatus,
		PolicyViolations: policyViolationCount,
		Vulnerabilities:  vulnerabilityCount}
	return podScan, nil
}

// ScanResultsForImage .....
func (model *Model) ScanResultsForImage(sha DockerImageSha) (*ScanResults, error) {
	imageInfo, ok := model.Images[sha]
	if !ok {
		return nil, fmt.Errorf("could not find image of sha %s in cache", sha)
	}

	if imageInfo.Layers == nil {
		return nil, nil
	}

	overallStatus := hub.PolicyStatusTypeNotInViolation
	policyViolationCount := 0
	vulnerabilityCount := 0
	for layer := range imageInfo.Layers {
		layerScan, err := model.ScanResultsForLayer(layer)
		if err != nil {
			log.Errorf("unable to get scan results for layer %s: %s", layer, err.Error())
			return nil, err
		}
		if layerScan == nil {
			return nil, nil
		}
		policyViolationCount += layerScan.PolicyViolations
		vulnerabilityCount += layerScan.Vulnerabilities
		imageScanOverallStatus := layerScan.OverallStatus
		if imageScanOverallStatus != hub.PolicyStatusTypeNotInViolation {
			overallStatus = imageScanOverallStatus
		}
	}

	scan := &ScanResults{
		OverallStatus:    overallStatus,
		PolicyViolations: policyViolationCount,
		Vulnerabilities:  vulnerabilityCount}
	return scan, nil
}

// ScanResultsForLayer .....
func (model *Model) ScanResultsForLayer(sha string) (*ScanResults, error) {
	layerInfo, ok := model.Layers[sha]
	if !ok {
		return nil, fmt.Errorf("could not find layer of sha %s", sha)
	}

	if layerInfo.ScanStatus != ScanStatusComplete {
		return nil, nil
	}
	if layerInfo.ScanResults == nil {
		return nil, fmt.Errorf("model inconsistency: could not find scan results for completed layer %s", sha)
	}

	scan := &ScanResults{
		OverallStatus:    layerInfo.ScanResults.OverallStatus(),
		PolicyViolations: layerInfo.ScanResults.PolicyViolationCount(),
		Vulnerabilities:  layerInfo.ScanResults.VulnerabilityCount()}
	return scan, nil
}

// Metrics .....
func (model *Model) Metrics() *Metrics {
	// number of images in each status
	statusCounts := make(map[ScanStatus]int)
	for _, layerInfo := range model.Layers {
		statusCounts[layerInfo.ScanStatus]++
	}

	// layers
	layerStatus := map[string]int{}
	layerPolicyViolations := map[int]int{}
	layerVulnerabilities := map[int]int{}
	imagesPerLayer := map[int]int{-8: 32 /* TODO implement by having a list of images in each layerInfo */}
	for sha, layerInfo := range model.Layers {
		if layerInfo.ScanStatus == ScanStatusComplete {
			scan := layerInfo.ScanResults
			if scan == nil {
				log.Errorf("found nil scan results for completed layer %s", sha)
				continue
			}
			layerStatus[scan.OverallStatus().String()]++
			layerPolicyViolations[scan.PolicyViolationCount()]++
			layerVulnerabilities[scan.VulnerabilityCount()]++
		} else {
			layerStatus["Unknown"]++
			layerPolicyViolations[-1]++
			layerVulnerabilities[-1]++
		}
	}

	// number of containers per pod (as a histogram, but not a prometheus histogram ???)
	containerCounts := make(map[int]int)
	for _, pod := range model.Pods {
		containerCounts[len(pod.Containers)]++
	}

	// number of times each image is referenced from a pod's container
	imageCounts := make(map[Image]int)
	for _, pod := range model.Pods {
		for _, cont := range pod.Containers {
			imageCounts[cont.Image]++
		}
	}
	imageCountHistogram := make(map[int]int)
	for _, count := range imageCounts {
		imageCountHistogram[count]++
	}

	podStatus := map[string]int{}
	podPolicyViolations := map[int]int{}
	podVulnerabilities := map[int]int{}
	for podName := range model.Pods {
		// TODO lots of duplicated work, because we recalculate images and layers over
		// and over -- can we build a cache?
		podScan, err := model.ScanResultsForPod(podName)
		if err != nil {
			log.Errorf("unable to get scan results for pod %s: %s", podName, err.Error())
			continue
		}
		if podScan != nil {
			podStatus[podScan.OverallStatus.String()]++
			podPolicyViolations[podScan.PolicyViolations]++
			podVulnerabilities[podScan.Vulnerabilities]++
		} else {
			podStatus["Unknown"]++
			podPolicyViolations[-1]++
			podVulnerabilities[-1]++
		}
	}

	imageStatus := map[string]int{}
	imagePolicyViolations := map[int]int{}
	imageVulnerabilities := map[int]int{}
	layersPerImage := map[int]int{}
	for sha, imageInfo := range model.Images {
		if imageInfo.Layers != nil {
			layersPerImage[len(imageInfo.Layers)]++
		} else {
			layersPerImage[-1]++
		}
		scan, err := model.ScanResultsForImage(sha)
		if err != nil {
			imageStatus["Error"]++
			log.Errorf("unable to get scan results for image %s: %s", sha, err.Error())
			continue
		}
		if scan != nil {
			imageStatus[scan.OverallStatus.String()]++
			imagePolicyViolations[scan.PolicyViolations]++
			imageVulnerabilities[scan.Vulnerabilities]++
		} else {
			imageStatus["Unknown"]++
			imagePolicyViolations[-1]++
			imageVulnerabilities[-1]++
		}
	}

	// TODO
	// number of images without a pod pointing to them
	return &Metrics{
		ScanStatusCounts:      statusCounts,
		NumberOfImages:        len(model.Images),
		NumberOfPods:          len(model.Pods),
		ContainerCounts:       containerCounts,
		ImageCountHistogram:   imageCountHistogram,
		PodStatus:             podStatus,
		PodPolicyViolations:   podPolicyViolations,
		PodVulnerabilities:    podVulnerabilities,
		ImageStatus:           imageStatus,
		ImagePolicyViolations: imagePolicyViolations,
		ImageVulnerabilities:  imageVulnerabilities,
		LayerStatus:           layerStatus,
		LayerPolicyViolations: layerPolicyViolations,
		LayerVulnerabilities:  layerVulnerabilities,
		LayersPerImage:        layersPerImage,
		ImagesPerLayer:        imagesPerLayer,
	}
}

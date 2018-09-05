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
	log "github.com/sirupsen/logrus"
)

// GetMetrics .....
type GetMetrics struct {
	Done chan *Metrics
}

// NewGetMetrics ...
func NewGetMetrics() *GetMetrics {
	return &GetMetrics{Done: make(chan *Metrics)}
}

// Apply .....
func (g *GetMetrics) Apply(model *Model) {
	modelMetrics := metrics(model)
	go func() {
		g.Done <- modelMetrics
	}()
}

func metrics(model *Model) *Metrics {
	// number of images in each status
	statusCounts := make(map[ScanStatus]int)
	for _, imageResults := range model.Images {
		statusCounts[imageResults.ScanStatus]++
	}

	// number of containers per pod (as a histgram, but not a prometheus histogram ???)
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
		podScan, err := scanResultsForPod(model, podName)
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
	for sha, imageInfo := range model.Images {
		if imageInfo.ScanStatus == ScanStatusComplete {
			imageScan := imageInfo.ScanResults
			if imageScan == nil {
				log.Errorf("found nil scan results for completed image %s", sha)
				continue
			}
			imageStatus[imageScan.OverallStatus().String()]++
			imagePolicyViolations[imageScan.PolicyViolationCount()]++
			imageVulnerabilities[imageScan.VulnerabilityCount()]++
		} else {
			imageStatus["Unknown"]++
			imagePolicyViolations[-1]++
			imageVulnerabilities[-1]++
		}
	}

	return &Metrics{
		ScanStatusCounts:      statusCounts,
		NumberOfImages:        len(model.Images),
		NumberOfPods:          len(model.Pods),
		ContainerCounts:       containerCounts,
		ImageCountHistogram:   imageCountHistogram,
		PodStatus:             podStatus,
		ImageStatus:           imageStatus,
		PodPolicyViolations:   podPolicyViolations,
		ImagePolicyViolations: imagePolicyViolations,
		PodVulnerabilities:    podVulnerabilities,
		ImageVulnerabilities:  imageVulnerabilities,
	}
}

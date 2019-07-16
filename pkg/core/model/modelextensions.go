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

	"github.com/blackducksoftware/perceptor/pkg/api" // TODO I hate how this package depends on the api package
	"github.com/blackducksoftware/perceptor/pkg/hub"
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
)

func scanResultsForPod(model *Model, podName string) (*Scan, error) {
	pod, ok := model.Pods[podName]
	if !ok {
		return nil, fmt.Errorf("could not find pod of name %s in cache", podName)
	}

	overallStatus := hub.PolicyStatusTypeNotInViolation
	policyViolationCount := 0
	vulnerabilityCount := 0
	for _, container := range pod.Containers {
		imageScan, err := scanResultsForImage(model, container.Image.Sha)
		if err != nil {
			return nil, errors.Annotatef(err, "unable to get scan results for image %s", container.Image.Sha)
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
	podScan := &Scan{
		OverallStatus:    overallStatus,
		PolicyViolations: policyViolationCount,
		Vulnerabilities:  vulnerabilityCount}
	return podScan, nil
}

func scanResultsForImage(model *Model, sha DockerImageSha) (*Scan, error) {
	imageInfo, ok := model.Images[sha]
	if !ok {
		return nil, fmt.Errorf("could not find image of sha %s in cache", sha)
	}

	if imageInfo.ScanStatus != ScanStatusComplete {
		return nil, nil
	}
	if imageInfo.ScanResults == nil {
		return nil, fmt.Errorf("model inconsistency: could not find scan results for completed image %s", sha)
	}

	imageScan := &Scan{
		OverallStatus:    imageInfo.ScanResults.OverallStatus(),
		PolicyViolations: imageInfo.ScanResults.PolicyViolationCount(),
		Vulnerabilities:  imageInfo.ScanResults.VulnerabilityCount()}
	return imageScan, nil
}

func scanResults(model *Model) (api.ScanResults, error) {
	errors := []error{}
	// pods
	pods := []api.ScannedPod{}
	for podName, pod := range model.Pods {
		podScan, err := scanResultsForPod(model, podName)
		if err != nil {
			errors = append(errors, fmt.Errorf("unable to retrieve scan results for Pod %s: %s", podName, err.Error()))
			continue
		}
		if podScan == nil {
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
	for sha, imageInfo := range model.Images {
		if imageInfo.ScanStatus != ScanStatusComplete {
			continue
		}
		if imageInfo.ScanResults == nil {
			errors = append(errors, fmt.Errorf("model inconsistency: found ScanStatusComplete for image %s, but nil ScanResults (imageInfo %+v)", sha, imageInfo))
			continue
		}
		image := imageInfo.Image()
		apiImage := api.ScannedImage{
			Repository:       image.Repository,
			Tag:              image.Tag,
			Sha:              string(image.Sha),
			PolicyViolations: imageInfo.ScanResults.PolicyViolationCount(),
			Vulnerabilities:  imageInfo.ScanResults.VulnerabilityCount(),
			OverallStatus:    imageInfo.ScanResults.OverallStatus(),
			ComponentsURL:    imageInfo.ScanResults.ComponentsHref}
		images = append(images, apiImage)
	}

	return *api.NewScanResults(pods, images), combineErrors("scanResults", errors)
}

func coreContainerToAPIContainer(coreContainer Container) *api.Container {
	image := coreContainer.Image
	priority := image.Priority
	return &api.Container{
		Image: *api.NewImage(image.Repository, image.Tag, string(image.Sha), &priority, image.BlackDuckProjectName, image.BlackDuckProjectVersion),
		Name:  coreContainer.Name,
	}
}

func corePodToAPIPod(corePod Pod) *api.Pod {
	containers := []api.Container{}
	for _, coreContainer := range corePod.Containers {
		containers = append(containers, *coreContainerToAPIContainer(coreContainer))
	}
	return &api.Pod{
		Containers: containers,
		Name:       corePod.Name,
		Namespace:  corePod.Namespace,
		UID:        corePod.UID,
	}
}

func coreModelToAPIModel(model *Model) *api.CoreModel {
	// pods
	pods := map[string]*api.Pod{}
	for podName, pod := range model.Pods {
		pods[podName] = corePodToAPIPod(pod)
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
	// image transitions
	imageTransitions := make([]*api.ModelImageTransition, len(model.ImageTransitions))
	for ix, it := range model.ImageTransitions {
		errString := ""
		if it.Err != nil {
			errString = it.Err.Error()
		}
		imageTransitions[ix] = &api.ModelImageTransition{
			Sha:  string(it.Sha),
			From: it.From,
			To:   it.To.String(),
			Err:  errString,
			Time: it.Time.String(),
		}
	}
	// return value
	return &api.CoreModel{
		Pods:             pods,
		Images:           images,
		ImageScanQueue:   model.ImageScanQueue.Dump(),
		ImageTransitions: imageTransitions,
	}
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
			podStatus[podScan.OverallStatus]++
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
			imageStatus[imageScan.OverallStatus()]++
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

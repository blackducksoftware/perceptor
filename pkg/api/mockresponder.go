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

package api

import (
	"fmt"
	"math/rand"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// MockResponder .....
type MockResponder struct {
	Pods             map[string]Pod
	Images           map[string]ImageInfo
	NextImageCounter int
}

// NewMockResponder .....
func NewMockResponder() *MockResponder {
	return &MockResponder{
		Pods:             map[string]Pod{},
		Images:           map[string]ImageInfo{},
		NextImageCounter: 0,
	}
}

// ImageInfo .....
type ImageInfo struct {
	Image            Image
	PolicyViolations int
	Vulnerabilities  int
	OverallStatus    string
	ComponentsURL    string
}

// GetMetrics .....
func (mr *MockResponder) GetMetrics(w http.ResponseWriter, r *http.Request) {
	// TODO
}

// GetModel .....
func (mr *MockResponder) GetModel() Model {
	// TODO
	return Model{}
}

// perceiver

// AddPod .....
func (mr *MockResponder) AddPod(pod Pod) {
	log.Infof("add pod: %+v", pod)
	qualifiedName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	_, ok := mr.Pods[qualifiedName]
	if ok {
		return
	}

	mr.Pods[qualifiedName] = pod
	for _, cont := range pod.Containers {
		mr.AddImage(cont.Image)
	}
}

// UpdatePod .....
func (mr *MockResponder) UpdatePod(pod Pod) {
	log.Infof("update pod: %+v", pod)
	qualifiedName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	mr.Pods[qualifiedName] = pod
}

// DeletePod .....
func (mr *MockResponder) DeletePod(qualifiedName string) {
	log.Infof("delete pod: %s", qualifiedName)
	delete(mr.Pods, qualifiedName)
}

// GetScanResults .....
func (mr *MockResponder) GetScanResults() ScanResults {
	log.Info("get scan results")
	scannedPods := []ScannedPod{}
	scannedImages := []ScannedImage{}
	for _, pod := range mr.Pods {
		scannedPods = append(scannedPods, ScannedPod{
			Name:             pod.Name,
			Namespace:        pod.Namespace,
			OverallStatus:    "",
			PolicyViolations: 0,
			Vulnerabilities:  0})
	}
	for _, imageInfo := range mr.Images {
		scannedImages = append(scannedImages, ScannedImage{
			Repository:       imageInfo.Image.Repository,
			Tag:              imageInfo.Image.Tag,
			ComponentsURL:    imageInfo.ComponentsURL,
			OverallStatus:    imageInfo.OverallStatus,
			PolicyViolations: imageInfo.PolicyViolations,
			Sha:              imageInfo.Image.Sha,
			Vulnerabilities:  imageInfo.Vulnerabilities})
	}
	return ScanResults{
		Pods:   scannedPods,
		Images: scannedImages,
	}
}

// AddImage .....
func (mr *MockResponder) AddImage(image Image) {
	_, ok := mr.Images[image.Sha]
	if ok {
		return
	}

	log.Infof("add image: %+v", image)
	policyViolations := rand.Intn(3)
	vulnerabilities := rand.Intn(3)
	overallStatus := "NOT_IN_VIOLATION"
	if (policyViolations + vulnerabilities) > 0 {
		overallStatus = "IN_VIOLATION"
	}
	// TODO have the "scan" take some non-zero amount of time?
	// TODO have some "scans" fail?
	mr.Images[image.Sha] = ImageInfo{
		Image:            image,
		ComponentsURL:    fmt.Sprintf("https://something-hub/%s", image.Sha),
		OverallStatus:    overallStatus,
		PolicyViolations: policyViolations,
		Vulnerabilities:  vulnerabilities}
}

// UpdateAllPods .....
func (mr *MockResponder) UpdateAllPods(allPods AllPods) {
	log.Infof("update all pods: %+v", allPods)
	mr.Pods = map[string]Pod{}
	for _, pod := range allPods.Pods {
		mr.AddPod(pod)
	}
}

// UpdateAllImages .....
func (mr *MockResponder) UpdateAllImages(allImages AllImages) {
	log.Infof("update all images: %+v", allImages)
	mr.Images = map[string]ImageInfo{}
	for _, image := range allImages.Images {
		mr.AddImage(image)
	}
}

// scanner

// GetNextImage .....
func (mr *MockResponder) GetNextImage() NextImage {
	mr.NextImageCounter++
	imageSpec := ImageSpec{
		HubProjectName:        fmt.Sprintf("mock-perceptor-%d", mr.NextImageCounter),
		HubProjectVersionName: fmt.Sprintf("mock-perceptor-project-version-%d", mr.NextImageCounter),
		HubScanName:           fmt.Sprintf("mock-perceptor-scan-name-%d", mr.NextImageCounter),
		Repository:            "abc/def/ghi",
		Sha:                   "123abc456def"}
	return NextImage{ImageSpec: &imageSpec}
}

// PostFinishScan .....
func (mr *MockResponder) PostFinishScan(job FinishedScanClientJob) {
	log.Infof("finished scan job: %+v", job)
}

// internal use

// PostConfig .....
func (mr *MockResponder) PostConfig(config *PostConfig) {
	// TODO
}

// PostCommand ...
func (mr *MockResponder) PostCommand(command *PostCommand) {
	// TODO
}

// errors

// NotFound .....
func (mr *MockResponder) NotFound(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

// Error .....
func (mr *MockResponder) Error(w http.ResponseWriter, r *http.Request, err error, statusCode int) {
	http.Error(w, err.Error(), statusCode)
}

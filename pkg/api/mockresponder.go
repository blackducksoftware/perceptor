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
	Pods             map[string]*Pod
	Images           map[string]ImageInfo
	NextImageCounter int
}

// NewMockResponder .....
func NewMockResponder() *MockResponder {
	return &MockResponder{
		Pods:             map[string]*Pod{},
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

// GetModel .....
func (mr *MockResponder) GetModel() (*Model, error) {
	// images := map[string]*ModelImageInfo{}
	// for key, image := range mr.Images {
	// 	scanResults := map[string]interface{}{
	// 		"PolicyStatus": "NOT_IN_VIOLATION",
	// 	}
	// 	// &hub.ScanResults{
	// 	// PolicyStatus: hub.PolicyStatus{
	// 	// 	OverallStatus: hub.PolicyStatusTypeNotInViolation,
	// 	// 	UpdatedAt:     time.Now().String(),
	// 	// 	ComponentVersionStatusCounts: map[hub.PolicyStatusType]int{
	// 	// 		hub.PolicyStatusTypeNotInViolation: 3,
	// 	// 	},
	// 	// },
	// 	// RiskProfile: hub.RiskProfile{BomLastUpdatedAt: time.Now().String()}}
	// 	images[key] = &ModelImageInfo{
	// 		ImageSha: key,
	// 		RepoTags: []*ModelRepoTag{
	// 			{Repository: image.Image.Repository, Tag: image.Image.Tag},
	// 		},
	// 		ScanResults: scanResults}
	// }
	// return Model{
	// 	Images: images,
	// 	Pods:   mr.Pods,
	// }
	return &Model{}, nil
}

// perceiver

// AddPod .....
func (mr *MockResponder) AddPod(pod Pod) error {
	log.Infof("add pod: %+v", pod)
	qualifiedName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	_, ok := mr.Pods[qualifiedName]
	if ok {
		return nil
	}

	mr.Pods[qualifiedName] = &pod
	for _, cont := range pod.Containers {
		err := mr.AddImage(cont.Image)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdatePod .....
func (mr *MockResponder) UpdatePod(pod Pod) error {
	log.Infof("update pod: %+v", pod)
	qualifiedName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	mr.Pods[qualifiedName] = &pod
	return nil
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
func (mr *MockResponder) AddImage(image Image) error {
	_, ok := mr.Images[image.Sha]
	if ok {
		return nil
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
	return nil
}

// UpdateAllPods .....
func (mr *MockResponder) UpdateAllPods(allPods AllPods) error {
	log.Infof("update all pods: %+v", allPods)
	mr.Pods = map[string]*Pod{}
	for _, pod := range allPods.Pods {
		mr.AddPod(pod)
	}
	return nil
}

// UpdateAllImages .....
func (mr *MockResponder) UpdateAllImages(allImages AllImages) error {
	log.Infof("update all images: %+v", allImages)
	mr.Images = map[string]ImageInfo{}
	for _, image := range allImages.Images {
		mr.AddImage(image)
	}
	return nil
}

// scanner

// GetNextImage .....
func (mr *MockResponder) GetNextImage() NextImage {
	mr.NextImageCounter++
	imageSpec := ImageSpec{
		BlackDuckProjectName:        fmt.Sprintf("mock-perceptor-%d", mr.NextImageCounter),
		BlackDuckProjectVersionName: fmt.Sprintf("mock-perceptor-project-version-%d", mr.NextImageCounter),
		BlackDuckScanName:           fmt.Sprintf("mock-perceptor-scan-name-%d", mr.NextImageCounter),
		Repository:                  "abc/def/ghi",
		Tag:                         "latest",
		Sha:                         "123abc456def"}
	return NextImage{ImageSpec: &imageSpec}
}

// PostFinishScan .....
func (mr *MockResponder) PostFinishScan(job FinishedScanClientJob) error {
	log.Infof("finished scan job: %+v", job)
	return nil
}

// internal use

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

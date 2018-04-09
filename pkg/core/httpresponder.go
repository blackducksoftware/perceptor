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
	"net/http"
	"sync"

	api "github.com/blackducksoftware/perceptor/pkg/api"
	a "github.com/blackducksoftware/perceptor/pkg/core/actions"
	model "github.com/blackducksoftware/perceptor/pkg/core/model"
	log "github.com/sirupsen/logrus"
)

// HTTPResponder ...
type HTTPResponder struct {
	AddPodChannel                 chan model.Pod
	UpdatePodChannel              chan model.Pod
	DeletePodChannel              chan string
	AddImageChannel               chan model.Image
	AllPodsChannel                chan []model.Pod
	AllImagesChannel              chan []model.Image
	PostNextImageChannel          chan func(*model.Image)
	PostFinishScanJobChannel      chan *a.FinishScanClient
	SetConcurrentScanLimitChannel chan int
	GetModelChannel               chan func(api.Model)
	GetScanResultsChannel         chan func(scanResults api.ScanResults)
}

func NewHTTPResponder() *HTTPResponder {
	return &HTTPResponder{
		AddPodChannel:                 make(chan model.Pod),
		UpdatePodChannel:              make(chan model.Pod),
		DeletePodChannel:              make(chan string),
		AddImageChannel:               make(chan model.Image),
		AllPodsChannel:                make(chan []model.Pod),
		AllImagesChannel:              make(chan []model.Image),
		PostNextImageChannel:          make(chan func(*model.Image)),
		PostFinishScanJobChannel:      make(chan *a.FinishScanClient),
		SetConcurrentScanLimitChannel: make(chan int),
		GetModelChannel:               make(chan func(api.Model)),
		GetScanResultsChannel:         make(chan func(api.ScanResults))}
}

func (hr *HTTPResponder) GetModel() api.Model {
	var wg sync.WaitGroup
	wg.Add(1)
	var model api.Model
	hr.GetModelChannel <- func(tempModel api.Model) {
		model = tempModel
		wg.Done()
	}
	wg.Wait()
	return model
}

func (hr *HTTPResponder) AddPod(apiPod api.Pod) {
	recordAddPod()
	pod := *model.ApiPodToCorePod(apiPod)
	hr.AddPodChannel <- pod
	log.Debugf("handled add pod %s -- %s", pod.UID, pod.QualifiedName())
}

func (hr *HTTPResponder) DeletePod(qualifiedName string) {
	recordDeletePod()
	hr.DeletePodChannel <- qualifiedName
	log.Debugf("handled delete pod %s", qualifiedName)
}

func (hr *HTTPResponder) UpdatePod(apiPod api.Pod) {
	recordUpdatePod()
	pod := *model.ApiPodToCorePod(apiPod)
	hr.UpdatePodChannel <- pod
	log.Debugf("handled update pod %s -- %s", pod.UID, pod.QualifiedName())
}

func (hr *HTTPResponder) AddImage(apiImage api.Image) {
	recordAddImage()
	image := *model.ApiImageToCoreImage(apiImage)
	hr.AddImageChannel <- image
	log.Debugf("handled add image %s", image.PullSpec())
}

func (hr *HTTPResponder) UpdateAllPods(allPods api.AllPods) {
	recordAllPods()
	pods := []model.Pod{}
	for _, apiPod := range allPods.Pods {
		pods = append(pods, *model.ApiPodToCorePod(apiPod))
	}
	hr.AllPodsChannel <- pods
	log.Debugf("handled update all pods -- %d pods", len(allPods.Pods))
}

func (hr *HTTPResponder) UpdateAllImages(allImages api.AllImages) {
	recordAllImages()
	images := []model.Image{}
	for _, apiImage := range allImages.Images {
		images = append(images, *model.ApiImageToCoreImage(apiImage))
	}
	hr.AllImagesChannel <- images
	log.Debugf("handled update all images -- %d images", len(allImages.Images))
}

// GetScanResults returns results for:
//  - all images that have a scan status of complete
//  - all pods for which all their images have a scan status of complete
func (hr *HTTPResponder) GetScanResults() api.ScanResults {
	recordGetScanResults()
	var wg sync.WaitGroup
	wg.Add(1)
	var scanResults api.ScanResults
	hr.GetScanResultsChannel <- func(results api.ScanResults) {
		wg.Done()
		scanResults = results
	}
	wg.Wait()
	return scanResults
}

func (hr *HTTPResponder) GetNextImage() api.NextImage {
	recordGetNextImage()
	var wg sync.WaitGroup
	var nextImage api.NextImage
	wg.Add(1)
	hr.PostNextImageChannel <- func(image *model.Image) {
		imageString := "null"
		var imageSpec *api.ImageSpec
		if image != nil {
			imageString = image.HumanReadableName()
			imageSpec = api.NewImageSpec(
				image.Name,
				image.PullSpec(),
				string(image.Sha),
				image.HubProjectName(),
				image.HubProjectVersionName(),
				image.HubScanName())
		}
		nextImage = *api.NewNextImage(imageSpec)
		log.Debugf("handled GET next image -- %s", imageString)
		wg.Done()
	}
	wg.Wait()
	return nextImage
}

func (hr *HTTPResponder) PostFinishScan(job api.FinishedScanClientJob) {
	recordPostFinishedScan()
	var err error
	if job.Err == "" {
		err = nil
	} else {
		err = fmt.Errorf(job.Err)
	}
	image := model.NewImage(job.ImageSpec.ImageName, model.DockerImageSha(job.ImageSpec.Sha))
	hr.PostFinishScanJobChannel <- &a.FinishScanClient{Image: image, Err: err}
	log.Debugf("handled finished scan job -- %v", job)
}

// internal use

func (hr *HTTPResponder) SetConcurrentScanLimit(limit api.SetConcurrentScanLimit) {
	hr.SetConcurrentScanLimitChannel <- limit.Limit
	log.Debugf("handled set concurrent scan limit -- %d", limit)
}

// errors

func (hr *HTTPResponder) NotFound(w http.ResponseWriter, r *http.Request) {
	log.Errorf("HTTPResponder not found from request %+v", r)
	recordHTTPNotFound(r)
	http.NotFound(w, r)
}

func (hr *HTTPResponder) Error(w http.ResponseWriter, r *http.Request, err error, statusCode int) {
	log.Errorf("HTTPResponder error %s with code %d from request %+v", err.Error(), statusCode, r)
	recordHTTPError(r, err, statusCode)
	http.Error(w, err.Error(), statusCode)
}

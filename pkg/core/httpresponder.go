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
	AddPodChannel              chan *a.AddPod
	UpdatePodChannel           chan *a.UpdatePod
	DeletePodChannel           chan *a.DeletePod
	AddImageChannel            chan *a.AddImage
	AllPodsChannel             chan *a.AllPods
	AllImagesChannel           chan *a.AllImages
	PostNextImageChannel       chan func(*model.Image)
	PostFinishScanJobChannel   chan *a.FinishScanClient
	PostConfigChannel          chan *api.PostConfig
	ResetCircuitBreakerChannel chan bool
	GetModelChannel            chan *a.GetModel
	GetScanResultsChannel      chan *a.GetScanResults
}

// NewHTTPResponder .....
func NewHTTPResponder() *HTTPResponder {
	return &HTTPResponder{
		AddPodChannel:              make(chan *a.AddPod),
		UpdatePodChannel:           make(chan *a.UpdatePod),
		DeletePodChannel:           make(chan *a.DeletePod),
		AddImageChannel:            make(chan *a.AddImage),
		AllPodsChannel:             make(chan *a.AllPods),
		AllImagesChannel:           make(chan *a.AllImages),
		PostNextImageChannel:       make(chan func(*model.Image)),
		PostFinishScanJobChannel:   make(chan *a.FinishScanClient),
		PostConfigChannel:          make(chan *api.PostConfig),
		ResetCircuitBreakerChannel: make(chan bool),
		GetModelChannel:            make(chan *a.GetModel),
		GetScanResultsChannel:      make(chan *a.GetScanResults)}
}

// GetModel .....
func (hr *HTTPResponder) GetModel() api.Model {
	get := a.NewGetModel()
	hr.GetModelChannel <- get
	return *<-get.Done
}

// AddPod .....
func (hr *HTTPResponder) AddPod(apiPod api.Pod) error {
	recordAddPod()
	pod, err := APIPodToCorePod(apiPod)
	if err != nil {
		return err
	}
	action := &a.AddPod{*pod}
	hr.AddPodChannel <- action
	log.Debugf("handled add pod %s -- %s", pod.UID, pod.QualifiedName())
	return nil
}

// DeletePod .....
func (hr *HTTPResponder) DeletePod(qualifiedName string) {
	recordDeletePod()
	hr.DeletePodChannel <- &a.DeletePod{qualifiedName}
	log.Debugf("handled delete pod %s", qualifiedName)
}

// UpdatePod .....
func (hr *HTTPResponder) UpdatePod(apiPod api.Pod) error {
	recordUpdatePod()
	pod, err := APIPodToCorePod(apiPod)
	if err != nil {
		return err
	}
	hr.UpdatePodChannel <- &a.UpdatePod{*pod}
	log.Debugf("handled update pod %s -- %s", pod.UID, pod.QualifiedName())
	return nil
}

// AddImage .....
func (hr *HTTPResponder) AddImage(apiImage api.Image) error {
	recordAddImage()
	image, err := APIImageToCoreImage(apiImage)
	if err != nil {
		return err
	}
	hr.AddImageChannel <- &a.AddImage{*image}
	log.Debugf("handled add image %s", image.PullSpec())
	return nil
}

// UpdateAllPods .....
func (hr *HTTPResponder) UpdateAllPods(allPods api.AllPods) error {
	recordAllPods()
	pods := []model.Pod{}
	for _, apiPod := range allPods.Pods {
		pod, err := APIPodToCorePod(apiPod)
		if err != nil {
			return err
		}
		pods = append(pods, *pod)
	}
	hr.AllPodsChannel <- &a.AllPods{pods}
	log.Debugf("handled update all pods -- %d pods", len(allPods.Pods))
	return nil
}

// UpdateAllImages .....
func (hr *HTTPResponder) UpdateAllImages(allImages api.AllImages) error {
	recordAllImages()
	images := []model.Image{}
	for _, apiImage := range allImages.Images {
		image, err := APIImageToCoreImage(apiImage)
		if err != nil {
			return err
		}
		images = append(images, *image)
	}
	hr.AllImagesChannel <- &a.AllImages{images}
	log.Debugf("handled update all images -- %d images", len(allImages.Images))
	return nil
}

// GetScanResults returns results for:
//  - all images that have a scan status of complete
//  - all pods for which all their images have a scan status of complete
func (hr *HTTPResponder) GetScanResults() api.ScanResults {
	recordGetScanResults()
	get := a.NewGetScanResults()
	hr.GetScanResultsChannel <- get
	return <-get.Done
}

// GetNextImage .....
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

// PostFinishScan .....
func (hr *HTTPResponder) PostFinishScan(job api.FinishedScanClientJob) error {
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
	return nil
}

// internal use

// PostConfig .....
func (hr *HTTPResponder) PostConfig(config *api.PostConfig) {
	hr.PostConfigChannel <- config
	log.Debugf("handled post config -- %+v", config)
}

// PostCommand .....
func (hr *HTTPResponder) PostCommand(command *api.PostCommand) {
	if command.ResetCircuitBreaker != nil {
		hr.ResetCircuitBreakerChannel <- true
	}
	log.Debugf("handled post command -- %+v", command)
}

// errors

// NotFound .....
func (hr *HTTPResponder) NotFound(w http.ResponseWriter, r *http.Request) {
	log.Errorf("HTTPResponder not found from request %+v", r)
	recordHTTPNotFound(r)
	http.NotFound(w, r)
}

// Error .....
func (hr *HTTPResponder) Error(w http.ResponseWriter, r *http.Request, err error, statusCode int) {
	log.Errorf("HTTPResponder error %s with code %d from request %+v", err.Error(), statusCode, r)
	recordHTTPError(r, err, statusCode)
	http.Error(w, err.Error(), statusCode)
}

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
	"net/http"
	"sync"

	api "github.com/blackducksoftware/perceptor/pkg/api"
	log "github.com/sirupsen/logrus"
)

// HTTPResponder ...
type HTTPResponder struct {
	addPod                 chan Pod
	updatePod              chan Pod
	deletePod              chan string
	addImage               chan Image
	allPods                chan []Pod
	allImages              chan []Image
	postNextImage          chan func(*Image)
	postFinishScanJob      chan api.FinishedScanClientJob
	setConcurrentScanLimit chan int
	getModel               chan func(json string)
	getScanResults         chan func(scanResults api.ScanResults)
}

func NewHTTPResponder() *HTTPResponder {
	return &HTTPResponder{
		addPod:                 make(chan Pod),
		updatePod:              make(chan Pod),
		deletePod:              make(chan string),
		addImage:               make(chan Image),
		allPods:                make(chan []Pod),
		allImages:              make(chan []Image),
		postNextImage:          make(chan func(*Image)),
		postFinishScanJob:      make(chan api.FinishedScanClientJob),
		setConcurrentScanLimit: make(chan int),
		getModel:               make(chan func(json string))}
}

func (hr *HTTPResponder) GetMetrics(w http.ResponseWriter, r *http.Request) {
	metricsHandler.httpHandler.ServeHTTP(w, r)
}

func (hr *HTTPResponder) GetModel() string {
	var wg sync.WaitGroup
	wg.Add(1)
	var modelString string
	hr.getModel <- func(json string) {
		modelString = json
		wg.Done()
	}
	wg.Wait()
	return modelString
}

func (hr *HTTPResponder) AddPod(apiPod api.Pod) {
	pod := *newPod(apiPod)
	metricsHandler.addPod(pod)
	hr.addPod <- pod
	log.Infof("handled add pod %s -- %s", pod.UID, pod.QualifiedName())
}

func (hr *HTTPResponder) DeletePod(qualifiedName string) {
	metricsHandler.deletePod(qualifiedName)
	hr.deletePod <- qualifiedName
	log.Infof("handled delete pod %s", qualifiedName)
}

func (hr *HTTPResponder) UpdatePod(apiPod api.Pod) {
	pod := *newPod(apiPod)
	metricsHandler.updatePod(pod)
	hr.updatePod <- pod
	log.Infof("handled update pod %s -- %s", pod.UID, pod.QualifiedName())
}

func (hr *HTTPResponder) AddImage(apiImage api.Image) {
	image := *newImage(apiImage)
	metricsHandler.addImage(image)
	hr.addImage <- image
	log.Infof("handled add image %s", image.HumanReadableName())
}

func (hr *HTTPResponder) UpdateAllPods(allPods api.AllPods) {
	pods := []Pod{}
	for _, apiPod := range allPods.Pods {
		pods = append(pods, *newPod(apiPod))
	}
	metricsHandler.allPods(pods)
	hr.allPods <- pods
	log.Infof("handled update all pods -- %d pods", len(allPods.Pods))
}

func (hr *HTTPResponder) UpdateAllImages(allImages api.AllImages) {
	images := []Image{}
	for _, apiImage := range allImages.Images {
		images = append(images, *newImage(apiImage))
	}
	metricsHandler.allImages(images)
	hr.allImages <- images
	log.Infof("handled update all images -- %d images", len(allImages.Images))
}

// GetScanResults returns results for:
//  - all images that have a scan status of complete
//  - all pods for which all their images have a scan status of complete
func (hr *HTTPResponder) GetScanResults() api.ScanResults {
	metricsHandler.getScanResults()
	var wg sync.WaitGroup
	wg.Add(1)
	var scanResults api.ScanResults
	hr.getScanResults <- func(results api.ScanResults) {
		wg.Done()
		scanResults = results
	}
	wg.Wait()
	return scanResults
}

func (hr *HTTPResponder) GetNextImage() api.NextImage {
	metricsHandler.getNextImage()
	var wg sync.WaitGroup
	var nextImage api.NextImage
	wg.Add(1)
	hr.postNextImage <- func(image *Image) {
		imageString := "null"
		var imageSpec *api.ImageSpec
		if image != nil {
			imageString = image.HumanReadableName()
			imageSpec = api.NewImageSpec(
				image.PullSpec(),
				string(image.Sha),
				image.HubProjectName(),
				image.HubProjectVersionName(),
				image.HubScanName())
		}
		nextImage = *api.NewNextImage(imageSpec)
		log.Infof("handled GET next image -- %s", imageString)
		wg.Done()
	}
	wg.Wait()
	return nextImage
}

func (hr *HTTPResponder) PostFinishScan(job api.FinishedScanClientJob) {
	metricsHandler.postFinishedScan()
	hr.postFinishScanJob <- job
	log.Infof("handled finished scan job -- %v", job)
}

// internal use

func (hr *HTTPResponder) SetConcurrentScanLimit(limit api.SetConcurrentScanLimit) {
	hr.setConcurrentScanLimit <- limit.Limit
	log.Infof("handled set concurrent scan limit -- %d", limit)
}

// errors

func (hr *HTTPResponder) NotFound(w http.ResponseWriter, r *http.Request) {
	metricsHandler.httpNotFound(r)
	http.NotFound(w, r)
}

func (hr *HTTPResponder) Error(w http.ResponseWriter, r *http.Request, err error, statusCode int) {
	metricsHandler.httpError(r, err)
	http.Error(w, err.Error(), statusCode)
}

/*
Copyright (C) 2018 Black Duck Software, Inc.

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
	"encoding/json"
	"fmt"
	"net/http"

	api "github.com/blackducksoftware/perceptor/pkg/api"
	log "github.com/sirupsen/logrus"
)

// HTTPResponder ...
type HTTPResponder struct {
	model             Model
	metricsHandler    *metrics
	addPod            chan Pod
	updatePod         chan Pod
	deletePod         chan string
	addImage          chan Image
	allPods           chan []Pod
	postNextImage     chan func(*Image)
	postFinishScanJob chan api.FinishedScanClientJob
}

func NewHTTPResponder(model <-chan Model, metricsHandler *metrics) *HTTPResponder {
	hr := HTTPResponder{
		metricsHandler:    metricsHandler,
		addPod:            make(chan Pod),
		updatePod:         make(chan Pod),
		deletePod:         make(chan string),
		addImage:          make(chan Image),
		allPods:           make(chan []Pod),
		postNextImage:     make(chan func(*Image)),
		postFinishScanJob: make(chan api.FinishedScanClientJob)}
	go func() {
		for {
			select {
			case m := <-model:
				hr.model = m
			}
		}
	}()
	return &hr
}

func (hr *HTTPResponder) GetMetrics(w http.ResponseWriter, r *http.Request) {
	hr.metricsHandler.httpHandler.ServeHTTP(w, r)
}

func (hr *HTTPResponder) GetModel() string {
	jsonBytes, err := json.Marshal(hr.model)
	if err != nil {
		return fmt.Sprintf("unable to serialize model: %s", err.Error())
	}
	return string(jsonBytes)
}

func (hr *HTTPResponder) AddPod(apiPod api.Pod) {
	pod := *newPod(apiPod)
	hr.metricsHandler.addPod(pod)
	hr.addPod <- pod
	log.Infof("handled add pod %s -- %s", pod.UID, pod.QualifiedName())
}

func (hr *HTTPResponder) DeletePod(qualifiedName string) {
	hr.metricsHandler.deletePod(qualifiedName)
	hr.deletePod <- qualifiedName
	log.Infof("handled delete pod %s", qualifiedName)
}

func (hr *HTTPResponder) UpdatePod(apiPod api.Pod) {
	pod := *newPod(apiPod)
	hr.metricsHandler.updatePod(pod)
	hr.updatePod <- pod
	log.Infof("handled update pod %s -- %s", pod.UID, pod.QualifiedName())
}

func (hr *HTTPResponder) AddImage(apiImage api.Image) {
	image := *newImage(apiImage)
	hr.metricsHandler.addImage(image)
	hr.addImage <- image
	log.Infof("handled add image %s", image.HumanReadableName())
}

func (hr *HTTPResponder) UpdateAllPods(allPods api.AllPods) {
	pods := []Pod{}
	for _, apiPod := range allPods.Pods {
		pods = append(pods, *newPod(apiPod))
	}
	hr.metricsHandler.allPods(pods)
	hr.allPods <- pods
	log.Infof("handled update all pods -- %d pods", len(allPods.Pods))
}

func (hr *HTTPResponder) GetScanResults() api.ScanResults {
	hr.metricsHandler.getScanResults()
	pods := []api.ScannedPod{}
	images := []api.ScannedImage{}
	for podName, pod := range hr.model.Pods {
		policyViolationCount, vulnerabilityCount, overallStatus, err := hr.model.scanResults(podName)
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
	for _, imageInfo := range hr.model.Images {
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
	return *api.NewScanResults(pods, images)
}

func (hr *HTTPResponder) GetNextImage(continuation func(nextImage api.NextImage)) {
	hr.metricsHandler.getNextImage()
	hr.postNextImage <- func(image *Image) {
		imageString := "null"
		var imageSpec *api.ImageSpec
		if image != nil {
			sha := string(image.Sha)
			imageString = image.HumanReadableName()
			imageSpec = api.NewImageSpec(image.PullSpec(), sha, sha, sha, sha)
		}
		nextImage := *api.NewNextImage(imageSpec)
		continuation(nextImage)
		log.Infof("handled GET next image -- %s", imageString)
	}
}

func (hr *HTTPResponder) PostFinishScan(job api.FinishedScanClientJob) {
	hr.metricsHandler.postFinishedScan()
	hr.postFinishScanJob <- job
	log.Infof("handled finished scan job -- %v", job)
}

// errors

func (hr *HTTPResponder) NotFound(w http.ResponseWriter, r *http.Request) {
	hr.metricsHandler.httpNotFound(r)
	http.NotFound(w, r)
}

func (hr *HTTPResponder) Error(w http.ResponseWriter, r *http.Request, err error, statusCode int) {
	hr.metricsHandler.httpError(r, err)
	http.Error(w, err.Error(), statusCode)
}

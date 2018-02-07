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

	api "bitbucket.org/bdsengineering/perceptor/pkg/api"
	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
	"github.com/prometheus/common/log"
)

// HTTPResponder ...
type HTTPResponder struct {
	model             Model
	metricsHandler    *metrics
	addPod            chan common.Pod
	updatePod         chan common.Pod
	deletePod         chan string
	addImage          chan common.Image
	allPods           chan []common.Pod
	postNextImage     chan func(*common.Image)
	postFinishScanJob chan api.FinishedScanClientJob
}

func NewHTTPResponder(model <-chan Model, metricsHandler *metrics) *HTTPResponder {
	hr := HTTPResponder{
		metricsHandler:    metricsHandler,
		addPod:            make(chan common.Pod),
		updatePod:         make(chan common.Pod),
		deletePod:         make(chan string),
		addImage:          make(chan common.Image),
		allPods:           make(chan []common.Pod),
		postNextImage:     make(chan func(*common.Image)),
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

func (hr *HTTPResponder) AddPod(pod common.Pod) {
	hr.metricsHandler.addPod(pod)
	hr.addPod <- pod
	log.Infof("handled add pod %s -- %s", pod.UID, pod.QualifiedName())
}

func (hr *HTTPResponder) DeletePod(qualifiedName string) {
	hr.metricsHandler.deletePod(qualifiedName)
	hr.deletePod <- qualifiedName
	log.Infof("handled delete pod %s", qualifiedName)
}

func (hr *HTTPResponder) UpdatePod(pod common.Pod) {
	hr.metricsHandler.updatePod(pod)
	hr.updatePod <- pod
	log.Infof("handled update pod %s -- %s", pod.UID, pod.QualifiedName())
}

func (hr *HTTPResponder) AddImage(image common.Image) {
	hr.metricsHandler.addImage(image)
	hr.addImage <- image
	log.Infof("handled add image %s", image.HumanReadableName())
}

func (hr *HTTPResponder) UpdateAllPods(allPods api.AllPods) {
	hr.metricsHandler.allPods(allPods)
	hr.allPods <- allPods.Pods
	log.Infof("handled update all pods -- %d pods", len(allPods.Pods))
}

func (hr *HTTPResponder) GetScanResults() api.ScanResults {
	hr.metricsHandler.getScanResults()
	scannerVersion := "TODO"
	hubServer := "TODO"
	pods := []api.Pod{}
	images := []api.Image{}
	for podName, pod := range hr.model.Pods {
		policyViolationCount, vulnerabilityCount, overallStatus, err := hr.model.scanResults(podName)
		if err != nil {
			log.Errorf("unable to retrieve scan results for Pod %s: %s", podName, err.Error())
			continue
		}
		pods = append(pods, api.Pod{
			Namespace:        pod.Namespace,
			Name:             pod.Name,
			PolicyViolations: policyViolationCount,
			Vulnerabilities:  vulnerabilityCount,
			OverallStatus:    overallStatus})
	}
	for image, imageResults := range hr.model.Images {
		scanID := image.HubScanName()
		projectVersionURL := "TODO"
		policyViolations := 0
		vulnerabilities := 0
		if imageResults.ScanResults != nil {
			policyViolations = imageResults.ScanResults.PolicyViolationCount()
			vulnerabilities = imageResults.ScanResults.VulnerabilityCount()
		}
		apiImage := api.Image{
			Name:              image.HumanReadableName(),
			Sha:               image.Sha,
			ScanID:            scanID,
			PolicyViolations:  policyViolations,
			Vulnerabilities:   vulnerabilities,
			ProjectVersionURL: projectVersionURL}
		images = append(images, apiImage)
	}
	return *api.NewScanResults(scannerVersion, hubServer, pods, images)
}

func (hr *HTTPResponder) GetNextImage(continuation func(nextImage api.NextImage)) {
	hr.metricsHandler.getNextImage()
	hr.postNextImage <- func(image *common.Image) {
		continuation(*api.NewNextImage(image))
		imageString := "null"
		if image != nil {
			imageString = image.HumanReadableName()
		}
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

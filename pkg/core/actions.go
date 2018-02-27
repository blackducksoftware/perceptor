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
	"encoding/json"

	"github.com/blackducksoftware/perceptor/pkg/api"
	log "github.com/sirupsen/logrus"
)

type action interface {
	apply(model *Model)
}

type addPod struct {
	pod Pod
}

func (a *addPod) apply(model *Model) {
	model.AddPod(a.pod)
}

type updatePod struct {
	pod Pod
}

func (u *updatePod) apply(model *Model) {
	model.AddPod(u.pod)
}

type deletePod struct {
	podName string
}

func (d *deletePod) apply(model *Model) {
	_, ok := model.Pods[d.podName]
	if !ok {
		log.Warnf("unable to delete pod %s, pod not found", d.podName)
		return
	}
	delete(model.Pods, d.podName)
}

type addImage struct {
	image Image
}

func (a *addImage) apply(model *Model) {
	model.AddImage(a.image)
}

type allPods struct {
	pods []Pod
}

func (a *allPods) apply(model *Model) {
	model.Pods = map[string]Pod{}
	for _, pod := range a.pods {
		model.AddPod(pod)
	}
}

type getNextImage struct {
	continuation func(image *Image)
}

func (g *getNextImage) apply(model *Model) {
	log.Infof("looking for next image to scan with concurrency limit of %d, and %d currently in progress", model.ConcurrentScanLimit, model.inProgressScanCount())
	image := model.getNextImageFromScanQueue()
	go g.continuation(image)
}

type finishScanClient struct {
	sha DockerImageSha
	err string
}

func (f *finishScanClient) apply(model *Model) {
	newModel := model
	log.Infof("finished scan client job action: error was empty? %t, %+v", f.err == "", f.sha)
	if f.err == "" {
		newModel.finishRunningScanClient(f.sha)
	} else {
		newModel.errorRunningScanClient(f.sha)
	}
}

type getNextImageForHubPolling struct {
	continuation func(image *Image)
}

func (g *getNextImageForHubPolling) apply(model *Model) {
	log.Infof("looking for next image to search for in hub")
	image := model.getNextImageFromHubCheckQueue()
	go g.continuation(image)
}

type hubCheckResults struct {
	scan HubImageScan
}

func (h *hubCheckResults) apply(model *Model) {
	scan := h.scan
	imageInfo, ok := model.Images[scan.Sha]
	if !ok {
		log.Warnf("expected to already have image %s, but did not", string(scan.Sha))
		return
	}

	imageInfo.ScanResults = scan.Scan

	//	log.Infof("completing image scan of image %s ? %t", image.ShaName(), scan.Scan.IsDone())
	if scan.Scan == nil {
		model.addImageToScanQueue(scan.Sha)
	} else if scan.Scan.IsDone() {
		imageInfo.setScanStatus(ScanStatusComplete)
	} else {
		// it could be in the scan client stage, in the hub stage ...
		// maybe perceptor crashed and just came back up
		// since we don't know, we have to put it into the scan queue
		model.addImageToScanQueue(scan.Sha)
	}
}

type hubScanResults struct {
	scan HubImageScan
}

func (h *hubScanResults) apply(model *Model) {
	scan := h.scan
	imageInfo, ok := model.Images[scan.Sha]
	if !ok {
		log.Warnf("expected to already have image %s, but did not", string(scan.Sha))
		return
	}

	imageInfo.ScanResults = scan.Scan

	//	log.Infof("completing image scan of image %s ? %t", image.ShaName(), scan.Scan.IsDone())
	if scan.Scan != nil && scan.Scan.IsDone() {
		imageInfo.setScanStatus(ScanStatusComplete)
	}
}

type requeueStalledScan struct {
	sha DockerImageSha
}

func (r *requeueStalledScan) apply(model *Model) {
	imageInfo, ok := model.Images[r.sha]
	if !ok {
		return
	}
	if imageInfo.ScanStatus != ScanStatusRunningScanClient {
		return
	}
	imageInfo.setScanStatus(ScanStatusError)
	model.addImageToScanQueue(r.sha)
}

type setConcurrentScanLimit struct {
	limit int
}

func (s *setConcurrentScanLimit) apply(model *Model) {
	limit := s.limit
	if limit < 0 {
		log.Errorf("cannot set concurrent scan limit to less than 0 (got %d)", limit)
		return
	}
	model.ConcurrentScanLimit = limit
}

type allImages struct {
	images []Image
}

func (a *allImages) apply(model *Model) {
	for _, image := range a.images {
		model.AddImage(image)
	}
}

type getModel struct {
	continuation func(json string)
}

func (g *getModel) apply(model *Model) {
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		jsonBytes = []byte{}
		log.Errorf("unable to serialize model: %s", err.Error())
	}
	go g.continuation(string(jsonBytes))
}

type getScanResults struct {
	continuation func(results api.ScanResults)
}

func (g *getScanResults) apply(model *Model) {
	scanResults := model.scanResults()
	go g.continuation(scanResults)
}

type getInProgressHubScans struct {
	continuation func(images []Image)
}

func (g *getInProgressHubScans) apply(model *Model) {
	scans := []Image{}
	for _, image := range model.inProgressHubScans() {
		scans = append(scans, image)
	}
	go g.continuation(scans)
}

type getInProgressScanClientScans struct {
	continuation func(imageInfos []*ImageInfo)
}

func (g *getInProgressScanClientScans) apply(model *Model) {
	imageInfos := []*ImageInfo{}
	for _, imageInfo := range model.inProgressScanClientScans() {
		// TODO could make a deep copy of imageInfo in case it is being
		// changed while we're looking at it
		imageInfos = append(imageInfos, imageInfo)
	}
	go g.continuation(imageInfos)
}

type getMetrics struct {
	continuation func(metrics *ModelMetrics)
}

func (g *getMetrics) apply(model *Model) {
	modelMetrics := model.metrics()
	go g.continuation(modelMetrics)
}

type debugGetModel struct {
	continuation func(model *Model)
}

func (d *debugGetModel) apply(model *Model) {
	go d.continuation(model)
}

type getCompletedScans struct {
	continuation func(images []*Image)
}

func (g *getCompletedScans) apply(model *Model) {
	images := []*Image{}
	for _, imageInfo := range model.Images {
		if imageInfo.ScanStatus == ScanStatusComplete {
			image := imageInfo.image()
			images = append(images, &image)
		}
	}
	go g.continuation(images)
}

type hubRecheckResults struct {
	scan HubImageScan
}

func (h *hubRecheckResults) apply(model *Model) {
	scan := h.scan
	imageInfo, ok := model.Images[scan.Sha]
	if !ok {
		log.Warnf("expected to already have image %s, but did not", string(scan.Sha))
		return
	}

	if scan.Scan != nil {
		imageInfo.ScanResults = scan.Scan
	}
}

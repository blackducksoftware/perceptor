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
	log "github.com/sirupsen/logrus"
)

type action interface {
	apply(model Model) Model
}

type addPod struct {
	pod Pod
}

func (a addPod) apply(model Model) Model {
	model.AddPod(a.pod)
	return model
}

type updatePod struct {
	pod Pod
}

func (u updatePod) apply(model Model) Model {
	model.AddPod(u.pod)
	return model
}

type deletePod struct {
	podName string
}

func (d deletePod) apply(model Model) Model {
	_, ok := model.Pods[d.podName]
	if !ok {
		log.Warnf("unable to delete pod %s, pod not found", d.podName)
		return model
	}
	delete(model.Pods, d.podName)
	return model
}

type addImage struct {
	image Image
}

func (a addImage) apply(model Model) Model {
	model.AddImage(a.image)
	return model
}

type allPods struct {
	pods []Pod
}

func (a allPods) apply(model Model) Model {
	model.Pods = map[string]Pod{}
	for _, pod := range a.pods {
		model.AddPod(pod)
	}
	return model
}

type getNextImage struct {
	continuation func(image *Image)
}

func (g getNextImage) apply(model Model) Model {
	log.Infof("looking for next image to scan with concurrency limit of %d, and %d currently in progress", model.ConcurrentScanLimit, model.inProgressScanCount())
	image := model.getNextImageFromScanQueue()
	g.continuation(image)
	return model
}

type finishScanClient struct {
	sha DockerImageSha
	err string
}

func (f finishScanClient) apply(model Model) Model {
	newModel := model
	log.Infof("finished scan client job action: error was empty? %t, %+v", f.err == "", f.sha)
	if f.err == "" {
		newModel.finishRunningScanClient(f.sha)
	} else {
		newModel.errorRunningScanClient(f.sha)
	}
	return newModel
}

type getNextImageForHubPolling struct {
	continuation func(image *Image)
}

func (g getNextImageForHubPolling) apply(model Model) Model {
	log.Infof("looking for next image to search for in hub")
	image := model.getNextImageFromHubCheckQueue()
	g.continuation(image)
	return model
}

type hubCheckResults struct {
	scan HubImageScan
}

func (h hubCheckResults) apply(model Model) Model {
	scan := h.scan
	imageInfo, ok := model.Images[scan.Sha]
	if !ok {
		log.Warnf("expected to already have image %s, but did not", string(scan.Sha))
		return model
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

	return model
}

type hubScanResults struct {
	scan HubImageScan
}

func (h hubScanResults) apply(model Model) Model {
	scan := h.scan
	imageInfo, ok := model.Images[scan.Sha]
	if !ok {
		log.Warnf("expected to already have image %s, but did not", string(scan.Sha))
		return model
	}

	imageInfo.ScanResults = scan.Scan

	//	log.Infof("completing image scan of image %s ? %t", image.ShaName(), scan.Scan.IsDone())
	if scan.Scan != nil && scan.Scan.IsDone() {
		imageInfo.setScanStatus(ScanStatusComplete)
	}

	return model
}

type requeueStalledScan struct {
	sha DockerImageSha
}

func (r requeueStalledScan) apply(model Model) Model {
	imageInfo, ok := model.Images[r.sha]
	if !ok {
		return model
	}
	if imageInfo.ScanStatus != ScanStatusRunningScanClient {
		return model
	}
	imageInfo.setScanStatus(ScanStatusError)
	model.addImageToScanQueue(r.sha)
	return model
}

type setConcurrentScanLimit struct {
	limit int
}

func (s setConcurrentScanLimit) apply(model Model) Model {
	limit := s.limit
	if limit < 0 {
		log.Errorf("cannot set concurrent scan limit to less than 0 (got %d)", limit)
		return model
	}
	model.ConcurrentScanLimit = limit
	return model
}

type allImages struct {
	images []Image
}

func (a allImages) apply(model Model) Model {
	for _, image := range a.images {
		model.AddImage(image)
	}
	return model
}

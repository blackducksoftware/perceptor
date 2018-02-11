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
	log "github.com/sirupsen/logrus"
)

type reducer struct {
	model <-chan Model
}

// logic

func newReducer(initialModel Model,
	actions <-chan action,
	getNextImageForHubPolling <-chan func(image *Image),
	hubCheckResults <-chan HubImageScan,
	hubScanResults <-chan HubImageScan) *reducer {
	model := initialModel
	modelStream := make(chan Model)
	go func() {
		for {
			select {
			case nextAction := <-actions:
				model = nextAction.apply(model)
				go func() {
					modelStream <- model
				}()
			case continuation := <-getNextImageForHubPolling:
				model = updateModelGetNextImageForHubPolling(continuation, model)
				go func() {
					modelStream <- model
				}()
			case imageScan := <-hubCheckResults:
				model = updateModelAddHubCheckResults(imageScan, model)
				go func() {
					modelStream <- model
				}()
			case imageScan := <-hubScanResults:
				model = updateModelAddHubScanResults(imageScan, model)
				go func() {
					modelStream <- model
				}()
			}
		}
	}()
	return &reducer{model: modelStream}
}

func updateModelGetNextImageForHubPolling(continuation func(image *Image), model Model) Model {
	log.Infof("looking for next image to search for in hub")
	image := model.getNextImageFromHubCheckQueue()
	continuation(image)
	return model
}

func updateModelAddHubCheckResults(scan HubImageScan, model Model) Model {
	image := scan.Image

	imageInfo, ok := model.Images[image.Sha]
	if !ok {
		log.Warnf("expected to already have image %s, but did not", image.HumanReadableName())
		return model
	}

	imageInfo.ScanResults = scan.Scan

	//	log.Infof("completing image scan of image %s ? %t", image.ShaName(), scan.Scan.IsDone())
	if scan.Scan == nil {
		model.addImageToScanQueue(image.Sha)
	} else if scan.Scan.IsDone() {
		imageInfo.ScanStatus = ScanStatusComplete
	} else {
		// it could be in the scan client stage, in the hub stage ...
		// maybe perceptor crashed and just came back up
		// since we don't know, we have to put it into the scan queue
		model.addImageToScanQueue(image.Sha)
	}

	return model
}

func updateModelAddHubScanResults(scan HubImageScan, model Model) Model {
	image := scan.Image

	imageInfo, ok := model.Images[image.Sha]
	if !ok {
		log.Warnf("expected to already have image %s, but did not", image.HumanReadableName())
		return model
	}

	imageInfo.ScanResults = scan.Scan

	//	log.Infof("completing image scan of image %s ? %t", image.ShaName(), scan.Scan.IsDone())
	if scan.Scan != nil && scan.Scan.IsDone() {
		imageInfo.ScanStatus = ScanStatusComplete
	}

	return model
}

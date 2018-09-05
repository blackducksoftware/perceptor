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

package model

import (
	log "github.com/sirupsen/logrus"
)

// PruneOrphanedImages .....
type PruneOrphanedImages struct {
	CompletedImageShas chan []string
}

// Apply .....
func (p *PruneOrphanedImages) Apply(model *Model) {
	// find images that aren't in a pod
	imagesInPod := map[DockerImageSha]bool{}
	for _, pod := range model.Pods {
		for _, cont := range pod.Containers {
			imagesInPod[cont.Image.Sha] = true
		}
	}
	//
	completed := []string{}
	deleteImmediately := []DockerImageSha{}
	for sha, imageInfo := range model.Images {
		if !imagesInPod[sha] {
			switch imageInfo.ScanStatus {
			case ScanStatusUnknown, ScanStatusInQueue:
				deleteImmediately = append(deleteImmediately, sha)
			case ScanStatusComplete:
				completed = append(completed, string(sha))
			default:
				// let's leave ScanStatusRunningHubScan and ScanStatusRunningScanClient
				// alone, so that they don't get messed up.  They can always be deleted
				// later.
			}
		}
	}
	// 1. immediately delete any orphaned images in the scan queue or status unknown
	for _, sha := range deleteImmediately {
		err := model.deleteImage(sha)
		if err != nil {
			log.Errorf("unable to delete image: %s", err.Error())
		}
	}
	// 2. get a list of completed images for further processing
	p.CompletedImageShas <- completed
}

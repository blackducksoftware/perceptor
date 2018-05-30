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

package actions

import (
	"time"

	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	log "github.com/sirupsen/logrus"
)

// EnqueueImagesNeedingRefreshing .....
type EnqueueImagesNeedingRefreshing struct {
	RefreshThresholdDuration time.Duration
}

// Apply .....
func (e *EnqueueImagesNeedingRefreshing) Apply(model *m.Model) {
	for sha, imageInfo := range model.Images {
		isComplete := imageInfo.ScanStatus == m.ScanStatusComplete
		if !isComplete {
			log.Debugf("not enqueueing %s: not complete", sha)
			continue
		}

		_, isInRefreshQueue := model.ImageRefreshQueueSet[sha]
		if isInRefreshQueue {
			log.Debugf("not enqueueing %s: already in refresh queue", sha)
			continue
		}

		hasBeenRefreshedRecently := time.Now().Sub(imageInfo.TimeOfLastRefresh) < e.RefreshThresholdDuration
		if hasBeenRefreshedRecently {
			log.Debugf("not enqueueing %s: has been refreshed recently", sha)
			continue
		}

		err := model.AddImageToRefreshQueue(sha)
		if err != nil {
			log.Error(err.Error())
			recordError("EnqueueImagesNeedingRefreshing", "unable to add image to refresh queue")
		}
	}
}

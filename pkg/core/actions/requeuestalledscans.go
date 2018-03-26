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
)

type RequeueStalledScans struct {
	StalledScanClientTimeout time.Duration
	StalledHubScanTimeout    time.Duration
}

func (r *RequeueStalledScans) Apply(model *m.Model) {
	for _, imageInfo := range model.Images {
		switch imageInfo.ScanStatus {
		case m.ScanStatusRunningScanClient:
			if imageInfo.TimeInCurrentScanStatus() > r.StalledScanClientTimeout {
				model.SetImageScanStatus(imageInfo.ImageSha, m.ScanStatusInQueue)
			}
		case m.ScanStatusRunningHubScan:
			if imageInfo.TimeInCurrentScanStatus() > r.StalledHubScanTimeout {
				model.SetImageScanStatus(imageInfo.ImageSha, m.ScanStatusInQueue)
			}
		default:
			// nothing to do
		}
	}
}

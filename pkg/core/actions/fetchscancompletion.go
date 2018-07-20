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
	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	"github.com/blackducksoftware/perceptor/pkg/hub"
	log "github.com/sirupsen/logrus"
)

// HubScanDidFinish .....
type HubScanDidFinish struct {
	HubURL string
	Scan   *hub.HubImageScan
}

// Apply .....
func (h *HubScanDidFinish) Apply(model *m.Model) {
	scan := h.Scan

	// case 0: in progress. shouldn't happen
	if scan.Scan.ScanSummaryStatus() == hub.ScanSummaryStatusInProgress {
		log.Errorf("unexpected scan status in progress.  expected failure or success")
		return
	}

	sha := m.DockerImageSha(scan.ScanName)

	// case 1: image mysteriously gone from model
	imageInfo, ok := model.Images[sha]
	if !ok {
		log.Errorf("expected to already have image %s, but did not", scan.ScanName)
		return
	}

	// case 2: failed.  Put it back in the scan queue
	if scan.Scan.ScanSummaryStatus() == hub.ScanSummaryStatusFailure {
		// TODO unassign it from its hub
		model.SetImageScanStatus(sha, m.ScanStatusInQueue)
		return
	}

	// case 3: success
	imageInfo.SetScanResults(scan.Scan)
	model.SetImageScanStatus(sha, m.ScanStatusComplete)
}

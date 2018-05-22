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

// FetchScanRefresh .....
type FetchScanRefresh struct {
	Scan *m.HubImageScan
}

// Apply .....
func (h *FetchScanRefresh) Apply(model *m.Model) {
	scan := h.Scan

	// case 0: unable to fetch scan results
	if scan.Err != nil {
		model.HubCircuitBreaker.HubFailure()
		log.Errorf("unable to fetch updated scan results for sha %s: %s", scan.Sha, scan.Err.Error())
		return
	}

	// case 1: image mysteriously gone from model
	imageInfo, ok := model.Images[scan.Sha]
	if !ok {
		log.Errorf("expected to already have image %s, but did not", string(scan.Sha))
		return
	}

	err := model.RemoveImageFromRefreshQueue(scan.Sha)
	if err != nil {
		log.Errorf("unable to remove %s from refresh queue: %s", scan.Sha, err.Error())
		// no need to return -- this should only happen if it wasn't in the refresh
		// queue already
	}

	// 2. successfully hit hub, but didn't find project
	//   not sure why this would happen -- we should ALWAYS find the hub project
	//   unless something else deleted it
	if scan.Scan == nil {
		log.Errorf("unable to fetch updated scan results for sha %s: got nil", scan.Sha)
		return
	}

	// 3. scan is not done or is failure -- not sure why this would happen
	if scan.Scan.ScanSummaryStatus() != hub.ScanSummaryStatusSuccess {
		log.Errorf("found scan for sha %s in status %s, expected completed scan", scan.Sha, scan.Scan.ScanSummaryStatus())
		return
	}

	// 4. successfully found project: update the image results
	log.Infof("received results for hub rechecking for sha %s: %+v", scan.Sha, scan.Scan)
	imageInfo.SetScanResults(scan.Scan)
}

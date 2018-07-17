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

// FetchScanInitial .....
type FetchScanInitial struct {
	Scan *m.HubScan
}

// Apply .....
func (h *FetchScanInitial) Apply(model *m.Model) {
	scan := h.Scan

	// case -1: image mysteriousy gone from model
	layerInfo, ok := model.Layers[scan.Sha]
	if !ok {
		log.Errorf("expected to already have sha %s, but did not", scan.Sha)
		return
	}

	// case 0: error trying to access the hub code location.  Don't change the status.
	if scan.Err != nil {
		log.Errorf("check image in hub -- unable to fetch layer scan for sha %s: %s", scan.Sha, scan.Err.Error())
		return
	}

	err := model.RemoveLayerFromHubCheckQueue(scan.Sha)
	if err != nil {
		log.Error(err.Error())
	}

	// case 1: image surprisingly has different status
	if layerInfo.ScanStatus != m.ScanStatusUnknown {
		log.Warnf("ignoring hub check results for sha %s, invalid status (expected Unknown, found %s)", scan.Sha, layerInfo.ScanStatus)
		return
	}

	// case 2: successfully determined that there's no code location, or no scan summary for that code location
	//   likely interpretation: no scan was started for this sha, and it needs to be run
	//   less likely interpretations:
	//     - a scan client was started, perceptor crashed, and the scan hasn't
	//       shown up in the hub yet.  TODO is there anything we can do in this case?
	//       For now, we'll just ignore this case.
	if scan.Scan == nil {
		log.Infof("check image in hub -- unable to find layer scan for sha %s, found nil", scan.Sha)
		model.SetLayerScanStatus(scan.Sha, m.ScanStatusNotScanned)
		return
	}

	// case 3: found hub code location, and it's complete
	if scan.Scan.ScanSummaryStatus() == hub.ScanSummaryStatusSuccess {
		log.Infof("check image in hub -- found finished layer scan for sha %s: %+v", scan.Sha, *scan)
		model.SetLayerScanStatus(scan.Sha, m.ScanStatusComplete)
		layerInfo.SetScanResults(scan.Scan)
		return
	}

	// case 4: found hub code location, and it failed
	if scan.Scan.ScanSummaryStatus() == hub.ScanSummaryStatusFailure {
		log.Infof("check image in hub -- found failed layer scan for sha %s: %+v", scan.Sha, *scan)
		model.SetLayerScanStatus(scan.Sha, m.ScanStatusNotScanned)
		return
	}

	// case 5: found hub code location, and it's in progress
	//   this likely means that a scan was started, perceptor went down, and now
	//   perceptor is recovering on initial startup.
	//   The scan could actually either be in the RunningScanClient or RunningHubScan
	//   stage; is there a way to determine which one it's in?
	//   For now, let's just assume it's in the RunningHubScan stage, and then if
	//   there's a problem, it'll automatically get rescheduled by the regular
	//   job that cleans up stalled scans.
	log.Infof("check image in hub -- found running scan for sha %s: %+v", scan.Sha, *scan)
	model.SetLayerScanStatus(scan.Sha, m.ScanStatusRunningHubScan)
}

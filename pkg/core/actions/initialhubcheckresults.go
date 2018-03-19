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
	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	"github.com/prometheus/common/log"
)

type InitialHubCheckResults struct {
	Scan *m.HubImageScan
}

func (h *InitialHubCheckResults) Apply(model *m.Model) {
	scan := h.Scan
	imageInfo, ok := model.Images[scan.Sha]
	if !ok {
		log.Warnf("expected to already have image %s, but did not", string(scan.Sha))
		return
	}

	// case 1: error trying to access the hub project
	//   Put the image back in the hub check queue.
	if scan.Err != nil {
		log.Errorf("check image in hub -- unable to fetch image scan for sha %s: %s", scan.Sha, scan.Err.Error())
		return
	}

	// case 2: successfully determined that there's no hub project
	//   likely interpretation: no scan was started for this sha, and it needs to be run
	//   less likely interpretations:
	//     - a scan client was started, perceptor crashed, and the scan hasn't
	//       shown up in the hub yet.  TODO is there anything we can do in this case?
	//       For now, we'll just ignore this case.
	if scan.Scan == nil {
		log.Infof("check image in hub -- unable to find image scan for sha %s, found nil", scan.Sha)
		model.AddImageToScanQueue(scan.Sha)
		return
	}

	// case 3: found hub project, and it's complete
	if scan.Scan.IsDone() {
		log.Infof("check image in hub -- found finished image scan for sha %s: %+v", scan.Sha, *scan)
		imageInfo.ScanResults = scan.Scan
		imageInfo.SetScanStatus(m.ScanStatusComplete)
		return
	}

	// case 4: found hub project, and it's in progress
	//   this likely means that a scan was started, perceptor went down, and now
	//   perceptor is recovering on initial startup.
	//   The scan could actually either be in the RunningScanClient or RunningHubScan
	//   stage; is there a way to determine which one it's in?
	//   For now, let's just assume it's in the RunningHubScan stage, and then if
	//   there's a problem, it'll automatically get rescheduled by the regular
	//   job that cleans up stalled scans.
	log.Infof("check image in hub -- found running scan for sha %s: %+v", scan.Sha, *scan)
	// imageInfo.ScanResults = scan.Scan // TODO we don't want to do this, do we?
	imageInfo.SetScanStatus(m.ScanStatusRunningHubScan)
}

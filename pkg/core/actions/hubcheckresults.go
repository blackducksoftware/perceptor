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

type HubCheckResults struct {
	Scan *m.HubImageScan
}

func (h *HubCheckResults) Apply(model *m.Model) {
	scan := h.Scan

	// case 1: error
	if scan.Err != nil {
		log.Errorf("error checking hub for completed scan for sha %s: %s", scan.Sha, scan.Err.Error())
		return
	}

	// case 2: nil
	if scan.Scan == nil {
		log.Debugf("found nil checking hub for completed scan for image %s", string(scan.Sha))
		return
	}

	// case 3: found it, and it's not done
	if scan.Scan.ScanSummaryStatus() == hub.ScanSummaryStatusInProgress {
		log.Debugf("found running scan in hub for image %s: %+v", string(scan.Sha), scan.Scan)
		return
	}

	// case 4: found it, and it failed.  Put it back in the scan queue
	if scan.Scan.ScanSummaryStatus() == hub.ScanSummaryStatusFailure {
		model.SetImageScanStatus(scan.Sha, m.ScanStatusInQueue)
		return
	}

	// case 5: found it, and it's done
	imageInfo, ok := model.Images[scan.Sha]
	if !ok {
		log.Errorf("expected to already have image %s, but did not", string(scan.Sha))
		return
	}

	imageInfo.ScanResults = scan.Scan
	model.SetImageScanStatus(scan.Sha, m.ScanStatusComplete)
}

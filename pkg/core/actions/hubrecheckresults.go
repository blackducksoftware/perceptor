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

type HubRecheckResults struct {
	Scan *m.HubImageScan
}

func (h *HubRecheckResults) Apply(model *m.Model) {
	scan := h.Scan
	imageInfo, ok := model.Images[scan.Sha]
	if !ok {
		log.Warnf("expected to already have image %s, but did not", string(scan.Sha))
		return
	}

	// case 1: unable to fetch scan results
	if scan.Err != nil {
		log.Errorf("unable to fetch updated scan results for sha %s: %s", scan.Sha, scan.Err.Error())
		return
	}

	// 2. successfully hit hub, but didn't find project
	//   not sure why this would happen -- we should ALWAYS find the hub project
	//   unless something else deleted it
	if scan.Scan == nil {
		log.Errorf("unable to fetch updated scan results for sha %s: got nil", scan.Sha)
		return
	}

	// 3. successfully found project: update the image results
	//   TODO: what if the scan is not done?  (and how/why would that happen?)
	log.Infof("received results for hub rechecking for sha %s: %+v", scan.Sha, scan.Scan)
	imageInfo.ScanResults = scan.Scan
}

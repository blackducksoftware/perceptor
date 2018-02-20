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

package hub

// ImageScan models the results that we expect to get from the hub after
// scanning a docker image.
type ImageScan struct {
	RiskProfile                      RiskProfile
	PolicyStatus                     PolicyStatus
	ScanSummary                      ScanSummary
	ComponentsHref                   string
	CodeLocationCreatedAt            string
	CodeLocationMappedProjectVersion string
	CodeLocationName                 string
	CodeLocationType                 string
	CodeLocationURL                  string
	CodeLocationUpdatedAt            string
}

// IsDone returns whether the hub imagescan results indicate that the scan is
// complete.
func (scan *ImageScan) IsDone() bool {
	return isScanSummaryStatusDone(scan.ScanSummary.Status)
}

func (scan *ImageScan) VulnerabilityCount() int {
	return scan.RiskProfile.HighRiskVulnerabilityCount()
}

func (scan *ImageScan) PolicyViolationCount() int {
	return scan.PolicyStatus.ViolationCount()
}

func (scan *ImageScan) OverallStatus() PolicyStatusType {
	return scan.PolicyStatus.OverallStatus
}

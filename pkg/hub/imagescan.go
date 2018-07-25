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

// ScanResults models the results that we expect to get from the hub after
// scanning a docker image.
type ScanResults struct {
	RiskProfile                      RiskProfile
	PolicyStatus                     PolicyStatus
	ScanSummaries                    []ScanSummary
	ComponentsHref                   string
	CodeLocationCreatedAt            string
	CodeLocationMappedProjectVersion string
	CodeLocationName                 string
	CodeLocationType                 string
	CodeLocationURL                  string
	CodeLocationUpdatedAt            string
}

// ScanSummaryStatus looks through all the scan summaries and:
//  - 1+ success: returns success
//  - 0 success, 1+ inprogress: returns inprogress
//  - 0 success, 0 inprogress: returns failure
// TODO: weird corner cases:
//  - no scan summaries ... ? should that be inprogress, or error?
//    or should we just assume that we'll always have at least 1?
func (scan *ScanResults) ScanSummaryStatus() ScanSummaryStatus {
	inProgress := false
	for _, scanSummary := range scan.ScanSummaries {
		switch scanSummary.Status {
		case ScanSummaryStatusSuccess:
			return ScanSummaryStatusSuccess
		case ScanSummaryStatusInProgress:
			inProgress = true
		default:
			// nothing to do
		}
	}
	if inProgress {
		return ScanSummaryStatusInProgress
	}
	return ScanSummaryStatusFailure
}

// VulnerabilityCount .....
func (scan *ScanResults) VulnerabilityCount() int {
	return scan.RiskProfile.HighRiskVulnerabilityCount()
}

// PolicyViolationCount .....
func (scan *ScanResults) PolicyViolationCount() int {
	return scan.PolicyStatus.ViolationCount()
}

// OverallStatus .....
func (scan *ScanResults) OverallStatus() PolicyStatusType {
	return scan.PolicyStatus.OverallStatus
}

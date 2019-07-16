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

import (
	"fmt"

	"github.com/blackducksoftware/hub-client-go/hubapi"
)

// CircuitBreakerState .....
type CircuitBreakerState int

// .....
const (
	CircuitBreakerStateDisabled CircuitBreakerState = iota
	CircuitBreakerStateEnabled  CircuitBreakerState = iota
	CircuitBreakerStateChecking CircuitBreakerState = iota
)

// String .....
func (state CircuitBreakerState) String() string {
	switch state {
	case CircuitBreakerStateDisabled:
		return "CircuitBreakerStateDisabled"
	case CircuitBreakerStateEnabled:
		return "CircuitBreakerStateEnabled"
	case CircuitBreakerStateChecking:
		return "CircuitBreakerStateChecking"
	}
	panic(fmt.Errorf("invalid CircuitBreakerState value: %d", state))
}

// MarshalJSON .....
func (state CircuitBreakerState) MarshalJSON() ([]byte, error) {
	jsonString := fmt.Sprintf(`"%s"`, state.String())
	return []byte(jsonString), nil
}

// MarshalText .....
func (state CircuitBreakerState) MarshalText() (text []byte, err error) {
	return []byte(state.String()), nil
}

// CodeLocation .....
type CodeLocation struct {
	ScanSummaries        []ScanSummary
	CreatedAt            string
	MappedProjectVersion string
	Name                 string
	CodeLocationType     string
	URL                  string
	UpdatedAt            string
}

// ClientStatus describes the state of a hub client
type ClientStatus int

// .....
const (
	ClientStatusError ClientStatus = iota
	ClientStatusUp    ClientStatus = iota
	ClientStatusDown  ClientStatus = iota
)

// String .....
func (status ClientStatus) String() string {
	switch status {
	case ClientStatusError:
		return "ClientStatusError"
	case ClientStatusUp:
		return "ClientStatusUp"
	case ClientStatusDown:
		return "ClientStatusDown"
	}
	panic(fmt.Errorf("invalid ClientStatus value: %d", status))
}

// MarshalJSON .....
func (status ClientStatus) MarshalJSON() ([]byte, error) {
	jsonString := fmt.Sprintf(`"%s"`, status.String())
	return []byte(jsonString), nil
}

// MarshalText .....
func (status ClientStatus) MarshalText() (text []byte, err error) {
	return []byte(status.String()), nil
}

// PolicyStatus .....
type PolicyStatus struct {
	OverallStatus                string
	UpdatedAt                    string
	ComponentVersionStatusCounts map[string]int
}

// ViolationCount .....
func (ps *PolicyStatus) ViolationCount() int {
	violationCount, ok := ps.ComponentVersionStatusCounts[PolicyStatusTypeInViolation]
	if !ok {
		return 0
	}
	return violationCount
}

// Project .....
type Project struct {
	Name     string
	Source   string
	Versions []Version
}

// RiskProfile .....
type RiskProfile struct {
	Categories       map[string]RiskProfileStatusCounts
	BomLastUpdatedAt string
}

// CriticalAndHighRiskVulnerabilityCount returns the combination of CRITICAL and HIGH risk profile count
func (rp *RiskProfile) CriticalAndHighRiskVulnerabilityCount() int {
	vulnerabilities, ok := rp.Categories[RiskProfileCategoryVulnerability]
	if !ok {
		return 0
	}
	return vulnerabilities.HighRiskVulnerabilityCount() + vulnerabilities.CriticalRiskVulnerabilityCount()
}

// RiskProfileStatusCounts .....
type RiskProfileStatusCounts struct {
	StatusCounts map[string]int
}

// HighRiskVulnerabilityCount .....
func (r *RiskProfileStatusCounts) HighRiskVulnerabilityCount() int {
	return r.StatusCounts[RiskProfileStatusHigh]
}

// CriticalRiskVulnerabilityCount return the CRITICAL vulnerability count
func (r *RiskProfileStatusCounts) CriticalRiskVulnerabilityCount() int {
	return r.StatusCounts[RiskProfileStatusCritical]
}

// ScanStage describes the current stage of the scan
type ScanStage int

// ...
const (
	ScanStageUnknown    ScanStage = iota
	ScanStageScanClient ScanStage = iota
	ScanStageHubScan    ScanStage = iota
	ScanStageComplete   ScanStage = iota
	ScanStageFailure    ScanStage = iota
)

// String .....
func (s ScanStage) String() string {
	switch s {
	case ScanStageUnknown:
		return "ScanStageUnknown"
	case ScanStageScanClient:
		return "ScanStageScanClient"
	case ScanStageHubScan:
		return "ScanStageHubScan"
	case ScanStageComplete:
		return "ScanStageComplete"
	case ScanStageFailure:
		return "ScanStageFailure"
	default:
		panic(fmt.Errorf("invalid ScanStage value: %d", s))
	}
}

// Scan is a wrapper around a Hub code location, and full scan results.
// If `ScanResults` is nil, that means the ScanResults have not been fetched yet.
type Scan struct {
	Stage       ScanStage
	ScanResults *ScanResults
}

// ScanResults models the results that we expect to get from the hub after
// scanning a docker image.
type ScanResults struct {
	RiskProfile                      RiskProfile
	PolicyStatus                     PolicyStatus
	ScanSummaries                    []ScanSummary
	ComponentsHref                   string
	CodeLocationCreatedAt            string
	CodeLocationHref                 string
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

// IsDone returns true if at least one scan summary is successfully finished.
func (scan *ScanResults) IsDone() bool {
	switch scan.ScanSummaryStatus() {
	case ScanSummaryStatusInProgress:
		return false
	default:
		return true
	}
}

// VulnerabilityCount .....
func (scan *ScanResults) VulnerabilityCount() int {
	return scan.RiskProfile.CriticalAndHighRiskVulnerabilityCount()
}

// PolicyViolationCount .....
func (scan *ScanResults) PolicyViolationCount() int {
	return scan.PolicyStatus.ViolationCount()
}

// OverallStatus .....
func (scan *ScanResults) OverallStatus() string {
	return scan.PolicyStatus.OverallStatus
}

// ScanSummary .....
type ScanSummary struct {
	CreatedAt string
	Status    ScanSummaryStatus
	UpdatedAt string
}

// NewScanSummaryFromHub .....
func NewScanSummaryFromHub(hubScanSummary hubapi.ScanSummary) *ScanSummary {
	return &ScanSummary{
		CreatedAt: hubScanSummary.CreatedAt,
		Status:    parseScanSummaryStatus(hubScanSummary.Status),
		UpdatedAt: hubScanSummary.UpdatedAt,
	}
}

// ScanSummaryStatus .....
type ScanSummaryStatus int

// .....
const (
	ScanSummaryStatusInProgress ScanSummaryStatus = iota
	ScanSummaryStatusSuccess    ScanSummaryStatus = iota
	ScanSummaryStatusFailure    ScanSummaryStatus = iota
)

// String .....
func (status ScanSummaryStatus) String() string {
	switch status {
	case ScanSummaryStatusInProgress:
		return "ScanSummaryStatusInProgress"
	case ScanSummaryStatusSuccess:
		return "ScanSummaryStatusSuccess"
	case ScanSummaryStatusFailure:
		return "ScanSummaryStatusFailure"
	}
	panic(fmt.Errorf("invalid ScanSummaryStatus value: %d", status))
}

func parseScanSummaryStatus(statusString string) ScanSummaryStatus {
	switch statusString {
	case "COMPLETE":
		return ScanSummaryStatusSuccess
	case "ERROR", "ERROR_BUILDING_BOM", "ERROR_MATCHING", "ERROR_SAVING_SCAN_DATA", "ERROR_SCANNING", "CANCELLED":
		return ScanSummaryStatusFailure
	default:
		return ScanSummaryStatusInProgress
	}
}

// Version .....
type Version struct {
	CodeLocations   []CodeLocation
	RiskProfile     RiskProfile
	PolicyStatus    PolicyStatus
	Distribution    string
	Nickname        string
	VersionName     string
	ReleasedOn      string
	ReleaseComments string
	Phase           string
}

// Update ...
type Update interface {
	updateMarker()
}

// DidFindScan ...
type DidFindScan struct {
	Name    string
	Results *ScanResults
}

func (dfs *DidFindScan) updateMarker() {}

// DidFinishScan ...
type DidFinishScan struct {
	Name    string
	Results *ScanResults
}

func (dfs *DidFinishScan) updateMarker() {}

// DidRefreshScan ...
type DidRefreshScan struct {
	Name    string
	Results *ScanResults
}

func (drs *DidRefreshScan) updateMarker() {}

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
	"encoding/json"
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

type clientStateMetrics struct {
	scanStageCounts map[ScanStage]int
	errorsCount     int
}

// PolicyStatus .....
type PolicyStatus struct {
	OverallStatus                PolicyStatusType
	UpdatedAt                    string
	ComponentVersionStatusCounts map[PolicyStatusType]int
}

// ViolationCount .....
func (ps *PolicyStatus) ViolationCount() int {
	violationCount, ok := ps.ComponentVersionStatusCounts[PolicyStatusTypeInViolation]
	if !ok {
		return 0
	}
	return violationCount
}

// PolicyStatusType .....
type PolicyStatusType int

// .....
const (
	PolicyStatusTypeNotInViolation        PolicyStatusType = iota
	PolicyStatusTypeInViolation           PolicyStatusType = iota
	PolicyStatusTypeInViolationOverridden PolicyStatusType = iota
)

// String .....
func (p PolicyStatusType) String() string {
	switch p {
	case PolicyStatusTypeNotInViolation:
		return "NOT_IN_VIOLATION"
	case PolicyStatusTypeInViolation:
		return "IN_VIOLATION"
	case PolicyStatusTypeInViolationOverridden:
		return "IN_VIOLATION_OVERRIDDEN"
	default:
		panic(fmt.Errorf("invalid PolicyStatusType value: %d", p))
	}
}

// MarshalJSON .....
func (p PolicyStatusType) MarshalJSON() ([]byte, error) {
	jsonString := fmt.Sprintf(`"%s"`, p.String())
	return []byte(jsonString), nil
}

// MarshalText .....
func (p PolicyStatusType) MarshalText() (text []byte, err error) {
	return []byte(p.String()), nil
}

// UnmarshalText .....
func (p *PolicyStatusType) UnmarshalText(text []byte) (err error) {
	status, err := parseHubPolicyStatusType(string(text))
	if err != nil {
		return err
	}
	*p = status
	return nil
}

// Project .....
type Project struct {
	Name     string
	Source   string
	Versions []Version
}

// Result models computations that may succeed or fail.
type Result struct {
	Value interface{}
	Err   error
}

// RiskProfile .....
type RiskProfile struct {
	Categories       map[RiskProfileCategory]RiskProfileStatusCounts
	BomLastUpdatedAt string
}

// HighRiskVulnerabilityCount .....
func (rp *RiskProfile) HighRiskVulnerabilityCount() int {
	vulnerabilities, ok := rp.Categories[RiskProfileCategoryVulnerability]
	if !ok {
		return 0
	}
	return vulnerabilities.HighRiskVulnerabilityCount()
}

// RiskProfileCategory .....
type RiskProfileCategory int

// .....
const (
	RiskProfileCategoryActivity      RiskProfileCategory = iota
	RiskProfileCategoryLicense       RiskProfileCategory = iota
	RiskProfileCategoryOperational   RiskProfileCategory = iota
	RiskProfileCategoryVersion       RiskProfileCategory = iota
	RiskProfileCategoryVulnerability RiskProfileCategory = iota
)

// String .....
func (r RiskProfileCategory) String() string {
	switch r {
	case RiskProfileCategoryActivity:
		return "ACTIVITY"
	case RiskProfileCategoryLicense:
		return "LICENSE"
	case RiskProfileCategoryOperational:
		return "OPERATIONAL"
	case RiskProfileCategoryVersion:
		return "VERSION"
	case RiskProfileCategoryVulnerability:
		return "VULNERABILITY"
	default:
		panic(fmt.Errorf("invalid RiskProfileCategory value: %d", r))
	}
}

// func (r RiskProfileCategory) MarshalJSON() ([]byte, error) {
// 	jsonString := fmt.Sprintf(`"%s"`, r.String())
// 	return []byte(jsonString), nil
// }

// UnmarshalJSON .....
func (r *RiskProfileCategory) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	status, err := parseHubRiskProfileCategory(str)
	if err != nil {
		return err
	}
	*r = status
	return nil
}

// MarshalText .....
func (r RiskProfileCategory) MarshalText() (text []byte, err error) {
	return []byte(r.String()), nil
}

// UnmarshalText .....
func (r *RiskProfileCategory) UnmarshalText(text []byte) (err error) {
	status, err := parseHubRiskProfileCategory(string(text))
	if err != nil {
		return err
	}
	*r = status
	return nil
}

// RiskProfileStatus .....
type RiskProfileStatus int

// .....
const (
	RiskProfileStatusHigh    RiskProfileStatus = iota
	RiskProfileStatusMedium  RiskProfileStatus = iota
	RiskProfileStatusLow     RiskProfileStatus = iota
	RiskProfileStatusOK      RiskProfileStatus = iota
	RiskProfileStatusUnknown RiskProfileStatus = iota
)

// String .....
func (r RiskProfileStatus) String() string {
	switch r {
	case RiskProfileStatusHigh:
		return "HIGH"
	case RiskProfileStatusMedium:
		return "MEDIUM"
	case RiskProfileStatusLow:
		return "LOW"
	case RiskProfileStatusOK:
		return "OK"
	case RiskProfileStatusUnknown:
		return "UNKNOWN"
	default:
		panic(fmt.Errorf("invalid RiskProfileStatus value: %d", r))
	}
}

// MarshalJSON .....
func (r RiskProfileStatus) MarshalJSON() ([]byte, error) {
	jsonString := fmt.Sprintf(`"%s"`, r.String())
	return []byte(jsonString), nil
}

// UnmarshalJSON .....
func (r *RiskProfileStatus) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	status, err := parseHubRiskProfileStatus(str)
	if err != nil {
		return err
	}
	*r = status
	return nil
}

// MarshalText .....
func (r RiskProfileStatus) MarshalText() (text []byte, err error) {
	return []byte(r.String()), nil
}

// UnmarshalText .....
func (r *RiskProfileStatus) UnmarshalText(text []byte) (err error) {
	status, err := parseHubRiskProfileStatus(string(text))
	if err != nil {
		return err
	}
	*r = status
	return nil
}

// RiskProfileStatusCounts .....
type RiskProfileStatusCounts struct {
	StatusCounts map[RiskProfileStatus]int
}

// HighRiskVulnerabilityCount .....
func (r *RiskProfileStatusCounts) HighRiskVulnerabilityCount() int {
	highCount, ok := r.StatusCounts[RiskProfileStatusHigh]
	if !ok {
		return 0
	}
	return highCount
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

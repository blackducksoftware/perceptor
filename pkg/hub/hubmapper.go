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

func parseHubRiskProfileStatus(hubName string) (RiskProfileStatus, error) {
	switch hubName {
	case "HIGH":
		return RiskProfileStatusHigh, nil
	case "MEDIUM":
		return RiskProfileStatusMedium, nil
	case "LOW":
		return RiskProfileStatusLow, nil
	case "OK":
		return RiskProfileStatusOK, nil
	case "UNKNOWN":
		return RiskProfileStatusUnknown, nil
	default:
		return RiskProfileStatusUnknown, fmt.Errorf("invalid hub name for risk profile status: %s", hubName)
	}
}

func newRiskProfileStatusCounts(hubCounts map[string]int) (*RiskProfileStatusCounts, error) {
	statusCounts := map[RiskProfileStatus]int{}
	for hubName, count := range hubCounts {
		status, err := parseHubRiskProfileStatus(hubName)
		if err != nil {
			return nil, err
		}
		statusCounts[status] = count
	}
	return &RiskProfileStatusCounts{StatusCounts: statusCounts}, nil
}

func parseHubRiskProfileCategory(hubName string) (RiskProfileCategory, error) {
	switch hubName {
	case "ACTIVITY":
		return RiskProfileCategoryActivity, nil
	case "LICENSE":
		return RiskProfileCategoryLicense, nil
	case "OPERATIONAL":
		return RiskProfileCategoryOperational, nil
	case "VERSION":
		return RiskProfileCategoryVersion, nil
	case "VULNERABILITY":
		return RiskProfileCategoryVulnerability, nil
	default:
		return RiskProfileCategoryActivity, fmt.Errorf("invalid hub name for risk profile category: %s", hubName)
	}
}

func newRiskProfile(bomLastUpdatedAt string, hubCategories map[string]map[string]int) (*RiskProfile, error) {
	categories := map[RiskProfileCategory]RiskProfileStatusCounts{}
	for hubCategory, hubCounts := range hubCategories {
		category, err := parseHubRiskProfileCategory(hubCategory)
		if err != nil {
			return nil, err
		}
		counts, err := newRiskProfileStatusCounts(hubCounts)
		if err != nil {
			return nil, err
		}
		categories[category] = *counts
	}
	return &RiskProfile{BomLastUpdatedAt: bomLastUpdatedAt, Categories: categories}, nil
}

func parseHubPolicyStatusType(hubName string) (PolicyStatusType, error) {
	switch hubName {
	case "NOT_IN_VIOLATION":
		return PolicyStatusTypeNotInViolation, nil
	case "IN_VIOLATION":
		return PolicyStatusTypeInViolation, nil
	case "IN_VIOLATION_OVERRIDDEN":
		return PolicyStatusTypeInViolationOverridden, nil
	default:
		return PolicyStatusTypeInViolation, fmt.Errorf("invalid hub name for policy status type: %s", hubName)
	}
}

func newPolicyStatus(hubOverallStatus string, hubUpdatedAt string, hubComponentVersionStatusCounts []hubapi.ComponentVersionStatusCount) (*PolicyStatus, error) {
	overallStatus, err := parseHubPolicyStatusType(hubOverallStatus)
	if err != nil {
		return nil, err
	}
	statusCounts := map[PolicyStatusType]int{}
	for _, hubStatusCount := range hubComponentVersionStatusCounts {
		status, err := parseHubPolicyStatusType(hubStatusCount.Name)
		if err != nil {
			return nil, err
		}
		statusCounts[status] = hubStatusCount.Value
	}
	return &PolicyStatus{OverallStatus: overallStatus, UpdatedAt: hubUpdatedAt, ComponentVersionStatusCounts: statusCounts}, nil
}

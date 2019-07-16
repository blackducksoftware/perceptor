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
	"github.com/blackducksoftware/hub-client-go/hubapi"
)

func newRiskProfileStatusCounts(riskProfileStatusCounts map[string]int) (*RiskProfileStatusCounts, error) {
	statusCounts := map[string]int{}
	for riskProfileStatus, count := range riskProfileStatusCounts {
		statusCounts[riskProfileStatus] = count
	}
	return &RiskProfileStatusCounts{StatusCounts: statusCounts}, nil
}

func newRiskProfile(bomLastUpdatedAt string, hubCategories map[string]map[string]int) (*RiskProfile, error) {
	categories := map[string]RiskProfileStatusCounts{}
	for category, riskProfileStatuses := range hubCategories {
		counts, err := newRiskProfileStatusCounts(riskProfileStatuses)
		if err != nil {
			return nil, err
		}
		categories[category] = *counts
	}
	return &RiskProfile{BomLastUpdatedAt: bomLastUpdatedAt, Categories: categories}, nil
}

func newPolicyStatus(hubOverallStatus string, hubUpdatedAt string, hubComponentVersionStatusCounts []hubapi.ComponentVersionStatusCount) (*PolicyStatus, error) {
	statusCounts := map[string]int{}
	for _, hubStatusCount := range hubComponentVersionStatusCounts {
		statusCounts[hubStatusCount.Name] = hubStatusCount.Value
	}
	return &PolicyStatus{OverallStatus: hubOverallStatus, UpdatedAt: hubUpdatedAt, ComponentVersionStatusCounts: statusCounts}, nil
}

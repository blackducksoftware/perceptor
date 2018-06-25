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

// MockHubClient ...
type MockHubClient struct {
	ShouldFail bool
}

// ListAllCodeLocations ...
func (mhc *MockHubClient) ListAllCodeLocations(options *hubapi.GetListOptions) (*hubapi.CodeLocationList, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch code locations list")
	}
	return &hubapi.CodeLocationList{}, nil
}

// GetProject ...
func (mhc *MockHubClient) GetProject(link hubapi.ResourceLink) (*hubapi.Project, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch project")
	}
	return &hubapi.Project{}, nil
}

// GetProjectVersion ...
func (mhc *MockHubClient) GetProjectVersion(link hubapi.ResourceLink) (*hubapi.ProjectVersion, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch project version")
	}
	return &hubapi.ProjectVersion{}, nil
}

// ListScanSummaries ...
func (mhc *MockHubClient) ListScanSummaries(link hubapi.ResourceLink) (*hubapi.ScanSummaryList, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch scan summary list")
	}
	return &hubapi.ScanSummaryList{}, nil
}

// GetProjectVersionRiskProfile ...
func (mhc *MockHubClient) GetProjectVersionRiskProfile(link hubapi.ResourceLink) (*hubapi.ProjectVersionRiskProfile, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch project version risk profile")
	}
	return &hubapi.ProjectVersionRiskProfile{}, nil
}

// GetProjectVersionPolicyStatus ...
func (mhc *MockHubClient) GetProjectVersionPolicyStatus(link hubapi.ResourceLink) (*hubapi.ProjectVersionPolicyStatus, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch project version policy status")
	}
	return &hubapi.ProjectVersionPolicyStatus{}, nil
}

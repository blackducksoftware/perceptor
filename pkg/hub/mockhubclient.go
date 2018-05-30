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

// ListProjects ...
func (mhc *MockHubClient) ListProjects(options *hubapi.GetListOptions) (*hubapi.ProjectList, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch project list")
	}
	return &hubapi.ProjectList{}, nil
}

// ListProjectVersions ...
func (mhc *MockHubClient) ListProjectVersions(link hubapi.ResourceLink, options *hubapi.GetListOptions) (*hubapi.ProjectVersionList, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch project version list")
	}
	return &hubapi.ProjectVersionList{}, nil
}

// ListScanSummaries ...
func (mhc *MockHubClient) ListScanSummaries(link hubapi.ResourceLink) (*hubapi.ScanSummaryList, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch scan summaries")
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

// ListCodeLocations ...
func (mhc *MockHubClient) ListCodeLocations(link hubapi.ResourceLink, options *hubapi.GetListOptions) (*hubapi.CodeLocationList, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch code locations")
	}
	return &hubapi.CodeLocationList{}, nil
}

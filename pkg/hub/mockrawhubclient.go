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
	"time"

	"github.com/blackducksoftware/hub-client-go/hubapi"
)

// MockRawClient ...
type MockRawClient struct {
	ShouldFail bool
}

// ListAllCodeLocations ...
func (mhc *MockRawClient) ListAllCodeLocations(options *hubapi.GetListOptions) (*hubapi.CodeLocationList, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch code locations list")
	}
	return &hubapi.CodeLocationList{}, nil
}

// CurrentVersion ...
func (mhc *MockRawClient) CurrentVersion() (*hubapi.CurrentVersion, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch current version")
	}
	return &hubapi.CurrentVersion{}, nil
}

// ListProjects ...
func (mhc *MockRawClient) ListProjects(options *hubapi.GetListOptions) (*hubapi.ProjectList, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch project list")
	}
	return &hubapi.ProjectList{}, nil
}

// DeleteCodeLocation ...
func (mhc *MockRawClient) DeleteCodeLocation(scanName string) error {
	if mhc.ShouldFail {
		return fmt.Errorf("unable to delete code location %s", scanName)
	}
	return nil
}

// DeleteProjectVersion ...
func (mhc *MockRawClient) DeleteProjectVersion(name string) error {
	if mhc.ShouldFail {
		return fmt.Errorf("unable to delete project %s", name)
	}
	return nil
}

// GetProject ...
func (mhc *MockRawClient) GetProject(link hubapi.ResourceLink) (*hubapi.Project, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch project")
	}
	return &hubapi.Project{}, nil
}

// GetProjectVersion ...
func (mhc *MockRawClient) GetProjectVersion(link hubapi.ResourceLink) (*hubapi.ProjectVersion, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch project version")
	}
	return &hubapi.ProjectVersion{}, nil
}

// ListScanSummaries ...
func (mhc *MockRawClient) ListScanSummaries(link hubapi.ResourceLink) (*hubapi.ScanSummaryList, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch scan summary list")
	}
	return &hubapi.ScanSummaryList{}, nil
}

// Login ...
func (mhc *MockRawClient) Login(username string, password string) error {
	if mhc.ShouldFail {
		return fmt.Errorf("unable to login")
	}
	return nil
}

// SetTimeout ...
func (mhc *MockRawClient) SetTimeout(timeout time.Duration) {}

// GetProjectVersionRiskProfile ...
func (mhc *MockRawClient) GetProjectVersionRiskProfile(link hubapi.ResourceLink) (*hubapi.ProjectVersionRiskProfile, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch project version risk profile")
	}
	return &hubapi.ProjectVersionRiskProfile{}, nil
}

// GetProjectVersionPolicyStatus ...
func (mhc *MockRawClient) GetProjectVersionPolicyStatus(link hubapi.ResourceLink) (*hubapi.ProjectVersionPolicyStatus, error) {
	if mhc.ShouldFail {
		return nil, fmt.Errorf("unable to fetch project version policy status")
	}
	return &hubapi.ProjectVersionPolicyStatus{}, nil
}

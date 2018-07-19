package hub

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

import (
	"fmt"

	"github.com/blackducksoftware/hub-client-go/hubapi"
)

// Hack for virtual hub functionality.
type HubCache struct {
	Items map[string]interface{}
}

// ListCodeLocations ...
func (cb *HubCache) ListCodeLocations(codeLocationName string) *hubapi.CodeLocationList {
	if v, ok := cb.Items[fmt.Sprintf("%v", codeLocationName)]; ok {
		return v.(*hubapi.CodeLocationList)
	}
	return nil
}

// GetProjectVersion ...
func (cb *HubCache) GetProjectVersion(link hubapi.ResourceLink) *hubapi.ProjectVersion {
	if v, ok := cb.Items[fmt.Sprintf("%v", link.Href)]; ok {
		return v.(*hubapi.ProjectVersion)
	}
	return nil
}

// GetProject ...
func (cb *HubCache) GetProject(link hubapi.ResourceLink) *hubapi.Project {
	if v, ok := cb.Items[fmt.Sprintf("%v", link.Href)]; ok {
		return v.(*hubapi.Project)
	}
	return nil
}

// GetProjectVersionRiskProfile ...
func (cb *HubCache) GetProjectVersionRiskProfile(link hubapi.ResourceLink) *hubapi.ProjectVersionRiskProfile {
	if v, ok := cb.Items[fmt.Sprintf("%v", link.Href)]; ok {
		return v.(*hubapi.ProjectVersionRiskProfile)
	}
	return nil
}

// GetProjectVersionPolicyStatus ...
func (cb *HubCache) GetProjectVersionPolicyStatus(link hubapi.ResourceLink) *hubapi.ProjectVersionPolicyStatus {
	if v, ok := cb.Items[fmt.Sprintf("%v", link.Href)]; ok {
		return v.(*hubapi.ProjectVersionPolicyStatus)
	}
	return nil
}

// ListScanSummaries ...
func (cb *HubCache) ListScanSummaries(link hubapi.ResourceLink) *hubapi.ScanSummaryList {
	if v, ok := cb.Items[fmt.Sprintf("%v", link.Href)]; ok {
		return v.(*hubapi.ScanSummaryList)
	}
	return nil
}

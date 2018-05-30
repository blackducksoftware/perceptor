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

import "github.com/blackducksoftware/hub-client-go/hubapi"

// ClientInterface provides an interface around hub-client-go's client,
// allowing it to be mocked for testing.
type ClientInterface interface {
	ListProjects(options *hubapi.GetListOptions) (*hubapi.ProjectList, error)
	ListProjectVersions(link hubapi.ResourceLink, options *hubapi.GetListOptions) (*hubapi.ProjectVersionList, error)
	ListScanSummaries(link hubapi.ResourceLink) (*hubapi.ScanSummaryList, error)
	GetProjectVersionRiskProfile(link hubapi.ResourceLink) (*hubapi.ProjectVersionRiskProfile, error)
	GetProjectVersionPolicyStatus(link hubapi.ResourceLink) (*hubapi.ProjectVersionPolicyStatus, error)
	ListCodeLocations(link hubapi.ResourceLink, options *hubapi.GetListOptions) (*hubapi.CodeLocationList, error)
}

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
	"github.com/blackducksoftware/hub-client-go/hubclient"
	log "github.com/sirupsen/logrus"
)

type Fetcher struct {
	client     hubclient.Client
	hubVersion string
	username   string
	password   string
	baseURL    string
}

func (hf *Fetcher) Login() error {
	start := time.Now()
	err := hf.client.Login(hf.username, hf.password)
	recordHubResponseTime("login", time.Now().Sub(start))
	return err
}

func (hf *Fetcher) fetchHubVersion() error {
	start := time.Now()
	currentVersion, err := hf.client.CurrentVersion()
	recordHubResponseTime("version", time.Now().Sub(start))
	if err != nil {
		log.Errorf("unable to get hub version: %s", err.Error())
		return err
	}

	hf.hubVersion = currentVersion.Version
	log.Infof("successfully got hub version %s", hf.hubVersion)
	return nil
}

// NewFetcher returns a new, logged-in Fetcher.
// It will instead return an error if any of the following happen:
//  - unable to instantiate an API client
//  - unable to sign in to the Hub
//  - unable to get hub version from the Hub
func NewFetcher(username string, password string, baseURL string, hubClientTimeoutSeconds int) (*Fetcher, error) {
	hubClientTimeout := time.Second * time.Duration(hubClientTimeoutSeconds)
	client, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings, hubClientTimeout)
	if err != nil {
		return nil, err
	}
	hf := Fetcher{
		client:   *client,
		username: username,
		password: password,
		baseURL:  baseURL}
	err = hf.Login()
	if err != nil {
		return nil, err
	}
	err = hf.fetchHubVersion()
	if err != nil {
		return nil, err
	}
	return &hf, nil
}

func (hf *Fetcher) HubVersion() string {
	return hf.hubVersion
}

// FetchScanFromImage returns an ImageScan only if:
// - it can find a project with the matching name, with
// - a project version with the matching name, with
// - one code location, with
// - one scan summary, with
// - a completed status
func (hf *Fetcher) FetchScanFromImage(image ImageInterface) (*ImageScan, error) {
	queryString := fmt.Sprintf("name:%s", image.HubProjectNameSearchString())
	startGetProjects := time.Now()
	projectList, err := hf.client.ListProjects(&hubapi.GetListOptions{Q: &queryString})
	recordHubResponseTime("projects", time.Now().Sub(startGetProjects))
	recordHubResponse("projects", err == nil)

	if err != nil {
		log.Errorf("error fetching project list: %v", err)
		return nil, err
	}
	projects := projectList.Items
	switch len(projects) {
	case 0:
		recordHubData("projects", true)
		return nil, nil
	case 1:
		recordHubData("projects", true) // good to go
	default:
		recordHubData("projects", false)
		log.Warnf("expected 1 project matching name search string %s, found %d", image.HubProjectNameSearchString(), len(projects))
	}

	project := projects[0]
	return hf.fetchImageScanUsingProject(project, image)
}

func (hf *Fetcher) fetchImageScanUsingProject(project hubapi.Project, image ImageInterface) (*ImageScan, error) {
	client := hf.client

	link, err := project.GetProjectVersionsLink()
	if err != nil {
		log.Errorf("error getting project versions link: %v", err)
		return nil, err
	}
	q := fmt.Sprintf("versionName:%s", image.HubProjectVersionNameSearchString())
	options := hubapi.GetListOptions{Q: &q}
	startGetVersions := time.Now()
	versionList, err := client.ListProjectVersions(*link, &options)
	recordHubResponseTime("projectVersions", time.Now().Sub(startGetVersions))
	recordHubResponse("projectVersions", err == nil)

	if err != nil {
		log.Errorf("error fetching project versions: %v", err)
		return nil, err
	}

	versions := []hubapi.ProjectVersion{}
	for _, v := range versionList.Items {
		if v.VersionName == image.HubProjectVersionNameSearchString() {
			versions = append(versions, v)
		}
	}

	switch len(versions) {
	case 0:
		recordHubData("project versions", true)
		return nil, nil
	case 1:
		recordHubData("project versions", true) // good to go, continue
	default:
		recordHubData("project versions", false)
		log.Warnf("expected to find one project version of name %s, found %d", image.HubProjectVersionNameSearchString(), len(versions))
	}

	version := versions[0]

	riskProfileLink, err := version.GetProjectVersionRiskProfileLink()
	if err != nil {
		log.Errorf("error getting risk profile link: %v", err)
		return nil, err
	}

	startGetRiskProfile := time.Now()
	riskProfile, err := client.GetProjectVersionRiskProfile(*riskProfileLink)
	recordHubResponseTime("projectVersionRiskProfile", time.Now().Sub(startGetRiskProfile))
	recordHubResponse("projectVersionRiskProfile", err == nil)
	if err != nil {
		log.Errorf("error fetching project version risk profile: %v", err)
		return nil, err
	}

	policyStatusLink, err := version.GetProjectVersionPolicyStatusLink()
	if err != nil {
		log.Errorf("error getting policy status link: %v", err)
		return nil, err
	}
	startGetPolicyStatus := time.Now()
	policyStatus, err := client.GetProjectVersionPolicyStatus(*policyStatusLink)
	recordHubResponseTime("projectVersionPolicyStatus", time.Now().Sub(startGetPolicyStatus))
	recordHubResponse("projectVersionPolicyStatus", err == nil)
	if err != nil {
		log.Errorf("error fetching project version policy status: %v", err)
		return nil, err
	}

	componentsLink, err := version.GetComponentsLink()
	if err != nil {
		log.Errorf("error getting components link: %v", err)
		return nil, err
	}

	codeLocationsLink, err := version.GetCodeLocationsLink()
	if err != nil {
		log.Errorf("error getting code locations link: %v", err)
		return nil, err
	}
	startGetCodeLocations := time.Now()
	codeLocationsList, err := client.ListCodeLocations(*codeLocationsLink, nil)
	recordHubResponseTime("codeLocations", time.Now().Sub(startGetCodeLocations))
	recordHubResponse("codeLocations", err == nil)
	if err != nil {
		log.Errorf("error fetching code locations: %v", err)
		return nil, err
	}

	codeLocations := []hubapi.CodeLocation{}
	for _, cl := range codeLocationsList.Items {
		if cl.Name == image.HubScanNameSearchString() {
			codeLocations = append(codeLocations, cl)
		}
	}

	switch len(codeLocations) {
	case 0:
		recordHubData("code locations", true)
		return nil, nil
	case 1:
		recordHubData("code locations", true) // good to go, continue
	default:
		recordHubData("code locations", false)
		log.Warnf("Found %d code locations for version %s, expected 1", len(codeLocations), version.VersionName)
	}

	codeLocation := codeLocations[0]

	scanSummariesLink, err := codeLocation.GetScanSummariesLink()
	if err != nil {
		log.Errorf("error getting scan summaries link: %v", err)
		return nil, err
	}
	startGetScanSummaries := time.Now()
	scanSummariesList, err := client.ListScanSummaries(*scanSummariesLink)
	recordHubResponseTime("scanSummaries", time.Now().Sub(startGetScanSummaries))
	recordHubResponse("scanSummaries", err == nil)
	if err != nil {
		log.Errorf("error fetching scan summaries: %v", err)
		return nil, err
	}

	scanSummaries := []hubapi.ScanSummary{}
	for _, scanSummary := range scanSummariesList.Items {
		if parseScanSummaryStatus(scanSummary.Status) == ScanSummaryStatusSuccess {
			scanSummaries = append(scanSummaries, scanSummary)
		}
	}

	switch len(scanSummaries) {
	case 0:
		recordHubData("scan summaries", true)
		return nil, nil
	case 1:
		recordHubData("scan summaries", true) // good to go, continue
	default:
		recordHubData("scan summaries", false)
		log.Warnf("expected to find one scan summary for code location %s, found %d", image.HubScanNameSearchString(), len(scanSummariesList.Items))
	}

	scanSummary := scanSummaries[0]

	mappedRiskProfile, err := newRiskProfile(riskProfile.BomLastUpdatedAt, riskProfile.Categories)

	if err != nil {
		return nil, err
	}

	mappedPolicyStatus, err := newPolicyStatus(policyStatus.OverallStatus, policyStatus.UpdatedAt, policyStatus.ComponentVersionStatusCounts)

	if err != nil {
		return nil, err
	}

	scan := ImageScan{
		RiskProfile:    *mappedRiskProfile,
		PolicyStatus:   *mappedPolicyStatus,
		ComponentsHref: componentsLink.Href,
		ScanSummary: ScanSummary{
			CreatedAt: scanSummary.CreatedAt,
			Status:    parseScanSummaryStatus(scanSummary.Status),
			UpdatedAt: scanSummary.UpdatedAt,
		},
		CodeLocationCreatedAt: codeLocation.CreatedAt,
		//CodeLocationMappedProjectVersion string
		CodeLocationName:      codeLocation.Name,
		CodeLocationType:      codeLocation.Type,
		CodeLocationURL:       codeLocation.URL,
		CodeLocationUpdatedAt: codeLocation.UpdatedAt,
	}

	return &scan, nil
}

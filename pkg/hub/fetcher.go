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

const (
	hubClientTimeout = 5 * time.Second
)

type Fetcher struct {
	client     hubclient.Client
	hubVersion string
	username   string
	password   string
	baseURL    string
}

func (hf *Fetcher) Login() error {
	err := hf.client.Login(hf.username, hf.password)
	return err
}

func (hf *Fetcher) fetchHubVersion() error {
	currentVersion, err := hf.client.CurrentVersion()
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
func NewFetcher(username string, password string, baseURL string) (*Fetcher, error) {
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
	projectList, err := hf.client.ListProjects(&hubapi.GetListOptions{Q: &queryString})

	recordHubResponse("projects", err == nil)

	if err != nil {
		log.Errorf("error fetching project list: %v", err)
		return nil, err
	}
	projects := projectList.Items
	if len(projects) == 0 {
		return nil, nil
	}
	if len(projects) > 1 {
		return nil, fmt.Errorf("expected 1 project matching name search string %s, found %d", image.HubProjectNameSearchString(), len(projects))
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
	versionList, err := client.ListProjectVersions(*link, &options)

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
		return nil, nil
	case 1:
		break // good to go, continue
	default:
		return nil, fmt.Errorf("expected to find one project version of name %s, found %d", image.HubProjectVersionNameSearchString(), len(versions))
	}

	version := versions[0]

	riskProfileLink, err := version.GetProjectVersionRiskProfileLink()
	if err != nil {
		log.Errorf("error getting risk profile link: %v", err)
		return nil, err
	}

	riskProfile, err := client.GetProjectVersionRiskProfile(*riskProfileLink)
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
	policyStatus, err := client.GetProjectVersionPolicyStatus(*policyStatusLink)
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
	codeLocationsList, err := client.ListCodeLocations(*codeLocationsLink)
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
		return nil, nil
	case 1:
		break // good to go, continue
	default:
		return nil, fmt.Errorf("expected to find one code location of name %s, found %d", image.HubScanNameSearchString(), len(codeLocations))
	}

	codeLocation := codeLocations[0]

	scanSummariesLink, err := codeLocation.GetScanSummariesLink()
	if err != nil {
		log.Errorf("error getting scan summaries link: %v", err)
		return nil, err
	}
	scanSummariesList, err := client.ListScanSummaries(*scanSummariesLink)
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
		return nil, nil
	case 1:
		break // good to go, continue
	default:
		return nil, fmt.Errorf("expected to find one scan summary for code location %s, found %d", image.HubScanNameSearchString(), len(scanSummariesList.Items))
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

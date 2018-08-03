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
	maxHubExponentialBackoffDuration = 1 * time.Hour
)

// Fetcher is a hub client which includes a circuit breaker.
// It does not provide rate limiting or concurrent job limiting.
type Fetcher struct {
	client         *hubclient.Client
	circuitBreaker *CircuitBreaker
	username       string
	password       string
	baseURL        string
}

// ResetCircuitBreaker ...
func (hf *Fetcher) ResetCircuitBreaker() {
	hf.circuitBreaker.Reset()
}

// Model ...
func (hf *Fetcher) Model() *FetcherModel {
	return &FetcherModel{
		ConsecutiveFailures: hf.circuitBreaker.ConsecutiveFailures,
		NextCheckTime:       hf.circuitBreaker.NextCheckTime,
		State:               hf.circuitBreaker.State,
	}
}

// IsEnabled returns whether the fetcher is currently enabled
// example: the circuit breaker is disabled -> the fetcher is disabled
func (hf *Fetcher) IsEnabled() <-chan bool {
	return hf.circuitBreaker.IsEnabledChannel
}

// Login ignores the circuit breaker, just in case the circuit breaker
// is closed because the calls were failing due to being unauthenticated.
// Or maybe TODO we need to distinguish between different types of
// request failure (network vs. 400 vs. 500 etc.)
// TODO could reset circuit breaker on success
func (hf *Fetcher) Login() error {
	start := time.Now()
	err := hf.client.Login(hf.username, hf.password)
	recordHubResponse("login", err == nil)
	recordHubResponseTime("login", time.Now().Sub(start))
	return err
}

// HubVersion fetches the hub version
func (hf *Fetcher) HubVersion() (string, error) {
	start := time.Now()
	currentVersion, err := hf.client.CurrentVersion()
	recordHubResponse("version", err == nil)
	recordHubResponseTime("version", time.Now().Sub(start))
	if err != nil {
		log.Errorf("unable to get hub version: %s", err.Error())
		return "", err
	}

	log.Infof("successfully got hub version %s", currentVersion.Version)
	return currentVersion.Version, nil
}

// NewFetcher returns a new Fetcher.
// It will not be logged in.
// It will return an error if: any of the following happen:
//  - unable to instantiate a Hub API client
func NewFetcher(username string, password string, hubHost string, hubPort int, hubClientTimeout time.Duration) (*Fetcher, error) {
	baseURL := fmt.Sprintf("https://%s:%d", hubHost, hubPort)
	client, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings, hubClientTimeout)
	if err != nil {
		return nil, err
	}
	hf := Fetcher{
		client:         client,
		circuitBreaker: NewCircuitBreaker(maxHubExponentialBackoffDuration),
		username:       username,
		password:       password,
		baseURL:        baseURL}
	return &hf, nil
}

// SetTimeout is currently not concurrent-safe, and should be made so TODO
func (hf *Fetcher) SetTimeout(timeout time.Duration) {
	hf.client.SetTimeout(timeout)
}

// DeleteScan deletes the code location and project version (but NOT the project)
// associated with the given scan name.
func (hf *Fetcher) DeleteScan(scanName string) error {
	clList, err := hf.ListCodeLocations(scanName)
	switch len(clList.Items) {
	case 0:
		return nil
	case 1:
		// continue
	default:
		return fmt.Errorf("expected 0 or 1 scans of name %s, found %d", scanName, len(clList.Items))
	}
	codeLocation := clList.Items[0]
	err = hf.DeleteCodeLocation(codeLocation.Meta.Href)
	if err != nil {
		return err
	}
	err = hf.DeleteProjectVersion(codeLocation.MappedProjectVersion)
	return err
}

// FetchScan finds ScanResults by starting from a code location,
// and following links from there.
// It returns:
//  - nil, if there's no code location with a matching name
//  - nil, if there's 0 scan summaries for the code location
//  - an error, if there were any HTTP problems or link problems
//  - an ScanResults, but possibly with garbage data, in all other cases
// Weird cases to watch out for:
//  - multiple code locations with a matching name
//  - multiple scan summaries for a code location
//  - zero scan summaries for a code location
func (hf *Fetcher) FetchScan(scanNameSearchString string) (*ScanResults, error) {
	codeLocationList, err := hf.ListCodeLocations(scanNameSearchString)

	if err != nil {
		log.Errorf("error fetching code location list: %v", err)
		return nil, err
	}
	codeLocations := codeLocationList.Items
	switch len(codeLocations) {
	case 0:
		recordHubData("codeLocations", true)
		return nil, nil
	case 1:
		recordHubData("codeLocations", true) // good to go
	default:
		recordHubData("codeLocations", false)
		log.Warnf("expected 1 code location matching name search string %s, found %d", scanNameSearchString, len(codeLocations))
	}

	codeLocation := codeLocations[0]
	return hf.fetchScanResultsUsingCodeLocation(codeLocation, scanNameSearchString)
}

func (hf *Fetcher) fetchScanResultsUsingCodeLocation(codeLocation hubapi.CodeLocation, scanNameSearchString string) (*ScanResults, error) {
	versionLink, err := codeLocation.GetProjectVersionLink()
	if err != nil {
		log.Errorf("unable to get project version link: %s", err.Error())
		return nil, err
	}

	version, err := hf.GetProjectVersion(*versionLink)
	if err != nil {
		log.Errorf("unable to fetch project version: %s", err.Error())
		return nil, err
	}

	riskProfileLink, err := version.GetProjectVersionRiskProfileLink()
	if err != nil {
		log.Errorf("error getting risk profile link: %v", err)
		return nil, err
	}

	riskProfile, err := hf.GetProjectVersionRiskProfile(*riskProfileLink)
	if err != nil {
		log.Errorf("error fetching project version risk profile: %v", err)
		return nil, err
	}

	policyStatusLink, err := version.GetProjectVersionPolicyStatusLink()
	if err != nil {
		log.Errorf("error getting policy status link: %v", err)
		return nil, err
	}
	policyStatus, err := hf.GetProjectVersionPolicyStatus(*policyStatusLink)
	if err != nil {
		log.Errorf("error fetching project version policy status: %v", err)
		return nil, err
	}

	componentsLink, err := version.GetComponentsLink()
	if err != nil {
		log.Errorf("error getting components link: %v", err)
		return nil, err
	}

	scanSummariesLink, err := codeLocation.GetScanSummariesLink()
	if err != nil {
		log.Errorf("error getting scan summaries link: %v", err)
		return nil, err
	}
	scanSummariesList, err := hf.ListScanSummaries(*scanSummariesLink)
	if err != nil {
		log.Errorf("error fetching scan summaries: %v", err)
		return nil, err
	}

	switch len(scanSummariesList.Items) {
	case 0:
		recordHubData("scan summaries", true)
		return nil, nil
	case 1:
		recordHubData("scan summaries", true) // good to go, continue
	default:
		recordHubData("scan summaries", false)
		log.Warnf("expected to find one scan summary for code location %s, found %d", scanNameSearchString, len(scanSummariesList.Items))
	}

	mappedRiskProfile, err := newRiskProfile(riskProfile.BomLastUpdatedAt, riskProfile.Categories)
	if err != nil {
		return nil, err
	}

	mappedPolicyStatus, err := newPolicyStatus(policyStatus.OverallStatus, policyStatus.UpdatedAt, policyStatus.ComponentVersionStatusCounts)
	if err != nil {
		return nil, err
	}

	scanSummaries := make([]ScanSummary, len(scanSummariesList.Items))
	for i, scanSummary := range scanSummariesList.Items {
		scanSummaries[i] = *NewScanSummaryFromHub(scanSummary)
	}

	scan := ScanResults{
		RiskProfile:           *mappedRiskProfile,
		PolicyStatus:          *mappedPolicyStatus,
		ComponentsHref:        componentsLink.Href,
		ScanSummaries:         scanSummaries,
		CodeLocationCreatedAt: codeLocation.CreatedAt,
		CodeLocationName:      codeLocation.Name,
		CodeLocationType:      codeLocation.Type,
		CodeLocationURL:       codeLocation.URL,
		CodeLocationUpdatedAt: codeLocation.UpdatedAt,
	}

	return &scan, nil
}

// "Raw" API calls

// ListAllProjects pulls in all projects in a single API call.
func (hf *Fetcher) ListAllProjects() (*hubapi.ProjectList, error) {
	var list *hubapi.ProjectList
	var fetchError error
	err := hf.circuitBreaker.IssueRequest("allProjects", func() error {
		limit := 2000000
		list, fetchError = hf.client.ListProjects(&hubapi.GetListOptions{Limit: &limit})
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return list, fetchError
}

// ListAllCodeLocations pulls in all code locations in a single API call.
func (hf *Fetcher) ListAllCodeLocations() (*hubapi.CodeLocationList, error) {
	var list *hubapi.CodeLocationList
	var fetchError error
	err := hf.circuitBreaker.IssueRequest("allCodeLocations", func() error {
		limit := 2000000
		list, fetchError = hf.client.ListAllCodeLocations(&hubapi.GetListOptions{Limit: &limit})
		if fetchError != nil {
			log.Errorf("fetch error: %s", fetchError.Error())
		}
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return list, fetchError
}

// ListCodeLocations ...
func (hf *Fetcher) ListCodeLocations(codeLocationName string) (*hubapi.CodeLocationList, error) {
	var list *hubapi.CodeLocationList
	var fetchError error
	err := hf.circuitBreaker.IssueRequest("codeLocations", func() error {
		queryString := fmt.Sprintf("name:%s", codeLocationName)
		list, fetchError = hf.client.ListAllCodeLocations(&hubapi.GetListOptions{Q: &queryString})
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return list, fetchError
}

// GetProjectVersion ...
func (hf *Fetcher) GetProjectVersion(link hubapi.ResourceLink) (*hubapi.ProjectVersion, error) {
	var pv *hubapi.ProjectVersion
	var fetchError error
	err := hf.circuitBreaker.IssueRequest("projectVersion", func() error {
		pv, fetchError = hf.client.GetProjectVersion(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return pv, fetchError
}

// GetProject ...
func (hf *Fetcher) GetProject(link hubapi.ResourceLink) (*hubapi.Project, error) {
	var val *hubapi.Project
	var fetchError error
	err := hf.circuitBreaker.IssueRequest("project", func() error {
		val, fetchError = hf.client.GetProject(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// GetProjectVersionRiskProfile ...
func (hf *Fetcher) GetProjectVersionRiskProfile(link hubapi.ResourceLink) (*hubapi.ProjectVersionRiskProfile, error) {
	var val *hubapi.ProjectVersionRiskProfile
	var fetchError error
	err := hf.circuitBreaker.IssueRequest("projectVersionRiskProfile", func() error {
		val, fetchError = hf.client.GetProjectVersionRiskProfile(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// GetProjectVersionPolicyStatus ...
func (hf *Fetcher) GetProjectVersionPolicyStatus(link hubapi.ResourceLink) (*hubapi.ProjectVersionPolicyStatus, error) {
	var val *hubapi.ProjectVersionPolicyStatus
	var fetchError error
	err := hf.circuitBreaker.IssueRequest("projectVersionPolicyStatus", func() error {
		val, fetchError = hf.client.GetProjectVersionPolicyStatus(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// ListScanSummaries ...
func (hf *Fetcher) ListScanSummaries(link hubapi.ResourceLink) (*hubapi.ScanSummaryList, error) {
	var val *hubapi.ScanSummaryList
	var fetchError error
	err := hf.circuitBreaker.IssueRequest("scanSummaries", func() error {
		val, fetchError = hf.client.ListScanSummaries(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// DeleteProjectVersion ...
func (hf *Fetcher) DeleteProjectVersion(projectVersionHRef string) error {
	var fetchError error
	err := hf.circuitBreaker.IssueRequest("deleteVersion", func() error {
		fetchError = hf.client.DeleteProjectVersion(projectVersionHRef)
		return fetchError
	})
	if err != nil {
		return err
	}
	return fetchError
}

// DeleteCodeLocation ...
func (hf *Fetcher) DeleteCodeLocation(codeLocationHRef string) error {
	var fetchError error
	err := hf.circuitBreaker.IssueRequest("deleteCodeLocation", func() error {
		fetchError = hf.client.DeleteCodeLocation(codeLocationHRef)
		return fetchError
	})
	if err != nil {
		return err
	}
	return fetchError
}

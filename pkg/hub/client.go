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
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
)

// Client combines a raw hub client with a circuit breaker
type Client struct {
	rawClient      RawClientInterface
	circuitBreaker *CircuitBreaker
	host           string
	username       string
	password       string
}

// NewClient returns a new Client.
func NewClient(username string, password string, host string, rawClient RawClientInterface) *Client {
	return &Client{
		rawClient:      rawClient,
		circuitBreaker: NewCircuitBreaker(host, maxHubExponentialBackoffDuration),
		username:       username,
		password:       password,
		host:           host,
	}
}

func (client *Client) resetCircuitBreaker() {
	client.circuitBreaker.Reset()
}

// Version fetches the hub version
func (client *Client) Version() (string, error) {
	start := time.Now()
	currentVersion, err := client.rawClient.CurrentVersion()
	recordHubResponse(client.host, "version", err == nil)
	recordHubResponseTime(client.host, "version", time.Now().Sub(start))
	if err != nil {
		log.Errorf("unable to get hub version: %s", err.Error())
		return "", errors.Trace(err)
	}

	log.Infof("successfully got hub version %s", currentVersion.Version)
	return currentVersion.Version, nil
}

// SetTimeout is currently not concurrent-safe, and should be made so TODO
func (client *Client) SetTimeout(timeout time.Duration) {
	client.rawClient.SetTimeout(timeout)
}

// login ignores the circuit breaker, just in case the circuit breaker
// is closed because the calls were failing due to being unauthenticated.
// Or maybe TODO we need to distinguish between different types of
// request failure (network vs. 400 vs. 500 etc.)
// TODO could reset circuit breaker on success
func (client *Client) login() error {
	start := time.Now()
	err := client.rawClient.Login(client.username, client.password)
	recordHubResponse(client.host, "login", err == nil)
	recordHubResponseTime(client.host, "login", time.Now().Sub(start))
	return errors.Trace(err)
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
func (client *Client) fetchScan(scanNameSearchString string) (*ScanResults, error) {
	codeLocationList, err := client.listCodeLocations(scanNameSearchString)

	if err != nil {
		recordError(client.host, "fetch code location list")
		log.Errorf("error fetching code location list: %v", err)
		return nil, errors.Trace(err)
	}
	codeLocations := codeLocationList.Items
	switch len(codeLocations) {
	case 0:
		recordHubData(client.host, "codeLocations", true)
		return nil, nil
	case 1:
		recordHubData(client.host, "codeLocations", true) // good to go
	default:
		recordHubData(client.host, "codeLocations", false)
		log.Warnf("expected 1 code location matching name search string %s, found %d", scanNameSearchString, len(codeLocations))
	}

	codeLocation := codeLocations[0]
	return client.fetchScanResultsUsingCodeLocation(codeLocation, scanNameSearchString)
}

func (client *Client) fetchScanResultsUsingCodeLocation(codeLocation hubapi.CodeLocation, scanNameSearchString string) (*ScanResults, error) {
	versionLink, err := codeLocation.GetProjectVersionLink()
	if err != nil {
		recordError(client.host, "get project version link")
		log.Errorf("unable to get project version link: %s", err.Error())
		return nil, err
	}

	version, err := client.getProjectVersion(*versionLink)
	if err != nil {
		recordError(client.host, "fetch project version")
		log.Errorf("unable to fetch project version: %s", err.Error())
		return nil, err
	}

	riskProfileLink, err := version.GetProjectVersionRiskProfileLink()
	if err != nil {
		recordError(client.host, "get risk profile link")
		log.Errorf("error getting risk profile link: %v", err)
		return nil, err
	}

	riskProfile, err := client.getProjectVersionRiskProfile(*riskProfileLink)
	if err != nil {
		recordError(client.host, "fetch project version risk profile")
		log.Errorf("error fetching project version risk profile: %v", err)
		return nil, err
	}

	policyStatusLink, err := version.GetProjectVersionPolicyStatusLink()
	if err != nil {
		recordError(client.host, "get policy status link")
		log.Errorf("error getting policy status link: %v", err)
		return nil, err
	}
	policyStatus, err := client.getProjectVersionPolicyStatus(*policyStatusLink)
	if err != nil {
		recordError(client.host, "fetch policy status")
		log.Errorf("error fetching project version policy status: %v", err)
		return nil, err
	}

	componentsLink, err := version.GetComponentsLink()
	if err != nil {
		recordError(client.host, "get components link")
		log.Errorf("error getting components link: %v", err)
		return nil, err
	}

	scanSummariesLink, err := codeLocation.GetScanSummariesLink()
	if err != nil {
		recordError(client.host, "get scan summaries link")
		log.Errorf("error getting scan summaries link: %v", err)
		return nil, err
	}
	scanSummariesList, err := client.listScanSummaries(*scanSummariesLink)
	if err != nil {
		recordError(client.host, "fetch scan summaries")
		log.Errorf("error fetching scan summaries: %v", err)
		return nil, err
	}

	switch len(scanSummariesList.Items) {
	case 0:
		recordHubData(client.host, "scan summaries", true)
		return nil, nil
	case 1:
		recordHubData(client.host, "scan summaries", true) // good to go, continue
	default:
		recordHubData(client.host, "scan summaries", false)
		log.Warnf("expected to find one scan summary for code location %s, found %d", scanNameSearchString, len(scanSummariesList.Items))
	}

	mappedRiskProfile, err := newRiskProfile(riskProfile.BomLastUpdatedAt, riskProfile.Categories)
	if err != nil {
		recordError(client.host, "map risk profile")
		return nil, err
	}

	mappedPolicyStatus, err := newPolicyStatus(policyStatus.OverallStatus, policyStatus.UpdatedAt, policyStatus.ComponentVersionStatusCounts)
	if err != nil {
		recordError(client.host, "map policy status")
		return nil, err
	}

	scanSummaries := make([]ScanSummary, len(scanSummariesList.Items))
	for i, scanSummary := range scanSummariesList.Items {
		scanSummaries[i] = *NewScanSummaryFromHub(scanSummary)
	}

	scan := ScanResults{
		RiskProfile:                      *mappedRiskProfile,
		PolicyStatus:                     *mappedPolicyStatus,
		ComponentsHref:                   componentsLink.Href,
		ScanSummaries:                    scanSummaries,
		CodeLocationCreatedAt:            codeLocation.CreatedAt,
		CodeLocationHref:                 codeLocation.Meta.Href,
		CodeLocationMappedProjectVersion: codeLocation.MappedProjectVersion,
		CodeLocationName:                 codeLocation.Name,
		CodeLocationType:                 codeLocation.Type,
		CodeLocationURL:                  codeLocation.URL,
		CodeLocationUpdatedAt:            codeLocation.UpdatedAt,
	}

	return &scan, nil
}

// "Raw" API calls

// listAllProjects pulls in all projects in a single API call.
func (client *Client) listAllProjects() (*hubapi.ProjectList, error) {
	var list *hubapi.ProjectList
	var fetchError error
	err := client.circuitBreaker.IssueRequest("allProjects", func() error {
		limit := 2000000
		list, fetchError = client.rawClient.ListProjects(&hubapi.GetListOptions{Limit: &limit})
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return list, fetchError
}

// ListAllCodeLocations pulls in all code locations in a single API call.
func (client *Client) listAllCodeLocations() (*hubapi.CodeLocationList, error) {
	var list *hubapi.CodeLocationList
	var fetchError error
	err := client.circuitBreaker.IssueRequest("allCodeLocations", func() error {
		limit := 2000000
		list, fetchError = client.rawClient.ListAllCodeLocations(&hubapi.GetListOptions{Limit: &limit})
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
func (client *Client) listCodeLocations(codeLocationName string) (*hubapi.CodeLocationList, error) {
	var list *hubapi.CodeLocationList
	var fetchError error
	err := client.circuitBreaker.IssueRequest("codeLocations", func() error {
		queryString := fmt.Sprintf("name:%s", codeLocationName)
		list, fetchError = client.rawClient.ListAllCodeLocations(&hubapi.GetListOptions{Q: &queryString})
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return list, fetchError
}

// GetProjectVersion ...
func (client *Client) getProjectVersion(link hubapi.ResourceLink) (*hubapi.ProjectVersion, error) {
	var pv *hubapi.ProjectVersion
	var fetchError error
	err := client.circuitBreaker.IssueRequest("projectVersion", func() error {
		pv, fetchError = client.rawClient.GetProjectVersion(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return pv, fetchError
}

// GetProject ...
func (client *Client) getProject(link hubapi.ResourceLink) (*hubapi.Project, error) {
	var val *hubapi.Project
	var fetchError error
	err := client.circuitBreaker.IssueRequest("project", func() error {
		val, fetchError = client.rawClient.GetProject(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// GetProjectVersionRiskProfile ...
func (client *Client) getProjectVersionRiskProfile(link hubapi.ResourceLink) (*hubapi.ProjectVersionRiskProfile, error) {
	var val *hubapi.ProjectVersionRiskProfile
	var fetchError error
	err := client.circuitBreaker.IssueRequest("projectVersionRiskProfile", func() error {
		val, fetchError = client.rawClient.GetProjectVersionRiskProfile(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// GetProjectVersionPolicyStatus ...
func (client *Client) getProjectVersionPolicyStatus(link hubapi.ResourceLink) (*hubapi.ProjectVersionPolicyStatus, error) {
	var val *hubapi.ProjectVersionPolicyStatus
	var fetchError error
	err := client.circuitBreaker.IssueRequest("projectVersionPolicyStatus", func() error {
		val, fetchError = client.rawClient.GetProjectVersionPolicyStatus(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// ListScanSummaries ...
func (client *Client) listScanSummaries(link hubapi.ResourceLink) (*hubapi.ScanSummaryList, error) {
	var val *hubapi.ScanSummaryList
	var fetchError error
	err := client.circuitBreaker.IssueRequest("scanSummaries", func() error {
		val, fetchError = client.rawClient.ListScanSummaries(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// DeleteProjectVersion ...
func (client *Client) deleteProjectVersion(projectVersionHRef string) error {
	var fetchError error
	err := client.circuitBreaker.IssueRequest("deleteVersion", func() error {
		fetchError = client.rawClient.DeleteProjectVersion(projectVersionHRef)
		return fetchError
	})
	if err != nil {
		return err
	}
	return fetchError
}

// DeleteCodeLocation ...
func (client *Client) deleteCodeLocation(codeLocationHRef string) error {
	var fetchError error
	err := client.circuitBreaker.IssueRequest("deleteCodeLocation", func() error {
		fetchError = client.rawClient.DeleteCodeLocation(codeLocationHRef)
		return fetchError
	})
	if err != nil {
		return err
	}
	return fetchError
}

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
	"github.com/blackducksoftware/perceptor/pkg/api"
	"github.com/blackducksoftware/perceptor/pkg/util"
	log "github.com/sirupsen/logrus"
)

const (
	maxHubExponentialBackoffDuration = 1 * time.Hour
	// hubDeleteTimeout                 = 1 * time.Hour
)

// Client .....
type Client struct {
	// TODO add a second hub client -- so that there's one for rare, slow requests (all projects,
	//   all code locations) and one for frequent, quick requests
	client         RawClientInterface
	circuitBreaker *CircuitBreaker
	// basic hub info
	username string
	password string
	host     string
	port     int
	status   ClientStatus
	// data
	codeLocations map[string]string
	projects      map[string]string
	errors        []error
	// TODO critical vulnerabilities
	// schedulers
	loginTimer              *util.Timer
	fetchProjectsTimer      *util.Timer
	fetchCodeLocationsTimer *util.Timer
	// channels
	stop                    chan struct{}
	resetCircuitBreakerCh   chan struct{}
	getModel                chan chan *api.HubModel
	getCodeLocationsCh      chan chan map[string]string
	getProjectsCh           chan chan map[string]string
	didLoginCh              chan error
	didFetchCodeLocationsCh chan *fetchCodeLocationsResult
	didFetchProjectsCh      chan *fetchProjectsResult
}

// NewClient returns a new, logged-in Client.
// It will not be logged in.
func NewClient(username string, password string, host string, port int, hubClientTimeout time.Duration, fetchAllProjectsPause time.Duration) *Client {
	hub := &Client{
		circuitBreaker: NewCircuitBreaker(maxHubExponentialBackoffDuration),
		username:       username,
		password:       password,
		host:           host,
		port:           port,
		status:         ClientStatusDown,
		//
		codeLocations: nil,
		projects:      nil,
		errors:        []error{},
		//
		stop: make(chan struct{}),
		resetCircuitBreakerCh:   make(chan struct{}),
		getModel:                make(chan chan *api.HubModel),
		getProjectsCh:           make(chan chan map[string]string),
		getCodeLocationsCh:      make(chan chan map[string]string),
		didLoginCh:              make(chan error),
		didFetchCodeLocationsCh: make(chan *fetchCodeLocationsResult),
		didFetchProjectsCh:      make(chan *fetchProjectsResult)}
	// initialize hub client
	baseURL := fmt.Sprintf("https://%s:%d", host, port)
	client, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings, hubClientTimeout)
	if err != nil {
		hub.status = ClientStatusError
	}
	hub.client = client
	// action processing
	go func() {
		for {
			select {
			case <-hub.stop:
				return
			case <-hub.resetCircuitBreakerCh:
				log.Warnf("resetting circuit breaker is currently disabled: ignoring")
				// TODO hub.circuitBreaker.Reset()
			case ch := <-hub.getModel:
				ch <- hub.apiModel()
			case ch := <-hub.getProjectsCh:
				ch <- hub.projects
			case ch := <-hub.getCodeLocationsCh:
				ch <- hub.codeLocations
			case result := <-hub.didFetchProjectsCh:
				hub.recordError(result.err)
				if result.err == nil {
					hub.projects = result.projects
				}
			case result := <-hub.didFetchCodeLocationsCh:
				hub.recordError(result.err)
				if result.err == nil {
					hub.codeLocations = result.codeLocations
				}
			case err := <-hub.didLoginCh:
				hub.recordError(err)
				if err != nil && hub.status == ClientStatusUp {
					hub.status = ClientStatusDown
					hub.recordError(hub.fetchProjectsTimer.Pause())
					hub.recordError(hub.fetchCodeLocationsTimer.Pause())
				} else if err == nil && hub.status == ClientStatusDown {
					hub.status = ClientStatusUp
					hub.recordError(hub.fetchProjectsTimer.Resume(true))
					hub.recordError(hub.fetchCodeLocationsTimer.Resume(true))
				}
			}
		}
	}()
	hub.fetchProjectsTimer = hub.startFetchProjectsTimer(fetchAllProjectsPause)
	hub.fetchCodeLocationsTimer = hub.startFetchCodeLocationsTimer(fetchAllProjectsPause)
	hub.loginTimer = hub.startLoginTimer()
	return hub
}

// Stop ...
func (hub *Client) Stop() {
	close(hub.stop)
}

// Host ...
func (hub *Client) Host() string {
	return hub.host
}

// // ResetCircuitBreaker ...
// func (hub *Client) ResetCircuitBreaker() {
//   hub.resetCircuitBreakerCh <- struct{}
// }
//
// // IsEnabled returns whether the Client is currently enabled
// // example: the circuit breaker is disabled -> the Client is disabled
// func (hub *Client) IsEnabled() <-chan bool {
// 	return hub.circuitBreaker.IsEnabledChannel
// }

// Model ...
func (hub *Client) Model() *api.HubModel {
	ch := make(chan *api.HubModel)
	hub.getModel <- ch
	return <-ch
}

// Private methods

func (hub *Client) recordError(err error) {
	if err != nil {
		hub.errors = append(hub.errors, err)
	}
	if len(hub.errors) > 1000 {
		hub.errors = hub.errors[500:]
	}
}

func (hub *Client) login() error {
	return hub.Login()
}

func (hub *Client) apiModel() *api.HubModel {
	errors := make([]string, len(hub.errors))
	for ix, err := range hub.errors {
		errors[ix] = err.Error()
	}
	projects := map[string]string{}
	for name, url := range hub.projects {
		projects[name] = url
	}
	codeLocations := map[string]string{}
	for name, url := range hub.codeLocations {
		codeLocations[name] = url
	}
	return &api.HubModel{
		Errors:                  errors,
		HasLoadedAllProjects:    hub.projects != nil,
		Status:                  hub.status.String(),
		IsCircuitBreakerEnabled: false, // TODO
		IsLoggedIn:              false, // TODO
		Projects:                projects,
		CodeLocations:           codeLocations,
	}
}

// Regular jobs

func (hub *Client) startLoginTimer() *util.Timer {
	pause := 30 * time.Second // Minute
	name := fmt.Sprintf("login-%s", hub.host)
	return util.NewRunningTimer(name, pause, hub.stop, true, func() {
		log.Debugf("starting to login to hub")
		err := hub.login()
		select {
		case hub.didLoginCh <- err:
		case <-hub.stop:
		}
	})
}

func (hub *Client) startFetchProjectsTimer(pause time.Duration) *util.Timer {
	name := fmt.Sprintf("fetchProjects-%s", hub.host)
	return util.NewTimer(name, pause, hub.stop, func() {
		log.Debugf("starting to fetch all projects")
		result := hub.fetchAllProjects()
		select {
		case hub.didFetchProjectsCh <- result:
		case <-hub.stop: // TODO should cancel when this happens
		}
	})
}

func (hub *Client) startFetchCodeLocationsTimer(pause time.Duration) *util.Timer {
	name := fmt.Sprintf("fetchCodeLocations-%s", hub.host)
	return util.NewTimer(name, pause, hub.stop, func() {
		log.Debugf("starting to fetch all code locations")
		result := hub.fetchAllCodeLocations()
		select {
		case hub.didFetchCodeLocationsCh <- result:
		case <-hub.stop:
		}
	})
}

// Hub api calls

type fetchCodeLocationsResult struct {
	codeLocations map[string]string
	err           error
}

func (hub *Client) fetchAllCodeLocations() *fetchCodeLocationsResult {
	codeLocationList, err := hub.ListAllCodeLocations()
	if err != nil {
		return &fetchCodeLocationsResult{codeLocations: nil, err: err}
	}
	log.Debugf("fetched all code locations: found %d, expected %d", len(codeLocationList.Items), codeLocationList.TotalCount)
	cls := map[string]string{}
	for _, cl := range codeLocationList.Items {
		cls[cl.Name] = cl.MappedProjectVersion
	}
	return &fetchCodeLocationsResult{codeLocations: cls, err: nil}
}

type fetchProjectsResult struct {
	projects map[string]string
	err      error
}

func (hub *Client) fetchAllProjects() *fetchProjectsResult {
	projectList, err := hub.ListAllProjects()
	if err != nil {
		return &fetchProjectsResult{projects: nil, err: err}
	}
	log.Debugf("fetched all projects: found %d, expected %d", len(projectList.Items), projectList.TotalCount)
	projects := map[string]string{}
	for _, proj := range projectList.Items {
		projects[proj.Name] = proj.Meta.Href
	}
	return &fetchProjectsResult{projects: projects, err: nil}
}

// Stuff from old Fetcher ... TODO clean up comments

// ResetCircuitBreaker ...
func (hub *Client) ResetCircuitBreaker() {
	hub.circuitBreaker.Reset()
}

// IsEnabled returns whether the fetcher is currently enabled
// example: the circuit breaker is disabled -> the fetcher is disabled
// func (hub *Client) IsEnabled() <-chan bool {
// 	return hub.circuitBreaker.IsEnabledChannel
// }

// Login ignores the circuit breaker, just in case the circuit breaker
// is closed because the calls were failing due to being unauthenticated.
// Or maybe TODO we need to distinguish between different types of
// request failure (network vs. 400 vs. 500 etc.)
// TODO could reset circuit breaker on success
func (hub *Client) Login() error {
	start := time.Now()
	err := hub.client.Login(hub.username, hub.password)
	recordHubResponse("login", err == nil)
	recordHubResponseTime("login", time.Now().Sub(start))
	return err
}

// Version fetches the hub version
func (hub *Client) Version() (string, error) {
	start := time.Now()
	currentVersion, err := hub.client.CurrentVersion()
	recordHubResponse("version", err == nil)
	recordHubResponseTime("version", time.Now().Sub(start))
	if err != nil {
		log.Errorf("unable to get hub version: %s", err.Error())
		return "", err
	}

	log.Infof("successfully got hub version %s", currentVersion.Version)
	return currentVersion.Version, nil
}

// SetTimeout is currently not concurrent-safe, and should be made so TODO
func (hub *Client) SetTimeout(timeout time.Duration) {
	hub.client.SetTimeout(timeout)
}

// DeleteScan deletes the code location and project version (but NOT the project)
// associated with the given scan name.
func (hub *Client) DeleteScan(scanName string) error {
	clList, err := hub.ListCodeLocations(scanName)
	switch len(clList.Items) {
	case 0:
		return nil
	case 1:
		// continue
	default:
		return fmt.Errorf("expected 0 or 1 scans of name %s, found %d", scanName, len(clList.Items))
	}
	codeLocation := clList.Items[0]
	err = hub.DeleteCodeLocation(codeLocation.Meta.Href)
	if err != nil {
		return err
	}
	err = hub.DeleteProjectVersion(codeLocation.MappedProjectVersion)
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
func (hub *Client) FetchScan(scanNameSearchString string) (*ScanResults, error) {
	codeLocationList, err := hub.ListCodeLocations(scanNameSearchString)

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
	return hub.fetchScanResultsUsingCodeLocation(codeLocation, scanNameSearchString)
}

func (hub *Client) fetchScanResultsUsingCodeLocation(codeLocation hubapi.CodeLocation, scanNameSearchString string) (*ScanResults, error) {
	versionLink, err := codeLocation.GetProjectVersionLink()
	if err != nil {
		log.Errorf("unable to get project version link: %s", err.Error())
		return nil, err
	}

	version, err := hub.GetProjectVersion(*versionLink)
	if err != nil {
		log.Errorf("unable to fetch project version: %s", err.Error())
		return nil, err
	}

	riskProfileLink, err := version.GetProjectVersionRiskProfileLink()
	if err != nil {
		log.Errorf("error getting risk profile link: %v", err)
		return nil, err
	}

	riskProfile, err := hub.GetProjectVersionRiskProfile(*riskProfileLink)
	if err != nil {
		log.Errorf("error fetching project version risk profile: %v", err)
		return nil, err
	}

	policyStatusLink, err := version.GetProjectVersionPolicyStatusLink()
	if err != nil {
		log.Errorf("error getting policy status link: %v", err)
		return nil, err
	}
	policyStatus, err := hub.GetProjectVersionPolicyStatus(*policyStatusLink)
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
	scanSummariesList, err := hub.ListScanSummaries(*scanSummariesLink)
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
func (hub *Client) ListAllProjects() (*hubapi.ProjectList, error) {
	var list *hubapi.ProjectList
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("allProjects", func() error {
		limit := 2000000
		list, fetchError = hub.client.ListProjects(&hubapi.GetListOptions{Limit: &limit})
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return list, fetchError
}

// ListAllCodeLocations pulls in all code locations in a single API call.
func (hub *Client) ListAllCodeLocations() (*hubapi.CodeLocationList, error) {
	var list *hubapi.CodeLocationList
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("allCodeLocations", func() error {
		limit := 2000000
		list, fetchError = hub.client.ListAllCodeLocations(&hubapi.GetListOptions{Limit: &limit})
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
func (hub *Client) ListCodeLocations(codeLocationName string) (*hubapi.CodeLocationList, error) {
	var list *hubapi.CodeLocationList
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("codeLocations", func() error {
		queryString := fmt.Sprintf("name:%s", codeLocationName)
		list, fetchError = hub.client.ListAllCodeLocations(&hubapi.GetListOptions{Q: &queryString})
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return list, fetchError
}

// GetProjectVersion ...
func (hub *Client) GetProjectVersion(link hubapi.ResourceLink) (*hubapi.ProjectVersion, error) {
	var pv *hubapi.ProjectVersion
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("projectVersion", func() error {
		pv, fetchError = hub.client.GetProjectVersion(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return pv, fetchError
}

// GetProject ...
func (hub *Client) GetProject(link hubapi.ResourceLink) (*hubapi.Project, error) {
	var val *hubapi.Project
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("project", func() error {
		val, fetchError = hub.client.GetProject(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// GetProjectVersionRiskProfile ...
func (hub *Client) GetProjectVersionRiskProfile(link hubapi.ResourceLink) (*hubapi.ProjectVersionRiskProfile, error) {
	var val *hubapi.ProjectVersionRiskProfile
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("projectVersionRiskProfile", func() error {
		val, fetchError = hub.client.GetProjectVersionRiskProfile(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// GetProjectVersionPolicyStatus ...
func (hub *Client) GetProjectVersionPolicyStatus(link hubapi.ResourceLink) (*hubapi.ProjectVersionPolicyStatus, error) {
	var val *hubapi.ProjectVersionPolicyStatus
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("projectVersionPolicyStatus", func() error {
		val, fetchError = hub.client.GetProjectVersionPolicyStatus(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// ListScanSummaries ...
func (hub *Client) ListScanSummaries(link hubapi.ResourceLink) (*hubapi.ScanSummaryList, error) {
	var val *hubapi.ScanSummaryList
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("scanSummaries", func() error {
		val, fetchError = hub.client.ListScanSummaries(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// DeleteProjectVersion ...
func (hub *Client) DeleteProjectVersion(projectVersionHRef string) error {
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("deleteVersion", func() error {
		fetchError = hub.client.DeleteProjectVersion(projectVersionHRef)
		return fetchError
	})
	if err != nil {
		return err
	}
	return fetchError
}

// DeleteCodeLocation ...
func (hub *Client) DeleteCodeLocation(codeLocationHRef string) error {
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("deleteCodeLocation", func() error {
		fetchError = hub.client.DeleteCodeLocation(codeLocationHRef)
		return fetchError
	})
	if err != nil {
		return err
	}
	return fetchError
}

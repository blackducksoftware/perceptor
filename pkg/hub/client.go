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
	client         RawClientInterface
	circuitBreaker *CircuitBreaker
	// basic hub info
	username string
	password string
	host     string
	status   ClientStatus
	// data
	hasFetchedCodeLocations bool
	codeLocations           map[string]*Scan
	errors                  []error
	// timers
	loginTimer                   *util.Timer
	fetchAllCodeLocationsTimer   *util.Timer
	fetchScansTimer              *util.Timer
	checkScansForCompletionTimer *util.Timer
	// public channels
	publishUpdatesCh chan Update
	// channels
	stop                      chan struct{}
	resetCircuitBreakerCh     chan struct{}
	getModel                  chan chan *api.ModelHub
	deleteScanCh              chan string
	didDeleteScanCh           chan *Result
	didLoginCh                chan error
	startScanClientCh         chan string
	finishScanClientCh        chan string
	getScanResultsCh          chan chan map[string]*ScanResults
	scanDidFinishCh           chan *ScanResults
	getCodeLocationsCountCh   chan chan int
	getInProgressScansCh      chan chan []string
	didFetchCodeLocationsCh   chan *Result
	didFetchScanResultsCh     chan *ScanResults
	hasFetchedCodeLocationsCh chan chan bool
	getCodeLocationsCh        chan chan map[string]ScanStage
	unknownCodeLocationsCh    chan chan []string
}

// NewClient returns a new Client.  It will not be logged in.
func NewClient(username string, password string, host string, client RawClientInterface, fetchUnknownScansPause time.Duration, fetchAllScansPause time.Duration) *Client {
	hub := &Client{
		client:         client,
		circuitBreaker: NewCircuitBreaker(maxHubExponentialBackoffDuration),
		username:       username,
		password:       password,
		host:           host,
		status:         ClientStatusDown,
		//
		hasFetchedCodeLocations: false,
		codeLocations:           map[string]*Scan{},
		errors:                  []error{},
		//
		publishUpdatesCh: make(chan Update),
		//
		stop:                      make(chan struct{}),
		resetCircuitBreakerCh:     make(chan struct{}),
		getModel:                  make(chan chan *api.ModelHub),
		deleteScanCh:              make(chan string),
		didDeleteScanCh:           make(chan *Result),
		didLoginCh:                make(chan error),
		startScanClientCh:         make(chan string),
		finishScanClientCh:        make(chan string),
		getScanResultsCh:          make(chan chan map[string]*ScanResults),
		scanDidFinishCh:           make(chan *ScanResults),
		getCodeLocationsCountCh:   make(chan chan int),
		getInProgressScansCh:      make(chan chan []string),
		didFetchCodeLocationsCh:   make(chan *Result),
		didFetchScanResultsCh:     make(chan *ScanResults),
		hasFetchedCodeLocationsCh: make(chan chan bool),
		getCodeLocationsCh:        make(chan chan map[string]ScanStage),
		unknownCodeLocationsCh:    make(chan chan []string)}
	// action processing
	go func() {
		for {
			select {
			case <-hub.stop:
				return
			case <-hub.resetCircuitBreakerCh:
				hub.circuitBreaker.Reset()
			case ch := <-hub.getModel:
				ch <- hub.apiModel()
			case ch := <-hub.unknownCodeLocationsCh:
				unknownCodeLocations := []string{}
				for name, scan := range hub.codeLocations {
					if scan.ScanResults == nil {
						unknownCodeLocations = append(unknownCodeLocations, name)
					}
				}
				ch <- unknownCodeLocations
			case ch := <-hub.getScanResultsCh:
				allScanResults := map[string]*ScanResults{}
				for name, scan := range hub.codeLocations {
					allScanResults[name] = scan.ScanResults
				}
				ch <- allScanResults
			case scanResults := <-hub.didFetchScanResultsCh:
				scan, ok := hub.codeLocations[scanResults.CodeLocationName]
				if !ok {
					scan = &Scan{
						ScanResults: scanResults,
						Stage:       ScanStageUnknown,
					}
					hub.codeLocations[scanResults.CodeLocationName] = scan
				}
				switch scanResults.ScanSummaryStatus() {
				case ScanSummaryStatusSuccess:
					scan.Stage = ScanStageComplete
				case ScanSummaryStatusInProgress:
					// TODO any way to distinguish between scanclient and hubscan?
					scan.Stage = ScanStageHubScan
				case ScanSummaryStatusFailure:
					// TODO add a failure state?
					scan.Stage = ScanStageUnknown
				}
				hub.codeLocations[scanResults.CodeLocationName].ScanResults = scanResults
				update := &DidFindScan{Name: scanResults.CodeLocationName, Results: scanResults}
				hub.publish(update)
			case scanName := <-hub.deleteScanCh:
				hub.recordError(hub.deleteScanAndProjectVersion(scanName))
			case result := <-hub.didDeleteScanCh:
				hub.recordError(result.Err)
				if result.Err == nil {
					scanName := result.Value.(string)
					delete(hub.codeLocations, scanName)
				}
			case result := <-hub.didFetchCodeLocationsCh:
				hub.recordError(result.Err)
				if result.Err == nil {
					hub.hasFetchedCodeLocations = true
					cls := result.Value.([]hubapi.CodeLocation)
					for _, cl := range cls {
						if _, ok := hub.codeLocations[cl.Name]; !ok {
							hub.codeLocations[cl.Name] = &Scan{Stage: ScanStageUnknown, ScanResults: nil}
						}
					}
				}
			case scanName := <-hub.startScanClientCh:
				hub.codeLocations[scanName] = &Scan{Stage: ScanStageScanClient}
			case scanName := <-hub.finishScanClientCh:
				scan, ok := hub.codeLocations[scanName]
				if !ok {
					log.Errorf("unable to handle finishScanClient for %s: not found", scanName)
					break
				}
				if scan.Stage != ScanStageScanClient {
					log.Warnf("unable to handle finishScanClient for %s: expected stage ScanClient, found %s", scanName, scan.Stage.String())
					break
				}
				scan.Stage = ScanStageHubScan
			case sr := <-hub.scanDidFinishCh:
				scanName := sr.CodeLocationName
				scan, ok := hub.codeLocations[scanName]
				if !ok {
					log.Errorf("unable to handle scanDidFinish for %s: not found", scanName)
					break
				}
				if scan.Stage != ScanStageHubScan {
					log.Warnf("unable to handle scanDidFinish for %s: expected stage HubScan, found %s", scanName, scan.Stage.String())
					break
				}
				scan.Stage = ScanStageComplete
				update := &DidFinishScan{Name: sr.CodeLocationName, Results: sr}
				hub.publish(update)
			case get := <-hub.getCodeLocationsCountCh:
				get <- len(hub.codeLocations)
			case get := <-hub.getInProgressScansCh:
				scans := []string{}
				for scanName, scan := range hub.codeLocations {
					if scan.Stage == ScanStageHubScan || scan.Stage == ScanStageScanClient {
						scans = append(scans, scanName)
					}
				}
				get <- scans
			case ch := <-hub.hasFetchedCodeLocationsCh:
				ch <- hub.hasFetchedCodeLocations
			case ch := <-hub.getCodeLocationsCh:
				codeLocations := map[string]ScanStage{}
				for scanName, scan := range hub.codeLocations {
					codeLocations[scanName] = scan.Stage
				}
				log.Debugf("handle getCodeLocations: found codelocations: %+v", codeLocations)
				ch <- codeLocations
			case err := <-hub.didLoginCh:
				hub.recordError(err)
				if err != nil && hub.status == ClientStatusUp {
					hub.status = ClientStatusDown
					hub.recordError(hub.checkScansForCompletionTimer.Pause())
					hub.recordError(hub.fetchScansTimer.Pause())
					hub.recordError(hub.fetchAllCodeLocationsTimer.Pause())
				} else if err == nil && hub.status == ClientStatusDown {
					hub.status = ClientStatusUp
					hub.recordError(hub.checkScansForCompletionTimer.Resume(true))
					hub.recordError(hub.fetchScansTimer.Resume(true))
					hub.recordError(hub.fetchAllCodeLocationsTimer.Resume(true))
				}
			}
		}
	}()
	hub.checkScansForCompletionTimer = hub.startCheckScansForCompletionTimer(1 * time.Minute)
	hub.fetchScansTimer = hub.startFetchUnknownScansTimer(fetchUnknownScansPause)
	hub.fetchAllCodeLocationsTimer = hub.startFetchAllCodeLocationsTimer(fetchAllScansPause)
	hub.loginTimer = hub.startLoginTimer(30 * time.Minute)
	return hub
}

func (hub *Client) publish(update Update) {
	// TODO also handle scan refreshes
	go func() {
		select {
		case <-hub.stop:
			return
		case hub.publishUpdatesCh <- update:
		}
	}()
}

// Stop ...
func (hub *Client) Stop() {
	close(hub.stop)
}

// StopCh returns a reference to the stop channel
func (hub *Client) StopCh() <-chan struct{} {
	return hub.stop
}

// Host ...
func (hub *Client) Host() string {
	return hub.host
}

// ResetCircuitBreaker ...
func (hub *Client) ResetCircuitBreaker() {
	hub.resetCircuitBreakerCh <- struct{}{}
}

// Model ...
func (hub *Client) Model() <-chan *api.ModelHub {
	ch := make(chan *api.ModelHub)
	hub.getModel <- ch
	return ch
}

// CodeLocations ...
func (hub *Client) CodeLocations() <-chan map[string]ScanStage {
	ch := make(chan map[string]ScanStage)
	hub.getCodeLocationsCh <- ch
	return ch
}

// HasFetchedCodeLocations ...
func (hub *Client) HasFetchedCodeLocations() <-chan bool {
	ch := make(chan bool)
	hub.hasFetchedCodeLocationsCh <- ch
	return ch
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

// login ignores the circuit breaker, just in case the circuit breaker
// is closed because the calls were failing due to being unauthenticated.
// Or maybe TODO we need to distinguish between different types of
// request failure (network vs. 400 vs. 500 etc.)
// TODO could reset circuit breaker on success
func (hub *Client) login() error {
	start := time.Now()
	err := hub.client.Login(hub.username, hub.password)
	recordHubResponse("login", err == nil)
	recordHubResponseTime("login", time.Now().Sub(start))
	return err
}

func (hub *Client) apiModel() *api.ModelHub {
	errors := make([]string, len(hub.errors))
	for ix, err := range hub.errors {
		errors[ix] = err.Error()
	}
	codeLocations := map[string]*api.ModelCodeLocation{}
	for name, scan := range hub.codeLocations {
		cl := &api.ModelCodeLocation{Stage: scan.Stage.String()}
		sr := scan.ScanResults
		if sr != nil {
			cl.Href = sr.CodeLocationHref
			cl.URL = sr.CodeLocationURL
			cl.MappedProjectVersion = sr.CodeLocationMappedProjectVersion
			cl.UpdatedAt = sr.CodeLocationUpdatedAt
			cl.ComponentsHref = sr.ComponentsHref
		}
		codeLocations[name] = cl
	}
	return &api.ModelHub{
		Errors:                    errors,
		Status:                    hub.status.String(),
		HasLoadedAllCodeLocations: hub.codeLocations != nil,
		CodeLocations:             codeLocations,
		CircuitBreaker:            hub.circuitBreaker.Model(),
		Host:                      hub.host,
	}
}

// Regular jobs

func (hub *Client) startLoginTimer(pause time.Duration) *util.Timer {
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

func (hub *Client) startFetchAllCodeLocationsTimer(pause time.Duration) *util.Timer {
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

func (hub *Client) unknownCodeLocations() []string {
	ch := make(chan []string)
	hub.unknownCodeLocationsCh <- ch
	return <-ch
}

func (hub *Client) startFetchUnknownScansTimer(pause time.Duration) *util.Timer {
	name := fmt.Sprintf("fetchUnknownScans-%s", hub.host)
	return util.NewTimer(name, pause, hub.stop, func() {
		log.Debugf("starting to fetch unknown scans")
		unknownCodeLocations := hub.unknownCodeLocations()
		log.Debugf("found %d unknown code locations", len(unknownCodeLocations))
		for _, codeLocationName := range unknownCodeLocations {
			scanResults, err := hub.fetchScan(codeLocationName)
			log.Debugf("fetched scan %s: %+v", codeLocationName, err)
			if err != nil {
				log.Error(err.Error())
				continue
			}
			select {
			case hub.didFetchScanResultsCh <- scanResults:
			case <-hub.stop:
				return
			}
		}
		log.Debugf("finished fetching unknown scans")
	})
}

func (hub *Client) startCheckScansForCompletionTimer(pause time.Duration) *util.Timer {
	name := fmt.Sprintf("checkScansForCompletion-%s", hub.host)
	return util.NewTimer(name, pause, hub.stop, func() {
		log.Debugf("starting to check scans for completion")
		var scanNames []string
		select {
		case scanNames = <-hub.InProgressScans():
		case <-hub.stop:
			return
		}
		for _, scanName := range scanNames {
			scanResults, err := hub.fetchScan(scanName)
			if err != nil {
				log.Errorf("unable to fetch scan %s: %s", scanName, err.Error())
				continue
			}
			if scanResults == nil {
				log.Debugf("nothing found for scan %s", scanName)
				continue
			}
			switch scanResults.ScanSummaryStatus() {
			case ScanSummaryStatusInProgress:
				// nothing to do
			case ScanSummaryStatusFailure, ScanSummaryStatusSuccess:
				select {
				case hub.scanDidFinishCh <- scanResults:
				case <-hub.stop:
					return
				}
			}
		}
	})
}

// Hub api calls

func (hub *Client) fetchAllCodeLocations() *Result {
	codeLocationList, err := hub.listAllCodeLocations()
	if err != nil {
		return &Result{Value: nil, Err: err}
	}
	log.Debugf("fetched all code locations: found %d, expected %d", len(codeLocationList.Items), codeLocationList.TotalCount)
	return &Result{Value: codeLocationList.Items, Err: nil}
}

// IsEnabled returns whether the fetcher is currently enabled
// example: the circuit breaker is disabled -> the fetcher is disabled
// func (hub *Client) IsEnabled() <-chan bool {
// 	return hub.circuitBreaker.IsEnabledChannel
// }

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
func (hub *Client) DeleteScan(scanName string) {
	hub.deleteScanCh <- scanName
}

func (hub *Client) deleteScanAndProjectVersion(scanName string) error {
	scan, ok := hub.codeLocations[scanName]
	if !ok {
		return fmt.Errorf("unable to delete scan %s, not found", scanName)
	}
	clURL := scan.ScanResults.CodeLocationHref
	projectVersionURL := scan.ScanResults.CodeLocationMappedProjectVersion
	finish := func(err error) {
		select {
		case hub.didDeleteScanCh <- &Result{Value: scanName, Err: err}:
		case <-hub.stop:
		}
	}
	go func() {
		err := hub.deleteCodeLocation(clURL)
		if err != nil {
			finish(err)
			return
		}
		finish(hub.deleteProjectVersion(projectVersionURL))
	}()
	return nil
}

// StartScanClient ...
func (hub *Client) StartScanClient(scanName string) {
	hub.startScanClientCh <- scanName
}

// FinishScanClient ...
func (hub *Client) FinishScanClient(scanName string) {
	hub.finishScanClientCh <- scanName
}

// CodeLocationsCount ...
func (hub *Client) CodeLocationsCount() <-chan int {
	ch := make(chan int)
	hub.getCodeLocationsCountCh <- ch
	return ch
}

// InProgressScans ...
func (hub *Client) InProgressScans() <-chan []string {
	ch := make(chan []string)
	hub.getInProgressScansCh <- ch
	return ch
}

// ScanResults ...
func (hub *Client) ScanResults() <-chan map[string]*ScanResults {
	ch := make(chan map[string]*ScanResults)
	hub.getScanResultsCh <- ch
	return ch
}

// Updates produces events for:
// - finding a scan for the first time
// - when a hub scan finishes
// - when a finished scan is repulled (to get any changes to its vulnerabilities, policies, etc.)
func (hub *Client) Updates() <-chan Update {
	return hub.publishUpdatesCh
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
func (hub *Client) fetchScan(scanNameSearchString string) (*ScanResults, error) {
	codeLocationList, err := hub.listCodeLocations(scanNameSearchString)

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

	version, err := hub.getProjectVersion(*versionLink)
	if err != nil {
		log.Errorf("unable to fetch project version: %s", err.Error())
		return nil, err
	}

	riskProfileLink, err := version.GetProjectVersionRiskProfileLink()
	if err != nil {
		log.Errorf("error getting risk profile link: %v", err)
		return nil, err
	}

	riskProfile, err := hub.getProjectVersionRiskProfile(*riskProfileLink)
	if err != nil {
		log.Errorf("error fetching project version risk profile: %v", err)
		return nil, err
	}

	policyStatusLink, err := version.GetProjectVersionPolicyStatusLink()
	if err != nil {
		log.Errorf("error getting policy status link: %v", err)
		return nil, err
	}
	policyStatus, err := hub.getProjectVersionPolicyStatus(*policyStatusLink)
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
	scanSummariesList, err := hub.listScanSummaries(*scanSummariesLink)
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

// ListAllProjects pulls in all projects in a single API call.
func (hub *Client) listAllProjects() (*hubapi.ProjectList, error) {
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
func (hub *Client) listAllCodeLocations() (*hubapi.CodeLocationList, error) {
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
func (hub *Client) listCodeLocations(codeLocationName string) (*hubapi.CodeLocationList, error) {
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
func (hub *Client) getProjectVersion(link hubapi.ResourceLink) (*hubapi.ProjectVersion, error) {
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
func (hub *Client) getProject(link hubapi.ResourceLink) (*hubapi.Project, error) {
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
func (hub *Client) getProjectVersionRiskProfile(link hubapi.ResourceLink) (*hubapi.ProjectVersionRiskProfile, error) {
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
func (hub *Client) getProjectVersionPolicyStatus(link hubapi.ResourceLink) (*hubapi.ProjectVersionPolicyStatus, error) {
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
func (hub *Client) listScanSummaries(link hubapi.ResourceLink) (*hubapi.ScanSummaryList, error) {
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
func (hub *Client) deleteProjectVersion(projectVersionHRef string) error {
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
func (hub *Client) deleteCodeLocation(codeLocationHRef string) error {
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

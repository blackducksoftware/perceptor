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
)

type finishScanClient struct {
	scanName string
	err      error
}

type clientAction struct {
	name  string
	apply func() error
}

// Hub .....
type Hub struct {
	client         RawClientInterface
	circuitBreaker *CircuitBreaker
	// basic hub info
	username string
	password string
	host     string
	status   ClientStatus
	// data
	hasFetchedScans bool
	scans           map[string]*Scan
	errors          []error
	// timers
	getMetricsTimer              *util.Timer
	loginTimer                   *util.Timer
	refreshScansTimer            *util.Timer
	fetchAllScansTimer           *util.Timer
	fetchScansTimer              *util.Timer
	checkScansForCompletionTimer *util.Timer
	// public channels
	publishUpdatesCh chan Update
	// channels
	stop                    chan struct{}
	actions                 chan *clientAction
	resetCircuitBreakerCh   chan struct{}
	getModel                chan chan *api.ModelHub
	didLoginCh              chan error
	startScanClientCh       chan string
	finishScanClientCh      chan *finishScanClient
	getScanResultsCh        chan chan map[string]*Scan
	scanDidFinishCh         chan *ScanResults
	getScansCountCh         chan chan int
	getInProgressScansCh    chan chan []string
	didFetchScansCh         chan *Result
	didFetchScanResultsCh   chan *ScanResults
	hasFetchedScansCh       chan chan bool
	getClientStateMetricsCh chan chan *clientStateMetrics
	unknownScansCh          chan chan []string
}

// NewHub returns a new Hub.  It will not be logged in.
func NewHub(username string, password string, host string, client RawClientInterface, timings *Timings) *Hub {
	hub := &Hub{
		client:         client,
		circuitBreaker: NewCircuitBreaker(host, maxHubExponentialBackoffDuration),
		username:       username,
		password:       password,
		host:           host,
		status:         ClientStatusDown,
		//
		hasFetchedScans: false,
		scans:           map[string]*Scan{},
		errors:          []error{},
		//
		publishUpdatesCh: make(chan Update),
		//
		stop:                    make(chan struct{}),
		resetCircuitBreakerCh:   make(chan struct{}),
		getModel:                make(chan chan *api.ModelHub),
		didLoginCh:              make(chan error),
		startScanClientCh:       make(chan string),
		finishScanClientCh:      make(chan *finishScanClient),
		getScanResultsCh:        make(chan chan map[string]*Scan),
		scanDidFinishCh:         make(chan *ScanResults),
		getScansCountCh:         make(chan chan int),
		getInProgressScansCh:    make(chan chan []string),
		didFetchScansCh:         make(chan *Result),
		didFetchScanResultsCh:   make(chan *ScanResults),
		hasFetchedScansCh:       make(chan chan bool),
		getClientStateMetricsCh: make(chan chan *clientStateMetrics),
		unknownScansCh:          make(chan chan []string)}
	// timers
	hub.getMetricsTimer = hub.startGetMetricsTimer(timings.GetMetricsPause)
	hub.checkScansForCompletionTimer = hub.startCheckScansForCompletionTimer(timings.ScanCompletionPause)
	hub.fetchScansTimer = hub.startFetchUnknownScansTimer(timings.FetchUnknownScansPause)
	hub.fetchAllScansTimer = hub.startFetchAllScansTimer(timings.FetchAllScansPause)
	hub.loginTimer = hub.startLoginTimer(timings.LoginPause)
	hub.refreshScansTimer = hub.startRefreshScansTimer(timings.RefreshScanThreshold)
	// action processing
	go func() {
		for {
			select {
			case <-hub.stop:
				return
			case <-hub.resetCircuitBreakerCh:
				recordEvent(hub.host, "resetCircuitBreaker")
				hub.circuitBreaker.Reset()
			case ch := <-hub.getModel:
				recordEvent(hub.host, "getModel")
				ch <- hub.apiModel()
			case ch := <-hub.unknownScansCh:
				recordEvent(hub.host, "getunknownScans")
				unknownScans := []string{}
				for name, scan := range hub.scans {
					if scan.Stage == ScanStageUnknown {
						unknownScans = append(unknownScans, name)
					}
				}
				ch <- unknownScans
			case ch := <-hub.getScanResultsCh:
				recordEvent(hub.host, "getScanResults")
				allScanResults := map[string]*Scan{}
				for name, scan := range hub.scans {
					allScanResults[name] = &Scan{Stage: scan.Stage, ScanResults: scan.ScanResults}
				}
				ch <- allScanResults
			case scanResults := <-hub.didFetchScanResultsCh:
				recordEvent(hub.host, "didFetchScanResults")
				scan, ok := hub.scans[scanResults.CodeLocationName]
				if !ok {
					scan = &Scan{
						ScanResults: scanResults,
						Stage:       ScanStageUnknown,
					}
					hub.scans[scanResults.CodeLocationName] = scan
				}
				switch scanResults.ScanSummaryStatus() {
				case ScanSummaryStatusSuccess:
					scan.Stage = ScanStageComplete
				case ScanSummaryStatusInProgress:
					// TODO any way to distinguish between scanclient and hubscan?
					scan.Stage = ScanStageHubScan
				case ScanSummaryStatusFailure:
					scan.Stage = ScanStageFailure
				}
				hub.scans[scanResults.CodeLocationName].ScanResults = scanResults
				update := &DidFindScan{Name: scanResults.CodeLocationName, Results: scanResults}
				hub.publish(update)
			case result := <-hub.didFetchScansCh:
				recordEvent(hub.host, "didFetchScans")
				hub.recordError(result.Err)
				if result.Err == nil {
					hub.hasFetchedScans = true
					cls := result.Value.([]hubapi.CodeLocation)
					for _, cl := range cls {
						if _, ok := hub.scans[cl.Name]; !ok {
							hub.scans[cl.Name] = &Scan{Stage: ScanStageUnknown, ScanResults: nil}
						}
					}
				}
			case scanName := <-hub.startScanClientCh:
				recordEvent(hub.host, "startScanClient")
				hub.scans[scanName] = &Scan{Stage: ScanStageScanClient}
			case obj := <-hub.finishScanClientCh:
				recordEvent(hub.host, "finishScanClient")
				scanName := obj.scanName
				scanErr := obj.err
				scan, ok := hub.scans[scanName]
				if !ok {
					log.Errorf("unable to handle finishScanClient for %s: not found", scanName)
					break
				}
				if scan.Stage != ScanStageScanClient {
					log.Warnf("unable to handle finishScanClient for %s: expected stage ScanClient, found %s", scanName, scan.Stage.String())
					break
				}
				if scanErr == nil {
					scan.Stage = ScanStageHubScan
				} else {
					scan.Stage = ScanStageFailure
				}
			case sr := <-hub.scanDidFinishCh:
				recordEvent(hub.host, "scanDidFinish")
				scanName := sr.CodeLocationName
				scan, ok := hub.scans[scanName]
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
			case get := <-hub.getScansCountCh:
				recordEvent(hub.host, "getScansCount")
				count := 0
				for _, cl := range hub.scans {
					if cl.Stage != ScanStageFailure {
						count++
					}
				}
				get <- count
			case get := <-hub.getInProgressScansCh:
				recordEvent(hub.host, "getInProgressScans")
				scans := []string{}
				for scanName, scan := range hub.scans {
					if scan.Stage == ScanStageHubScan || scan.Stage == ScanStageScanClient {
						scans = append(scans, scanName)
					}
				}
				get <- scans
			case ch := <-hub.hasFetchedScansCh:
				recordEvent(hub.host, "hasFetchedScans")
				ch <- hub.hasFetchedScans
			case ch := <-hub.getClientStateMetricsCh:
				recordEvent(hub.host, "getClientStateMetrics")
				scanStageCounts := map[ScanStage]int{}
				for _, scan := range hub.scans {
					scanStageCounts[scan.Stage]++
				}
				ch <- &clientStateMetrics{
					errorsCount:     len(hub.errors),
					scanStageCounts: scanStageCounts,
				}
			case err := <-hub.didLoginCh:
				recordEvent(hub.host, "didLogin")
				hub.recordError(err)
				if err != nil && hub.status == ClientStatusUp {
					hub.status = ClientStatusDown
					hub.recordError(hub.checkScansForCompletionTimer.Pause())
					hub.recordError(hub.fetchScansTimer.Pause())
					hub.recordError(hub.fetchAllScansTimer.Pause())
					hub.recordError(hub.refreshScansTimer.Pause())
				} else if err == nil && hub.status == ClientStatusDown {
					hub.status = ClientStatusUp
					hub.recordError(hub.checkScansForCompletionTimer.Resume(true))
					hub.recordError(hub.fetchScansTimer.Resume(true))
					hub.recordError(hub.fetchAllScansTimer.Resume(true))
					hub.recordError(hub.refreshScansTimer.Resume(true))
				}
			}
		}
	}()
	return hub
}

func (hub *Hub) publish(update Update) {
	go func() {
		select {
		case <-hub.stop:
			return
		case hub.publishUpdatesCh <- update:
		}
	}()
}

// Stop ...
func (hub *Hub) Stop() {
	close(hub.stop)
}

// StopCh returns a reference to the stop channel
func (hub *Hub) StopCh() <-chan struct{} {
	return hub.stop
}

// Host ...
func (hub *Hub) Host() string {
	return hub.host
}

// ResetCircuitBreaker ...
func (hub *Hub) ResetCircuitBreaker() {
	hub.resetCircuitBreakerCh <- struct{}{}
}

// Model ...
func (hub *Hub) Model() <-chan *api.ModelHub {
	ch := make(chan *api.ModelHub)
	hub.getModel <- ch
	return ch
}

// HasFetchedScans ...
func (hub *Hub) HasFetchedScans() <-chan bool {
	ch := make(chan bool)
	hub.hasFetchedScansCh <- ch
	return ch
}

func (hub *Hub) getStateMetrics() <-chan *clientStateMetrics {
	ch := make(chan *clientStateMetrics)
	hub.getClientStateMetricsCh <- ch
	return ch
}

// Private methods

func (hub *Hub) recordError(err error) {
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
func (hub *Hub) login() error {
	start := time.Now()
	err := hub.client.Login(hub.username, hub.password)
	recordHubResponse(hub.host, "login", err == nil)
	recordHubResponseTime(hub.host, "login", time.Now().Sub(start))
	return err
}

func (hub *Hub) apiModel() *api.ModelHub {
	errors := make([]string, len(hub.errors))
	for ix, err := range hub.errors {
		errors[ix] = err.Error()
	}
	codeLocations := map[string]*api.ModelCodeLocation{}
	for name, scan := range hub.scans {
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
		HasLoadedAllCodeLocations: hub.scans != nil,
		CodeLocations:             codeLocations,
		CircuitBreaker:            hub.circuitBreaker.Model(),
		Host:                      hub.host,
	}
}

// Regular jobs

func (hub *Hub) startRefreshScansTimer(pause time.Duration) *util.Timer {
	name := fmt.Sprintf("refresh-scans-%s", hub.host)
	return util.NewTimer(name, pause, hub.stop, func() {
		// TODO implement
	})
}

func (hub *Hub) startLoginTimer(pause time.Duration) *util.Timer {
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

func (hub *Hub) startFetchAllScansTimer(pause time.Duration) *util.Timer {
	name := fmt.Sprintf("fetchScans-%s", hub.host)
	return util.NewTimer(name, pause, hub.stop, func() {
		log.Debugf("starting to fetch all scans")
		result := hub.fetchAllCodeLocations()
		select {
		case hub.didFetchScansCh <- result:
		case <-hub.stop:
		}
	})
}

func (hub *Hub) getUnknownScans() []string {
	ch := make(chan []string)
	hub.unknownScansCh <- ch
	return <-ch
}

func (hub *Hub) startFetchUnknownScansTimer(pause time.Duration) *util.Timer {
	name := fmt.Sprintf("fetchUnknownScans-%s", hub.host)
	return util.NewTimer(name, pause, hub.stop, func() {
		log.Debugf("starting to fetch unknown scans")
		unknownScans := hub.getUnknownScans()
		log.Debugf("found %d unknown code locations", len(unknownScans))
		for _, codeLocationName := range unknownScans {
			scanResults, err := hub.fetchScan(codeLocationName)
			if err != nil {
				log.Errorf("unable to fetch scan %s: %s", codeLocationName, err.Error())
				continue
			}
			if scanResults == nil {
				log.Debugf("found nil scan for unknown code location %s", codeLocationName)
				continue
			}
			log.Debugf("fetched scan %s", codeLocationName)
			select {
			case hub.didFetchScanResultsCh <- scanResults:
			case <-hub.stop:
				return
			}
		}
		log.Debugf("finished fetching unknown scans")
	})
}

func (hub *Hub) startGetMetricsTimer(pause time.Duration) *util.Timer {
	name := fmt.Sprintf("getMetrics-%s", hub.host)
	return util.NewRunningTimer(name, pause, hub.stop, true, func() {
		var metrics *clientStateMetrics
		select {
		case <-hub.stop:
			return
		case m := <-hub.getStateMetrics():
			metrics = m
		}
		recordClientState(hub.host, metrics)
	})
}

func (hub *Hub) startCheckScansForCompletionTimer(pause time.Duration) *util.Timer {
	name := fmt.Sprintf("checkScansForCompletion-%s", hub.host)
	return util.NewTimer(name, pause, hub.stop, func() {
		var scanNames []string
		select {
		case scanNames = <-hub.InProgressScans():
		case <-hub.stop:
			return
		}
		log.Debugf("starting to check scans for completion: %+v", scanNames)
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

func (hub *Hub) fetchAllCodeLocations() *Result {
	codeLocationList, err := hub.listAllCodeLocations()
	if err != nil {
		return &Result{Value: nil, Err: err}
	}
	log.Debugf("fetched all code locations: found %d, expected %d", len(codeLocationList.Items), codeLocationList.TotalCount)
	return &Result{Value: codeLocationList.Items, Err: nil}
}

// Version fetches the hub version
func (hub *Hub) Version() (string, error) {
	start := time.Now()
	currentVersion, err := hub.client.CurrentVersion()
	recordHubResponse(hub.host, "version", err == nil)
	recordHubResponseTime(hub.host, "version", time.Now().Sub(start))
	if err != nil {
		log.Errorf("unable to get hub version: %s", err.Error())
		return "", err
	}

	log.Infof("successfully got hub version %s", currentVersion.Version)
	return currentVersion.Version, nil
}

// SetTimeout is currently not concurrent-safe, and should be made so TODO
func (hub *Hub) SetTimeout(timeout time.Duration) {
	hub.client.SetTimeout(timeout)
}

// StartScanClient ...
func (hub *Hub) StartScanClient(scanName string) {
	hub.startScanClientCh <- scanName
}

// FinishScanClient ...
func (hub *Hub) FinishScanClient(scanName string, scanErr error) {
	hub.finishScanClientCh <- &finishScanClient{scanName, scanErr}
}

// ScansCount ...
func (hub *Hub) ScansCount() <-chan int {
	ch := make(chan int)
	hub.getScansCountCh <- ch
	return ch
}

// InProgressScans ...
func (hub *Hub) InProgressScans() <-chan []string {
	ch := make(chan []string)
	hub.getInProgressScansCh <- ch
	return ch
}

// ScanResults ...
func (hub *Hub) ScanResults() <-chan map[string]*Scan {
	ch := make(chan map[string]*Scan)
	hub.getScanResultsCh <- ch
	return ch
}

// Updates produces events for:
// - finding a scan for the first time
// - when a hub scan finishes
// - when a finished scan is repulled (to get any changes to its vulnerabilities, policies, etc.)
func (hub *Hub) Updates() <-chan Update {
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
func (hub *Hub) fetchScan(scanNameSearchString string) (*ScanResults, error) {
	codeLocationList, err := hub.listCodeLocations(scanNameSearchString)

	if err != nil {
		recordError(hub.host, "fetch code location list")
		log.Errorf("error fetching code location list: %v", err)
		return nil, err
	}
	codeLocations := codeLocationList.Items
	switch len(codeLocations) {
	case 0:
		recordHubData(hub.host, "codeLocations", true)
		return nil, nil
	case 1:
		recordHubData(hub.host, "codeLocations", true) // good to go
	default:
		recordHubData(hub.host, "codeLocations", false)
		log.Warnf("expected 1 code location matching name search string %s, found %d", scanNameSearchString, len(codeLocations))
	}

	codeLocation := codeLocations[0]
	return hub.fetchScanResultsUsingCodeLocation(codeLocation, scanNameSearchString)
}

func (hub *Hub) fetchScanResultsUsingCodeLocation(codeLocation hubapi.CodeLocation, scanNameSearchString string) (*ScanResults, error) {
	versionLink, err := codeLocation.GetProjectVersionLink()
	if err != nil {
		recordError(hub.host, "get project version link")
		log.Errorf("unable to get project version link: %s", err.Error())
		return nil, err
	}

	version, err := hub.getProjectVersion(*versionLink)
	if err != nil {
		recordError(hub.host, "fetch project version")
		log.Errorf("unable to fetch project version: %s", err.Error())
		return nil, err
	}

	riskProfileLink, err := version.GetProjectVersionRiskProfileLink()
	if err != nil {
		recordError(hub.host, "get risk profile link")
		log.Errorf("error getting risk profile link: %v", err)
		return nil, err
	}

	riskProfile, err := hub.getProjectVersionRiskProfile(*riskProfileLink)
	if err != nil {
		recordError(hub.host, "fetch project version risk profile")
		log.Errorf("error fetching project version risk profile: %v", err)
		return nil, err
	}

	policyStatusLink, err := version.GetProjectVersionPolicyStatusLink()
	if err != nil {
		recordError(hub.host, "get policy status link")
		log.Errorf("error getting policy status link: %v", err)
		return nil, err
	}
	policyStatus, err := hub.getProjectVersionPolicyStatus(*policyStatusLink)
	if err != nil {
		recordError(hub.host, "fetch policy status")
		log.Errorf("error fetching project version policy status: %v", err)
		return nil, err
	}

	componentsLink, err := version.GetComponentsLink()
	if err != nil {
		recordError(hub.host, "get components link")
		log.Errorf("error getting components link: %v", err)
		return nil, err
	}

	scanSummariesLink, err := codeLocation.GetScanSummariesLink()
	if err != nil {
		recordError(hub.host, "get scan summaries link")
		log.Errorf("error getting scan summaries link: %v", err)
		return nil, err
	}
	scanSummariesList, err := hub.listScanSummaries(*scanSummariesLink)
	if err != nil {
		recordError(hub.host, "fetch scan summaries")
		log.Errorf("error fetching scan summaries: %v", err)
		return nil, err
	}

	switch len(scanSummariesList.Items) {
	case 0:
		recordHubData(hub.host, "scan summaries", true)
		return nil, nil
	case 1:
		recordHubData(hub.host, "scan summaries", true) // good to go, continue
	default:
		recordHubData(hub.host, "scan summaries", false)
		log.Warnf("expected to find one scan summary for code location %s, found %d", scanNameSearchString, len(scanSummariesList.Items))
	}

	mappedRiskProfile, err := newRiskProfile(riskProfile.BomLastUpdatedAt, riskProfile.Categories)
	if err != nil {
		recordError(hub.host, "map risk profile")
		return nil, err
	}

	mappedPolicyStatus, err := newPolicyStatus(policyStatus.OverallStatus, policyStatus.UpdatedAt, policyStatus.ComponentVersionStatusCounts)
	if err != nil {
		recordError(hub.host, "map policy status")
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
func (hub *Hub) listAllProjects() (*hubapi.ProjectList, error) {
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
func (hub *Hub) listAllCodeLocations() (*hubapi.CodeLocationList, error) {
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
func (hub *Hub) listCodeLocations(codeLocationName string) (*hubapi.CodeLocationList, error) {
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
func (hub *Hub) getProjectVersion(link hubapi.ResourceLink) (*hubapi.ProjectVersion, error) {
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
func (hub *Hub) getProject(link hubapi.ResourceLink) (*hubapi.Project, error) {
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
func (hub *Hub) getProjectVersionRiskProfile(link hubapi.ResourceLink) (*hubapi.ProjectVersionRiskProfile, error) {
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
func (hub *Hub) getProjectVersionPolicyStatus(link hubapi.ResourceLink) (*hubapi.ProjectVersionPolicyStatus, error) {
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
func (hub *Hub) listScanSummaries(link hubapi.ResourceLink) (*hubapi.ScanSummaryList, error) {
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
func (hub *Hub) deleteProjectVersion(projectVersionHRef string) error {
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
func (hub *Hub) deleteCodeLocation(codeLocationHRef string) error {
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

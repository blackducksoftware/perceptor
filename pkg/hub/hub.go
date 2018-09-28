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
	client *Client
	// basic hub info
	host   string
	status ClientStatus
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
func NewHub(username string, password string, host string, rawClient RawClientInterface, timings *Timings) *Hub {
	hub := &Hub{
		client: NewClient(username, password, host, rawClient),
		host:   host,
		status: ClientStatusDown,
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
				hub.client.resetCircuitBreaker()
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
		CircuitBreaker:            hub.client.circuitBreaker.Model(),
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
		err := hub.client.login()
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
			scanResults, err := hub.client.fetchScan(codeLocationName)
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
			scanResults, err := hub.client.fetchScan(scanName)
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
	codeLocationList, err := hub.client.listAllCodeLocations()
	if err != nil {
		return &Result{Value: nil, Err: err}
	}
	log.Debugf("fetched all code locations: found %d, expected %d", len(codeLocationList.Items), codeLocationList.TotalCount)
	return &Result{Value: codeLocationList.Items, Err: nil}
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

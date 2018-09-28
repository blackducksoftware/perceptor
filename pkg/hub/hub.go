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
	stop    chan struct{}
	actions chan *clientAction
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
		stop:    make(chan struct{}),
		actions: make(chan *clientAction)}
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
			case action := <-hub.actions:
				// TODO what other logging, metrics, etc. would help here?
				recordEvent(hub.host, action.name)
				err := action.apply()
				if err != nil {
					log.Error(err.Error())
					recordError(hub.host, action.name)
				}
			}
		}
	}()
	return hub
}

// Private methods

func (hub *Hub) publish(update Update) {
	go func() {
		select {
		case <-hub.stop:
			return
		case hub.publishUpdatesCh <- update:
		}
	}()
}

func (hub *Hub) getStateMetrics() <-chan *clientStateMetrics {
	ch := make(chan *clientStateMetrics)
	hub.actions <- &clientAction{"getClientStateMetrics", func() error {
		scanStageCounts := map[ScanStage]int{}
		for _, scan := range hub.scans {
			scanStageCounts[scan.Stage]++
		}
		ch <- &clientStateMetrics{
			errorsCount:     len(hub.errors),
			scanStageCounts: scanStageCounts,
		}
		return nil
	}}
	return ch
}

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

func (hub *Hub) didLogin(err error) {
	hub.actions <- &clientAction{"didLogin", func() error {
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
		return nil
	}}
}

func (hub *Hub) startLoginTimer(pause time.Duration) *util.Timer {
	name := fmt.Sprintf("login-%s", hub.host)
	return util.NewRunningTimer(name, pause, hub.stop, true, func() {
		log.Debugf("starting to login to hub")
		err := hub.client.login()
		hub.didLogin(err)
	})
}

func (hub *Hub) didFetchScans(cls *hubapi.CodeLocationList, err error) {
	hub.actions <- &clientAction{"didFetchScans", func() error {
		hub.recordError(err)
		if err == nil {
			hub.hasFetchedScans = true
			for _, cl := range cls.Items {
				if _, ok := hub.scans[cl.Name]; !ok {
					hub.scans[cl.Name] = &Scan{Stage: ScanStageUnknown, ScanResults: nil}
				}
			}
		}
		return nil
	}}
}

func (hub *Hub) startFetchAllScansTimer(pause time.Duration) *util.Timer {
	name := fmt.Sprintf("fetchScans-%s", hub.host)
	return util.NewTimer(name, pause, hub.stop, func() {
		log.Debugf("starting to fetch all scans")
		cls, err := hub.client.listAllCodeLocations()
		hub.didFetchScans(cls, err)
	})
}

func (hub *Hub) getUnknownScans() []string {
	ch := make(chan []string)
	hub.actions <- &clientAction{"getUnknownScans", func() error {
		unknownScans := []string{}
		for name, scan := range hub.scans {
			if scan.Stage == ScanStageUnknown {
				unknownScans = append(unknownScans, name)
			}
		}
		ch <- unknownScans
		return nil
	}}
	return <-ch
}

func (hub *Hub) didFetchScanResults(scanResults *ScanResults) {
	hub.actions <- &clientAction{"didFetchScanResults", func() error {
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
		return nil
	}}
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
			hub.didFetchScanResults(scanResults)
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

func (hub *Hub) scanDidFinish(scanResults *ScanResults) {
	hub.actions <- &clientAction{"scanDidFinish", func() error {
		scanName := scanResults.CodeLocationName
		scan, ok := hub.scans[scanName]
		if !ok {
			return fmt.Errorf("unable to handle scanDidFinish for %s: not found", scanName)
		}
		if scan.Stage != ScanStageHubScan {
			return fmt.Errorf("unable to handle scanDidFinish for %s: expected stage HubScan, found %s", scanName, scan.Stage.String())
		}
		scan.Stage = ScanStageComplete
		update := &DidFinishScan{Name: scanResults.CodeLocationName, Results: scanResults}
		hub.publish(update)
		return nil
	}}
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
				hub.scanDidFinish(scanResults)
			}
		}
	})
}

// Some public API methods ...

// StartScanClient ...
func (hub *Hub) StartScanClient(scanName string) {
	hub.actions <- &clientAction{"startScanClient", func() error {
		hub.scans[scanName] = &Scan{Stage: ScanStageScanClient}
		return nil
	}}
}

// FinishScanClient ...
func (hub *Hub) FinishScanClient(scanName string, scanErr error) {
	hub.actions <- &clientAction{"finishScanClient", func() error {
		scan, ok := hub.scans[scanName]
		if !ok {
			return fmt.Errorf("unable to handle finishScanClient for %s: not found", scanName)
		}
		if scan.Stage != ScanStageScanClient {
			return fmt.Errorf("unable to handle finishScanClient for %s: expected stage ScanClient, found %s", scanName, scan.Stage.String())
		}
		if scanErr == nil {
			scan.Stage = ScanStageHubScan
		} else {
			scan.Stage = ScanStageFailure
		}
		return nil
	}}
}

// ScansCount ...
func (hub *Hub) ScansCount() <-chan int {
	ch := make(chan int)
	hub.actions <- &clientAction{"getScansCount", func() error {
		count := 0
		for _, cl := range hub.scans {
			if cl.Stage != ScanStageFailure {
				count++
			}
		}
		ch <- count
		return nil
	}}
	return ch
}

// InProgressScans ...
func (hub *Hub) InProgressScans() <-chan []string {
	ch := make(chan []string)
	hub.actions <- &clientAction{"getInProgressScans", func() error {
		scans := []string{}
		for scanName, scan := range hub.scans {
			if scan.Stage == ScanStageHubScan || scan.Stage == ScanStageScanClient {
				scans = append(scans, scanName)
			}
		}
		ch <- scans
		return nil
	}}
	return ch
}

// ScanResults ...
func (hub *Hub) ScanResults() <-chan map[string]*Scan {
	ch := make(chan map[string]*Scan)
	hub.actions <- &clientAction{"getScanResults", func() error {
		allScanResults := map[string]*Scan{}
		for name, scan := range hub.scans {
			allScanResults[name] = &Scan{Stage: scan.Stage, ScanResults: scan.ScanResults}
		}
		ch <- allScanResults
		return nil
	}}
	return ch
}

// Updates produces events for:
// - finding a scan for the first time
// - when a hub scan finishes
// - when a finished scan is repulled (to get any changes to its vulnerabilities, policies, etc.)
func (hub *Hub) Updates() <-chan Update {
	return hub.publishUpdatesCh
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
	recordEvent(hub.host, "resetCircuitBreaker")
	hub.client.resetCircuitBreaker()
}

// Model ...
func (hub *Hub) Model() <-chan *api.ModelHub {
	ch := make(chan *api.ModelHub)
	hub.actions <- &clientAction{"getModel", func() error {
		ch <- hub.apiModel()
		return nil
	}}
	return ch
}

// HasFetchedScans ...
func (hub *Hub) HasFetchedScans() <-chan bool {
	ch := make(chan bool)
	hub.actions <- &clientAction{"hasFetchedScans", func() error {
		ch <- hub.hasFetchedScans
		return nil
	}}
	return ch
}

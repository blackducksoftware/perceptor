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

	"github.com/blackducksoftware/hub-client-go/hubapi"

	"github.com/blackducksoftware/perceptor/pkg/api"
	log "github.com/sirupsen/logrus"
)

type modelAction struct {
	name  string
	apply func() error
}

// Model .....
type Model struct {
	// basic hub info
	host string
	// data
	hasFetchedScans bool
	scans           map[string]*Scan
	// public channels
	stop    chan struct{}
	actions chan *modelAction
}

// NewModel ...
func NewModel(host string) *Model {
	model := &Model{
		host:            host,
		hasFetchedScans: false,
		scans:           map[string]*Scan{},
		stop:            make(chan struct{}),
		actions:         make(chan *modelAction)}
	// action processing
	go func() {
		for {
			select {
			case <-model.stop:
				return
			case action := <-model.actions:
				// TODO what other logging, metrics, etc. would help here?
				recordEvent(model.host, action.name)
				err := action.apply()
				if err != nil {
					log.Errorf("while processing action %s: %s", action.name, err.Error())
					recordError(model.host, action.name)
				}
			}
		}
	}()
	return model
}

// Private methods

func (model *Model) getStateMetrics() {
	ch := make(chan *clientStateMetrics)
	model.actions <- &modelAction{"getClientStateMetrics", func() error {
		scanStageCounts := map[ScanStage]int{}
		for _, scan := range model.scans {
			scanStageCounts[scan.Stage]++
		}
		ch <- &clientStateMetrics{
			//			errorsCount:     len(model.errors), TODO
			scanStageCounts: scanStageCounts,
		}
		return nil
	}}
	recordClientState(model.host, <-ch)
}

func (model *Model) apiModel() *api.ModelHub {
	// TODO
	// errors := make([]string, len(model.errors))
	// for ix, err := range model.errors {
	// 	errors[ix] = err.Error()
	// }
	codeLocations := map[string]*api.ModelCodeLocation{}
	for name, scan := range model.scans {
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
		// TODO
		// Errors:                    errors,
		// Status:                    model.status.String(),
		HasLoadedAllCodeLocations: model.scans != nil,
		CodeLocations:             codeLocations,
		// CircuitBreaker:            model.client.circuitBreaker.Model(),
		Host: model.host,
	}
}

func (model *Model) didFetchScans(cls *hubapi.CodeLocationList, err error) {
	model.actions <- &modelAction{"didFetchScans", func() error {
		// TODO
		//		model.recordError(err)
		if err == nil {
			model.hasFetchedScans = true
			for _, cl := range cls.Items {
				if _, ok := model.scans[cl.Name]; !ok {
					model.scans[cl.Name] = &Scan{Stage: ScanStageUnknown, ScanResults: nil}
				}
			}
		}
		return nil
	}}
}

func (model *Model) getUnknownScans() []string {
	ch := make(chan []string)
	model.actions <- &modelAction{"getUnknownScans", func() error {
		unknownScans := []string{}
		for name, scan := range model.scans {
			if scan.Stage == ScanStageUnknown {
				unknownScans = append(unknownScans, name)
			}
		}
		ch <- unknownScans
		return nil
	}}
	return <-ch
}

func (model *Model) didFetchScanResults(scanResults *ScanResults) {
	model.actions <- &modelAction{"didFetchScanResults", func() error {
		scan, ok := model.scans[scanResults.CodeLocationName]
		if !ok {
			scan = &Scan{
				ScanResults: scanResults,
				Stage:       ScanStageUnknown,
			}
			model.scans[scanResults.CodeLocationName] = scan
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
		model.scans[scanResults.CodeLocationName].ScanResults = scanResults
		update := &DidFindScan{Name: scanResults.CodeLocationName, Results: scanResults}
		model.publish(update)
		return nil
	}}
}

func (model *Model) fetchUnknownScans() {
	log.Debugf("starting to fetch unknown scans")
	unknownScans := model.getUnknownScans()
	log.Debugf("found %d unknown code locations", len(unknownScans))
	for _, codeLocationName := range unknownScans {
		scanResults, err := model.client.fetchScan(codeLocationName)
		if err != nil {
			log.Errorf("unable to fetch scan %s: %s", codeLocationName, err.Error())
			continue
		}
		if scanResults == nil {
			log.Debugf("found nil scan for unknown code location %s", codeLocationName)
			continue
		}
		log.Debugf("fetched scan %s", codeLocationName)
		model.didFetchScanResults(scanResults)
	}
	log.Debugf("finished fetching unknown scans")
}

func (model *Model) scanDidFinish(scanResults *ScanResults) {
	model.actions <- &modelAction{"scanDidFinish", func() error {
		scanName := scanResults.CodeLocationName
		scan, ok := model.scans[scanName]
		if !ok {
			return fmt.Errorf("unable to handle scanDidFinish for %s: not found", scanName)
		}
		if scan.Stage != ScanStageHubScan {
			return fmt.Errorf("unable to handle scanDidFinish for %s: expected stage HubScan, found %s", scanName, scan.Stage.String())
		}
		scan.Stage = ScanStageComplete
		if scanResults != nil {
			scan.ScanResults = scanResults
		}
		update := &DidFinishScan{Name: scanResults.CodeLocationName, Results: scanResults}
		model.publish(update)
		return nil
	}}
}

func (model *Model) checkScansForCompletion() {
	var scanNames []string
	select {
	case scanNames = <-model.InProgressScans():
	case <-model.stop:
		return
	}
	log.Debugf("starting to check scans for completion: %+v", scanNames)
	for _, scanName := range scanNames {
		scanResults, err := model.client.fetchScan(scanName)
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
			model.scanDidFinish(scanResults)
		}
	}
}

// Some public API methods ...

// StartScanClient ...
func (model *Model) StartScanClient(scanName string) {
	model.actions <- &modelAction{"startScanClient", func() error {
		model.scans[scanName] = &Scan{Stage: ScanStageScanClient}
		return nil
	}}
}

// FinishScanClient ...
func (model *Model) FinishScanClient(scanName string, scanErr error) {
	model.actions <- &modelAction{"finishScanClient", func() error {
		scan, ok := model.scans[scanName]
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
func (model *Model) ScansCount() <-chan int {
	ch := make(chan int)
	model.actions <- &modelAction{"getScansCount", func() error {
		count := 0
		for _, cl := range model.scans {
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
func (model *Model) InProgressScans() <-chan []string {
	ch := make(chan []string)
	model.actions <- &modelAction{"getInProgressScans", func() error {
		scans := []string{}
		for scanName, scan := range model.scans {
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
func (model *Model) ScanResults() <-chan map[string]*Scan {
	ch := make(chan map[string]*Scan)
	model.actions <- &modelAction{"getScanResults", func() error {
		allScanResults := map[string]*Scan{}
		for name, scan := range model.scans {
			allScanResults[name] = &Scan{Stage: scan.Stage, ScanResults: scan.ScanResults}
		}
		ch <- allScanResults
		return nil
	}}
	return ch
}

// Model ...
func (model *Model) Model() <-chan *api.ModelHub {
	ch := make(chan *api.ModelHub)
	model.actions <- &modelAction{"getModel", func() error {
		ch <- model.apiModel()
		return nil
	}}
	return ch
}

// HasFetchedScans ...
func (model *Model) HasFetchedScans() <-chan bool {
	ch := make(chan bool)
	model.actions <- &modelAction{"hasFetchedScans", func() error {
		ch <- model.hasFetchedScans
		return nil
	}}
	return ch
}

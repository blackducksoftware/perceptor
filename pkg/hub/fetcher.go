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
	"github.com/blackducksoftware/perceptor/pkg/util"
	log "github.com/sirupsen/logrus"
)

const (
	maxHubExponentialBackoffDuration = 1 * time.Hour
)

// Fetcher .....
// TODO treat this like an actor; give it thread safety
type Fetcher struct {
	client          *hubclient.Client
	circuitBreaker  *CircuitBreaker
	hubVersion      string
	username        string
	password        string
	baseURL         string
	inProgressScans map[string]bool
	finishScan      chan *HubImageScan
	hubScanPoller   *util.Scheduler
	reloginPause    time.Duration
	stop            <-chan struct{}
}

func (hf *Fetcher) fetchHubVersion() error {
	start := time.Now()
	currentVersion, err := hf.client.CurrentVersion()
	recordHubResponse("version", err == nil)
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
func NewFetcher(username string, password string, host string, port int, timeout time.Duration, checkHubPause time.Duration, stop <-chan struct{}, reloginPause time.Duration) (*Fetcher, error) {
	baseURL := fmt.Sprintf("https://%s:%d", host, port)
	client, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings, timeout)
	if err != nil {
		return nil, err
	}
	hf := Fetcher{
		client:          client,
		circuitBreaker:  NewCircuitBreaker(maxHubExponentialBackoffDuration, client),
		username:        username,
		password:        password,
		baseURL:         baseURL,
		inProgressScans: map[string]bool{},
		finishScan:      make(chan *HubImageScan),
		reloginPause:    reloginPause,
		stop:            stop}
	hf.hubScanPoller = util.NewScheduler(checkHubPause, stop, func() {
		hf.fetchScans()
	})
	err = hf.login()
	if err != nil {
		return nil, err
	}
	err = hf.fetchHubVersion()
	if err != nil {
		return nil, err
	}
	return &hf, nil
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

func (hf *Fetcher) startReloggingInToHub() *util.Scheduler {
	return util.NewScheduler(hf.reloginPause, hf.stop, func() {
		_ = hf.login()
	})
}

func (hf *Fetcher) login() error {
	start := time.Now()
	err := hf.client.Login(hf.username, hf.password)
	recordHubResponse("login", err == nil)
	recordHubResponseTime("login", time.Now().Sub(start))
	if err != nil {
		log.Errorf("unable to re-login to hub: %s", err.Error())
	} else {
		log.Infof("successfully re-logged in to hub")
	}
	return err
}

// SetTimeout ...
func (hf *Fetcher) SetTimeout(timeout time.Duration) {
	hf.client.SetTimeout(timeout)
}

// HubVersion .....
func (hf *Fetcher) HubVersion() string {
	return hf.hubVersion
}

func (hf *Fetcher) fetchScans() {
	for scanName := range hf.inProgressScans {
		scanResults, err := hf.fetchScan(scanName)
		if (err == nil) && (scanResults.ScanSummaryStatus() != ScanSummaryStatusInProgress) {
			hf.finishScan <- &HubImageScan{ScanName: scanName, Scan: scanResults}
			delete(hf.inProgressScans, scanName)
		} else if err != nil {
			log.Errorf("unable to fetch scan for %s: %s", scanName, err.Error())
		}
	}
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
func (hf *Fetcher) fetchScan(scanName string) (*ScanResults, error) {
	codeLocationList, err := hf.circuitBreaker.ListCodeLocations(scanName)

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
		log.Warnf("expected 1 code location matching name search string %s, found %d", scanName, len(codeLocations))
	}

	codeLocation := codeLocations[0]
	return hf.fetchScanResultsUsingCodeLocation(codeLocation, scanName)
}

func (hf *Fetcher) fetchScanResultsUsingCodeLocation(codeLocation hubapi.CodeLocation, scanName string) (*ScanResults, error) {
	versionLink, err := codeLocation.GetProjectVersionLink()
	if err != nil {
		log.Errorf("unable to get project version link: %s", err.Error())
		return nil, err
	}

	version, err := hf.circuitBreaker.GetProjectVersion(*versionLink)
	if err != nil {
		log.Errorf("unable to fetch project version: %s", err.Error())
		return nil, err
	}

	riskProfileLink, err := version.GetProjectVersionRiskProfileLink()
	if err != nil {
		log.Errorf("error getting risk profile link: %v", err)
		return nil, err
	}

	riskProfile, err := hf.circuitBreaker.GetProjectVersionRiskProfile(*riskProfileLink)
	if err != nil {
		log.Errorf("error fetching project version risk profile: %v", err)
		return nil, err
	}

	policyStatusLink, err := version.GetProjectVersionPolicyStatusLink()
	if err != nil {
		log.Errorf("error getting policy status link: %v", err)
		return nil, err
	}
	policyStatus, err := hf.circuitBreaker.GetProjectVersionPolicyStatus(*policyStatusLink)
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
	scanSummariesList, err := hf.circuitBreaker.ListScanSummaries(*scanSummariesLink)
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
		log.Warnf("expected to find one scan summary for code location %s, found %d", scanName, len(scanSummariesList.Items))
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
		ScanName:              scanName,
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

// AddScan ...
func (hf *Fetcher) AddScan(scanName string) {
	hf.inProgressScans[scanName] = true
}

// ScansInProgress ...
func (hf *Fetcher) ScansInProgress() []string {
	panic("unimplemented!  maybe remove this")
}

// ScanDidFinish ...
func (hf *Fetcher) ScanDidFinish() <-chan *HubImageScan {
	return hf.finishScan
}

// GetAllCodeLocations ...
func (hf *Fetcher) GetAllCodeLocations() ([]string, error) {
	clList, err := hf.circuitBreaker.ListAllCodeLocations()
	if err != nil {
		return nil, err
	}
	codeLocationNames := make([]string, len(clList.Items))
	for i, cl := range clList.Items {
		codeLocationNames[i] = cl.Name
	}
	return codeLocationNames, nil
}

// HubURL ...
func (hf *Fetcher) HubURL() string {
	return hf.baseURL
}

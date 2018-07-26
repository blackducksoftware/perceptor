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
	hubDeleteTimeout                 = 1 * time.Hour
)

// Fetcher .....
type Fetcher struct {
	client         *hubclient.Client
	deleteClient   *hubclient.Client
	scansToDelete  map[string]bool
	circuitBreaker *CircuitBreaker
	hubVersion     string
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

// Login .....
func (hf *Fetcher) Login() error {
	start := time.Now()
	err := hf.client.Login(hf.username, hf.password)
	recordHubResponse("login", err == nil)
	recordHubResponseTime("login", time.Now().Sub(start))
	if err != nil {
		return err
	}
	startDelete := time.Now()
	err = hf.deleteClient.Login(hf.username, hf.password)
	recordHubResponse("login", err == nil)
	recordHubResponseTime("login", time.Now().Sub(startDelete))
	return err
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
func NewFetcher(username string, password string, hubHost string, hubPort int, hubClientTimeoutMilliseconds int) (*Fetcher, error) {
	baseURL := fmt.Sprintf("https://%s:%d", hubHost, hubPort)
	hubClientTimeout := time.Millisecond * time.Duration(hubClientTimeoutMilliseconds)
	client, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings, hubClientTimeout)
	if err != nil {
		return nil, err
	}
	deleteClient, err := hubclient.NewWithSession(baseURL, 0, hubDeleteTimeout)
	if err != nil {
		return nil, err
	}
	hf := Fetcher{
		client:         client,
		deleteClient:   deleteClient,
		scansToDelete:  map[string]bool{},
		circuitBreaker: NewCircuitBreaker(maxHubExponentialBackoffDuration, client),
		username:       username,
		password:       password,
		baseURL:        baseURL}
	err = hf.Login()
	if err != nil {
		return nil, err
	}
	err = hf.fetchHubVersion()
	if err != nil {
		return nil, err
	}
	// TODO replace with scheduler
	hf.startDeletingScans()
	return &hf, nil
}

// SetTimeout ...
func (hf *Fetcher) SetTimeout(timeout time.Duration) {
	hf.client.SetTimeout(timeout)
}

// HubVersion .....
func (hf *Fetcher) HubVersion() string {
	return hf.hubVersion
}

// DeleteScans ...
func (hf *Fetcher) DeleteScans(scanNames []string) {
	// TODO protect from concurrent read/write
	for _, scanName := range scanNames {
		hf.scansToDelete[scanName] = true
	}
}

func (hf *Fetcher) startDeletingScans() {
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			var scanName *string
			for key := range hf.scansToDelete {
				scanName = &key
				break
			}
			if scanName != nil {
				err := hf.DeleteScan(*scanName)
				if err != nil {
					log.Errorf("unable to delete scan: %s", err.Error())
				} else {
					delete(hf.scansToDelete, *scanName)
				}
			}
		}
	}()
}

// DeleteScan deletes the code location and project version (but NOT the project)
// associated with the given scan name.
func (hf *Fetcher) DeleteScan(scanName string) error {
	if !hf.circuitBreaker.IsEnabled() {
		return fmt.Errorf("unable to delete scan, circuit breaker is disabled")
	}
	queryString := fmt.Sprintf("name:%s", scanName)
	start := time.Now()
	clList, err := hf.deleteClient.ListAllCodeLocations(&hubapi.GetListOptions{Q: &queryString})
	recordHubResponseTime("allCodeLocations", time.Now().Sub(start))
	recordHubResponse("allCodeLocations", err == nil)
	switch len(clList.Items) {
	case 0:
		return nil
	case 1:
		// continue
	default:
		return fmt.Errorf("expected 0 or 1 scans of name %s, found %d", scanName, len(clList.Items))
	}
	codeLocation := clList.Items[0]
	deleteCodeLocationStart := time.Now()
	err = hf.deleteClient.DeleteCodeLocation(codeLocation.Meta.Href)
	recordHubResponseTime("deleteCodeLocation", time.Now().Sub(deleteCodeLocationStart))
	recordHubResponse("deleteCodeLocation", err == nil)
	if err != nil {
		return err
	}
	deleteVersionStart := time.Now()
	err = hf.deleteClient.DeleteProjectVersion(codeLocation.MappedProjectVersion)
	recordHubResponseTime("deleteVersion", time.Now().Sub(deleteVersionStart))
	recordHubResponse("deleteVersion", err == nil)
	return err
}

// func (hf *Fetcher) FetchAllScanNames() ([]string, error) {
// 	codeLocationList, err := hf.circuitBreaker.ListAllCodeLocations()
// 	if err != nil {
// 		return nil, err
// 	}
// 	scanNames := make([]string, len(codeLocationList.Items))
// 	for i, cl := range codeLocationList.Items {
// 		scanNames[i] = cl.Name
// 	}
// 	return scanNames, nil
// }

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
	codeLocationList, err := hf.circuitBreaker.ListCodeLocations(scanNameSearchString)

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

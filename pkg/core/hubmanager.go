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

package core

import (
	"fmt"
	"regexp"
	"time"

	"github.com/blackducksoftware/hub-client-go/hubclient"
	"github.com/blackducksoftware/perceptor/pkg/hub"
	log "github.com/sirupsen/logrus"
)

var commonMistakesRegex = regexp.MustCompile("(http|://|:\\d+)")

type hubClientCreator func(scheme string, host string, port int, username string, password string, concurrentScanLimit int) (*hub.Hub, error)

func createMockHubClient(scheme string, host string, port int, username string, password string, concurrentScanLimit int) (*hub.Hub, error) {
	mockRawClient := hub.NewMockRawClient(false, []string{})
	return hub.NewHub(username, password, host, concurrentScanLimit, mockRawClient, hub.DefaultTimings), nil
}

func createHubClient(httpTimeout time.Duration) hubClientCreator {
	return func(scheme string, host string, port int, username string, password string, concurrentScanLimit int) (*hub.Hub, error) {
		potentialProblems := commonMistakesRegex.FindAllString(host, -1)
		if len(potentialProblems) > 0 {
			log.Warnf("Hub host %s may be invalid, potential problems are: %s", host, potentialProblems)
		}
		baseURL := fmt.Sprintf("%s://%s:%d", scheme, host, port)
		rawClient, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings, httpTimeout)
		if err != nil {
			return nil, err
		}
		return hub.NewHub(username, password, host, concurrentScanLimit, rawClient, hub.DefaultTimings), nil
	}
}

// Update is a wrapper around hub.Update which also tracks which Hub was the source.
type Update struct {
	HubURL string
	Update hub.Update
}

// HubManagerInterface ...
type HubManagerInterface interface {
	SetHubs(hubs []*Host)
	HubClients() map[string]*hub.Hub
	StartScanClient(hubURL string, scanName string) error
	FinishScanClient(hubURL string, scanName string, err error) error
	ScanResults() map[string]map[string]*hub.Scan
	Updates() <-chan *Update
}

// HubManager ...
type HubManager struct {
	newHub hubClientCreator
	//
	stop    <-chan struct{}
	updates chan *Update
	//
	hubs                  map[string]*hub.Hub
	didFetchScanResults   chan *hub.ScanResults
	didFetchCodeLocations chan []string
}

// NewHubManager ...
func NewHubManager(newHub hubClientCreator, stop <-chan struct{}) *HubManager {
	// TODO needs to be made concurrent-safe
	return &HubManager{
		newHub:                newHub,
		stop:                  stop,
		updates:               make(chan *Update),
		hubs:                  map[string]*hub.Hub{},
		didFetchScanResults:   make(chan *hub.ScanResults),
		didFetchCodeLocations: make(chan []string)}
}

// SetHubs ...
func (hm *HubManager) SetHubs(hubs []*Host) {
	newHubURLs := map[string]bool{}
	for _, hub := range hubs {
		newHubURLs[hub.Domain] = true
	}
	hubsToCreate := map[string]bool{}
	for hubURL := range newHubURLs {
		if _, ok := hm.hubs[hubURL]; !ok {
			hubsToCreate[hubURL] = true
		}
	}
	// 1. create new hubs
	// TODO handle retries and failures intelligently
	go func() {
		for _, hub := range hubs {
			if _, ok := hm.hubs[hub.Domain]; !ok {
				err := hm.create(hub.Scheme, hub.Domain, hub.Port, hub.User, hub.Password, hub.ConcurrentScanLimit)
				if err != nil {
					log.Errorf("unable to create Hub client for %s: %s", hub.Domain, err.Error())
				}
			}
		}
	}()
	// 2. delete removed hubs
	for hubURL, hub := range hm.hubs {
		if _, ok := newHubURLs[hubURL]; !ok {
			hub.Stop()
			delete(hm.hubs, hubURL)
		}
	}
}

func (hm *HubManager) create(scheme string, host string, port int, username string, password string, concurrentScanLimit int) error {
	if _, ok := hm.hubs[host]; ok {
		return fmt.Errorf("cannot create hub %s: already exists", host)
	}
	hubClient, err := hm.newHub(scheme, host, port, username, password, concurrentScanLimit)
	if err != nil {
		return err
	}
	hm.hubs[host] = hubClient
	go func() {
		stop := hubClient.StopCh()
		updates := hubClient.Updates()
		for {
			select {
			case <-stop:
				return
			case nextUpdate := <-updates:
				hm.updates <- &Update{HubURL: host, Update: nextUpdate}
			}
		}
	}()
	return nil
}

// Updates returns a read-only channel of the combined update stream of each hub.
func (hm *HubManager) Updates() <-chan *Update {
	return hm.updates
}

// HubClients ...
func (hm *HubManager) HubClients() map[string]*hub.Hub {
	return hm.hubs
}

// StartScanClient ...
func (hm *HubManager) StartScanClient(hubURL string, scanName string) error {
	hub, ok := hm.hubs[hubURL]
	if !ok {
		return fmt.Errorf("hub %s not found", hubURL)
	}
	hub.StartScanClient(scanName)
	return nil
}

// FinishScanClient tells the appropriate hub client to start polling for
// scan completion.
func (hm *HubManager) FinishScanClient(hubURL string, scanName string, scanErr error) error {
	hub, ok := hm.hubs[hubURL]
	if !ok {
		return fmt.Errorf("hub %s not found", hubURL)
	}
	hub.FinishScanClient(scanName, scanErr)
	return nil
}

// ScanResults ...
func (hm *HubManager) ScanResults() map[string]map[string]*hub.Scan {
	allScanResults := map[string]map[string]*hub.Scan{}
	for hubURL, hub := range hm.hubs {
		// TODO could cache to avoid blocking
		allScanResults[hubURL] = <-hub.ScanResults()
	}
	return allScanResults
}

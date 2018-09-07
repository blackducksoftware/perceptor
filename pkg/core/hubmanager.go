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
	"time"

	"github.com/blackducksoftware/hub-client-go/hubclient"
	"github.com/blackducksoftware/perceptor/pkg/hub"
	log "github.com/sirupsen/logrus"
)

type hubClientCreator func(host string) (hub.ClientInterface, error)

func createMockHubClient(hubURL string) (hub.ClientInterface, error) {
	mockRawClient := hub.NewMockRawClient(false, []string{})
	return hub.NewClient("mock-username", "mock-password", hubURL, mockRawClient, 1*time.Minute, 30*time.Second, 999999*time.Hour), nil
}

func createHubClient(username string, password string, port int, httpTimeout time.Duration) hubClientCreator {
	return func(host string) (hub.ClientInterface, error) {
		baseURL := fmt.Sprintf("https://%s:%d", host, port)
		rawClient, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings, httpTimeout)
		if err != nil {
			return nil, err
		}
		return hub.NewClient(username, password, host, rawClient, 1*time.Minute, 30*time.Second, 999999*time.Hour), nil
	}
}

// Update is a wrapper around hub.Update which also tracks which Hub was the source.
type Update struct {
	HubURL string
	Update hub.Update
}

// HubManagerInterface ...
type HubManagerInterface interface {
	SetHubs(hubURLs []string)
	HubClients() map[string]hub.ClientInterface
	StartScanClient(hubURL string, scanName string) error
	FinishScanClient(hubURL string, scanName string) error
	ScanResults() map[string]map[string]*hub.ScanResults
	Updates() <-chan *Update
}

// HubManager ...
type HubManager struct {
	newHub hubClientCreator
	//
	stop    <-chan struct{}
	updates chan *Update
	//
	hubs                  map[string]hub.ClientInterface
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
		hubs:                  map[string]hub.ClientInterface{},
		didFetchScanResults:   make(chan *hub.ScanResults),
		didFetchCodeLocations: make(chan []string)}
}

// SetHubs ...
func (hm *HubManager) SetHubs(hubURLs []string) {
	newHubURLs := map[string]bool{}
	for _, hubURL := range hubURLs {
		newHubURLs[hubURL] = true
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
		for hubURL := range hubsToCreate {
			err := hm.create(hubURL)
			if err != nil {
				log.Errorf("unable to create Hub client for %s: %s", hubURL, err.Error())
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

func (hm *HubManager) create(hubURL string) error {
	if _, ok := hm.hubs[hubURL]; ok {
		return fmt.Errorf("cannot create hub %s: already exists", hubURL)
	}
	hubClient, err := hm.newHub(hubURL)
	if err != nil {
		return err
	}
	hm.hubs[hubURL] = hubClient
	go func() {
		stop := hubClient.StopCh()
		updates := hubClient.Updates()
		for {
			select {
			case <-stop:
				return
			case nextUpdate := <-updates:
				hm.updates <- &Update{HubURL: hubURL, Update: nextUpdate}
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
func (hm *HubManager) HubClients() map[string]hub.ClientInterface {
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
func (hm *HubManager) FinishScanClient(hubURL string, scanName string) error {
	hub, ok := hm.hubs[hubURL]
	if !ok {
		return fmt.Errorf("hub %s not found", hubURL)
	}
	hub.FinishScanClient(scanName)
	return nil
}

// ScanResults ...
func (hm *HubManager) ScanResults() map[string]map[string]*hub.ScanResults {
	allScanResults := map[string]map[string]*hub.ScanResults{}
	for hubURL, hub := range hm.hubs {
		// TODO could cache to avoid blocking
		allScanResults[hubURL] = <-hub.ScanResults()
	}
	return allScanResults
}

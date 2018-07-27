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

package federation

import (
	"fmt"
	"time"

	// "github.com/blackducksoftware/hub-client-go/hubapi"
	"github.com/blackducksoftware/hub-client-go/hubapi"
	"github.com/blackducksoftware/hub-client-go/hubclient"
	"github.com/blackducksoftware/perceptor/pkg/util"
	log "github.com/sirupsen/logrus"
)

const (
	maxHubExponentialBackoffDuration = 1 * time.Hour
	// hubDeleteTimeout                 = 1 * time.Hour
)

// Hub .....
type Hub struct {
	client *hubclient.Client
	// circuitBreaker *CircuitBreaker
	HubVersion string
	// hub credentials
	username string
	port     int
	password string
	baseURL  string
	// data
	codeLocationNames []string
	projectNames      []string
	errors            []error
	// TODO critical vulnerabilities
	// schedulers
	loginScheduler         *util.Scheduler
	fetchProjectsScheduler *util.Scheduler
	// pullCodeLocationsScheduler *util.Scheduler
	// channels
	stop                        chan struct{}
	resetCircuitBreakerCh       chan struct{}
	setTimeoutCh                chan time.Duration
	getProjectNamesCh           chan chan []string
	didLoginCh                  chan error
	didFetchProjectNamesCh      chan []string
	didFetchProjectNamesErrorCh chan error
	//getCodeLocationNamesCh chan chan []string
}

// NewHub returns a new, logged-in Hub.
// It will instead return an error if any of the following happen:
//  - unable to instantiate an API client
//  - unable to log in to the Hub
//  - unable to get hub version from the Hub
func NewHub(username string, password string, host string, port int, hubClientTimeout time.Duration) (*Hub, error) {
	baseURL := fmt.Sprintf("https://%s:%d", host, port)
	client, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings, hubClientTimeout)
	if err != nil {
		return nil, err
	}
	hub := Hub{
		client: client,
		// circuitBreaker: NewCircuitBreaker(maxHubExponentialBackoffDuration, client),
		username: username,
		password: password,
		port:     port,
		baseURL:  baseURL,
		//
		codeLocationNames: nil,
		projectNames:      nil,
		errors:            []error{},
		//
		stop: make(chan struct{}),
		resetCircuitBreakerCh:       make(chan struct{}),
		setTimeoutCh:                make(chan time.Duration),
		getProjectNamesCh:           make(chan chan []string),
		didLoginCh:                  make(chan error),
		didFetchProjectNamesCh:      make(chan []string),
		didFetchProjectNamesErrorCh: make(chan error)}
	err = hub.login()
	if err != nil {
		return nil, err
	}
	err = hub.fetchHubVersion()
	if err != nil {
		return nil, err
	}
	// action processing
	go func() {
		for {
			select {
			case timeout := <-hub.setTimeoutCh:
				hub.client.SetTimeout(timeout)
			case <-hub.resetCircuitBreakerCh:
				// TODO hub.circuitBreaker.Reset()
			case ch := <-hub.getProjectNamesCh:
				ch <- hub.projectNames
			case projectNames := <-hub.didFetchProjectNamesCh:
				hub.projectNames = projectNames
			case err := <-hub.didFetchProjectNamesErrorCh:
				// TODO don't let this grow without bounds
				hub.errors = append(hub.errors, err)
			case err := <-hub.didLoginCh:
				if err != nil {
					hub.errors = append(hub.errors, err)
				}
				// case ch := <-hub.getCodeLocationNamesCh:
				// 	ch <- hub.codeLocationNames
			}
		}
	}()
	hub.loginScheduler = hub.startLoginScheduler()
	hub.fetchProjectsScheduler = hub.startFetchProjectsScheduler()
	return &hub, nil
}

func (hub *Hub) Stop() {
	close(hub.stop)
}

func (hub *Hub) fetchHubVersion() error {
	// start := time.Now()
	currentVersion, err := hub.client.CurrentVersion()
	// recordHubResponse("version", err == nil)
	// recordHubResponseTime("version", time.Now().Sub(start))
	if err != nil {
		log.Errorf("unable to get hub version: %s", err.Error())
		return err
	}

	hub.HubVersion = currentVersion.Version
	log.Infof("successfully got hub version %s", hub.HubVersion)
	return nil
}

// // ResetCircuitBreaker ...
// func (hub *Hub) ResetCircuitBreaker() {
//   hub.resetCircuitBreakerCh <- struct{}
// }
//
// // IsEnabled returns whether the Hub is currently enabled
// // example: the circuit breaker is disabled -> the Hub is disabled
// func (hub *Hub) IsEnabled() <-chan bool {
// 	return hub.circuitBreaker.IsEnabledChannel
// }

func (hub *Hub) login() error {
	// start := time.Now()
	err := hub.client.Login(hub.username, hub.password)
	// recordHubResponse("login", err == nil)
	// recordHubResponseTime("login", time.Now().Sub(start))
	return err
}

// SetTimeout ...
func (hub *Hub) SetTimeout(timeout time.Duration) {
	hub.setTimeoutCh <- timeout
}

// // CodeLocationNames ...
// func (hub *Hub) CodeLocationNames() []string {
// 	ch := make(chan []string)
// 	hub.getCodeLocationNamesCh <- ch
// 	return <-ch
// }

// ProjectNames ...
func (hub *Hub) ProjectNames() []string {
	ch := make(chan []string)
	hub.getProjectNamesCh <- ch
	return <-ch
}

// Regular jobs

func (hub *Hub) startLoginScheduler() *util.Scheduler {
	pause := 30 * time.Minute
	return util.NewScheduler(pause, hub.stop, func() {
		err := hub.login()
		hub.didLoginCh <- err
		if err != nil {
			log.Errorf("unable to re-login to hub: %s", err.Error())
		} else {
			log.Infof("successfully re-logged in to hub %s", hub.baseURL)
		}
	})
}

func (hub *Hub) startFetchProjectsScheduler() *util.Scheduler {
	pause := 30 * time.Minute
	return util.NewScheduler(pause, hub.stop, func() {
		projectNames, err := hub.fetchAllProjectNames()
		if err == nil {
			log.Infof("successfully fetched %d project names from %s", len(projectNames), hub.baseURL)
			hub.didFetchProjectNamesCh <- projectNames
		} else {
			log.Errorf("unable to fetch project names:  %s", err.Error())
			hub.didFetchProjectNamesErrorCh <- err
		}
	})
}

// Hub api calls

// func (hub *Hub) fetchAllCodeLocationNames() ([]string, error) {
// 	options := &hubapi.GetListOptions{}
// 	codeLocationList, err := hub.client.ListAllCodeLocations(options) //circuitBreaker.ListAllCodeLocations()
// 	if err != nil {
// 		return nil, err
// 	}
// 	scanNames := make([]string, len(codeLocationList.Items))
// 	for i, cl := range codeLocationList.Items {
// 		scanNames[i] = cl.Name
// 	}
// 	return scanNames, nil
// }

func (hub *Hub) fetchAllProjectNames() ([]string, error) {
	options := &hubapi.GetListOptions{}
	projectList, err := hub.client.ListProjects(options) //circuitBreaker.ListAllProjects()
	if err != nil {
		return nil, err
	}
	projectNames := make([]string, len(projectList.Items))
	for i, proj := range projectList.Items {
		projectNames[i] = proj.Name
	}
	return projectNames, nil
}

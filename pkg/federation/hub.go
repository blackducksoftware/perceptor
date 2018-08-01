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
	codeLocations map[string]string
	projects      map[string]string
	errors        []error
	// TODO critical vulnerabilities
	// schedulers
	loginScheduler              *util.Scheduler
	fetchProjectsScheduler      *util.Scheduler
	fetchCodeLocationsScheduler *util.Scheduler
	// channels
	stop                         chan struct{}
	resetCircuitBreakerCh        chan struct{}
	setTimeoutCh                 chan time.Duration
	getCodeLocationsCh           chan chan map[string]string
	getProjectsCh                chan chan map[string]string
	didLoginCh                   chan error
	didFetchCodeLocationsCh      chan map[string]string
	didFetchCodeLocationsErrorCh chan error
	didFetchProjectsCh           chan map[string]string
	didFetchProjectsErrorCh      chan error
}

// NewHub returns a new, logged-in Hub.
// It will instead return an error if any of the following happen:
//  - unable to instantiate an API client
//  - unable to log in to the Hub
//  - unable to get hub version from the Hub
func NewHub(username string, password string, host string, port int, hubClientTimeout time.Duration, fetchAllProjectsPause time.Duration) (*Hub, error) {
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
		codeLocations: nil,
		projects:      nil,
		errors:        []error{},
		//
		stop: make(chan struct{}),
		resetCircuitBreakerCh:        make(chan struct{}),
		setTimeoutCh:                 make(chan time.Duration),
		getProjectsCh:                make(chan chan map[string]string),
		getCodeLocationsCh:           make(chan chan map[string]string),
		didLoginCh:                   make(chan error),
		didFetchCodeLocationsCh:      make(chan map[string]string),
		didFetchCodeLocationsErrorCh: make(chan error),
		didFetchProjectsCh:           make(chan map[string]string),
		didFetchProjectsErrorCh:      make(chan error)}
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
			case ch := <-hub.getProjectsCh:
				ch <- hub.projects
			case projects := <-hub.didFetchProjectsCh:
				hub.projects = projects
			case codeLocations := <-hub.didFetchCodeLocationsCh:
				hub.codeLocations = codeLocations
			case err := <-hub.didFetchProjectsErrorCh:
				// TODO don't let this grow without bounds
				hub.errors = append(hub.errors, err)
			case err := <-hub.didLoginCh:
				if err != nil {
					hub.errors = append(hub.errors, err)
				}
			case ch := <-hub.getCodeLocationsCh:
				ch <- hub.codeLocations
			case err := <-hub.didFetchCodeLocationsErrorCh:
				hub.errors = append(hub.errors, err)
			}
		}
	}()
	hub.loginScheduler = hub.startLoginScheduler()
	hub.fetchProjectsScheduler = hub.startFetchProjectsScheduler(fetchAllProjectsPause)
	hub.fetchCodeLocationsScheduler = hub.startFetchCodeLocationsScheduler(fetchAllProjectsPause)
	return &hub, nil
}

// Stop ...
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

// CodeLocations ...
func (hub *Hub) CodeLocations() map[string]string {
	ch := make(chan map[string]string)
	hub.getCodeLocationsCh <- ch
	return <-ch
}

// Projects ...
func (hub *Hub) Projects() map[string]string {
	ch := make(chan map[string]string)
	hub.getProjectsCh <- ch
	return <-ch
}

// Regular jobs

func (hub *Hub) startLoginScheduler() *util.Scheduler {
	pause := 30 * time.Minute
	return util.NewScheduler(pause, hub.stop, false, func() {
		err := hub.login()
		hub.didLoginCh <- err
		if err != nil {
			log.Errorf("unable to re-login to hub: %s", err.Error())
		} else {
			log.Infof("successfully re-logged in to hub %s", hub.baseURL)
		}
	})
}

func (hub *Hub) startFetchProjectsScheduler(pause time.Duration) *util.Scheduler {
	return util.NewScheduler(pause, hub.stop, true, func() {
		log.Debugf("starting to fetch all project")
		projects, err := hub.fetchAllProjects()
		if err == nil {
			log.Infof("successfully fetched %d projects from %s", len(projects), hub.baseURL)
			hub.didFetchProjectsCh <- projects
		} else {
			log.Errorf("unable to fetch projects:  %s", err.Error())
			hub.didFetchProjectsErrorCh <- err
		}
	})
}

func (hub *Hub) startFetchCodeLocationsScheduler(pause time.Duration) *util.Scheduler {
	return util.NewScheduler(pause, hub.stop, true, func() {
		log.Debugf("starting to fetch all code locations")
		cls, err := hub.fetchAllCodeLocations()
		if err == nil {
			log.Infof("successfully fetched %d cls from %s", len(cls), hub.baseURL)
			hub.didFetchCodeLocationsCh <- cls
		} else {
			log.Errorf("unable to fetch code locations:  %s", err.Error())
			hub.didFetchCodeLocationsErrorCh <- err
		}
	})
}

// Hub api calls

func (hub *Hub) fetchAllCodeLocations() (map[string]string, error) {
	limit := 2000000
	options := &hubapi.GetListOptions{Limit: &limit}
	codeLocationList, err := hub.client.ListAllCodeLocations(options) //circuitBreaker.ListAllCodeLocations()
	if err != nil {
		return nil, err
	}
	cls := map[string]string{}
	for _, cl := range codeLocationList.Items {
		cls[cl.Name] = cl.MappedProjectVersion
	}
	return cls, nil
}

func (hub *Hub) fetchAllProjects() (map[string]string, error) {
	limit := 2000000
	options := &hubapi.GetListOptions{Limit: &limit}
	projectList, err := hub.client.ListProjects(options) //circuitBreaker.ListAllProjects()
	if err != nil {
		return nil, err
	}
	projects := map[string]string{}
	for _, proj := range projectList.Items {
		projects[proj.Name] = proj.Meta.Href
	}
	return projects, nil
}

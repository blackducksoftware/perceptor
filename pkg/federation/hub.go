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

	h "github.com/blackducksoftware/perceptor/pkg/hub"
	"github.com/blackducksoftware/perceptor/pkg/util"
	log "github.com/sirupsen/logrus"
)

const (
	maxHubExponentialBackoffDuration = 1 * time.Hour
	// hubDeleteTimeout                 = 1 * time.Hour
)

// HubStatus describes the state of a hub client
type HubStatus int

// .....
const (
	HubStatusInitializing HubStatus = iota
	HubStatusError        HubStatus = iota
	HubStatusUp           HubStatus = iota
	HubStatusDown         HubStatus = iota
)

// String .....
func (status HubStatus) String() string {
	switch status {
	case HubStatusInitializing:
		return "HubStatusInitializing"
	case HubStatusError:
		return "HubStatusError"
	case HubStatusUp:
		return "HubStatusUp"
	case HubStatusDown:
		return "HubStatusDown"
	}
	panic(fmt.Errorf("invalid HubStatus value: %d", status))
}

// MarshalJSON .....
func (status HubStatus) MarshalJSON() ([]byte, error) {
	jsonString := fmt.Sprintf(`"%s"`, status.String())
	return []byte(jsonString), nil
}

// MarshalText .....
func (status HubStatus) MarshalText() (text []byte, err error) {
	return []byte(status.String()), nil
}

// Hub .....
type Hub struct {
	fetcher *h.Fetcher
	// TODO add a second hub client -- so that there's one for rare, slow requests (all projects,
	//   all code locations) and one for frequent, quick requests
	hubStatus HubStatus
	host      string
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
	stop                    chan struct{}
	resetCircuitBreakerCh   chan struct{}
	getModel                chan chan *APIModelHub
	getCodeLocationsCh      chan chan map[string]string
	getProjectsCh           chan chan map[string]string
	didLoginCh              chan error
	didFetchCodeLocationsCh chan *fetchCodeLocationsResult
	didFetchProjectsCh      chan *fetchProjectsResult
}

// NewHub returns a new, logged-in Hub.
// It will instead return an error if any of the following happen:
//  - unable to instantiate an API client
//  - unable to log in to the Hub
//  - unable to get hub version from the Hub
func NewHub(username string, password string, host string, port int, hubClientTimeout time.Duration, fetchAllProjectsPause time.Duration) *Hub {
	hub := &Hub{
		host: host,
		//
		codeLocations: nil,
		projects:      nil,
		errors:        []error{},
		//
		stop: make(chan struct{}),
		resetCircuitBreakerCh:   make(chan struct{}),
		getModel:                make(chan chan *APIModelHub),
		getProjectsCh:           make(chan chan map[string]string),
		getCodeLocationsCh:      make(chan chan map[string]string),
		didLoginCh:              make(chan error),
		didFetchCodeLocationsCh: make(chan *fetchCodeLocationsResult),
		didFetchProjectsCh:      make(chan *fetchProjectsResult)}
	// initialize hub client
	fetcher, err := h.NewFetcher(username, password, host, port, hubClientTimeout)
	if err != nil {
		hub.hubStatus = HubStatusError
		hub.errors = append(hub.errors, err)
		return hub
	}
	hub.fetcher = fetcher
	// action processing
	go func() {
		for {
			select {
			case <-hub.resetCircuitBreakerCh:
				// TODO hub.circuitBreaker.Reset()
			case ch := <-hub.getModel:
				ch <- hub.apiModel()
			case ch := <-hub.getProjectsCh:
				ch <- hub.projects
			case ch := <-hub.getCodeLocationsCh:
				ch <- hub.codeLocations
			case result := <-hub.didFetchProjectsCh:
				hub.recordError(result.err)
				if result.err == nil {
					hub.projects = result.projects
				}
			case result := <-hub.didFetchCodeLocationsCh:
				hub.recordError(result.err)
				if result.err == nil {
					hub.codeLocations = result.codeLocations
				}
			case err := <-hub.didLoginCh:
				hub.recordError(err)
				if err != nil {
					hub.recordError(hub.fetchProjectsScheduler.Pause())
					hub.recordError(hub.fetchCodeLocationsScheduler.Pause())
				} else {
					hub.recordError(hub.fetchProjectsScheduler.Resume(true))
					hub.recordError(hub.fetchCodeLocationsScheduler.Resume(true))
				}
			}
		}
	}()
	hub.loginScheduler = hub.startLoginScheduler()
	hub.fetchProjectsScheduler = hub.startFetchProjectsScheduler(fetchAllProjectsPause)
	hub.fetchCodeLocationsScheduler = hub.startFetchCodeLocationsScheduler(fetchAllProjectsPause)
	hub.hubStatus = HubStatusUp
	return hub
}

// Stop ...
func (hub *Hub) Stop() {
	close(hub.stop)
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

// TODO these aren't threadsafe ... they're just thread-safely grabbing references to
// these mutable objects, then throwing them out into the wild.  Oops!
// Maybe re-enable these later, but make them ACTUALLY threadsafe :) :)
// // CodeLocations ...
// func (hub *Hub) CodeLocations() map[string]string {
// 	ch := make(chan map[string]string)
// 	hub.getCodeLocationsCh <- ch
// 	return <-ch
// }

// // Projects ...
// func (hub *Hub) Projects() map[string]string {
// 	ch := make(chan map[string]string)
// 	hub.getProjectsCh <- ch
// 	return <-ch
// }

// Model ...
func (hub *Hub) Model() *APIModelHub {
	ch := make(chan *APIModelHub)
	hub.getModel <- ch
	return <-ch
}

// Private methods

func (hub *Hub) recordError(err error) {
	// TODO don't let this grow without bounds
	if err != nil {
		hub.errors = append(hub.errors, err)
	}
}

func (hub *Hub) login() error {
	return hub.fetcher.Login()
}

func (hub *Hub) apiModel() *APIModelHub {
	errors := make([]string, len(hub.errors))
	for ix, err := range hub.errors {
		errors[ix] = err.Error()
	}
	projects := map[string]string{}
	for name, url := range hub.projects {
		projects[name] = url
	}
	codeLocations := map[string]string{}
	for name, url := range hub.codeLocations {
		codeLocations[name] = url
	}
	return &APIModelHub{
		Errors:                  errors,
		HasLoadedAllProjects:    hub.projects != nil,
		Status:                  hub.hubStatus.String(),
		IsCircuitBreakerEnabled: false, // TODO
		IsLoggedIn:              false, // TODO
		Projects:                projects,
		CodeLocations:           codeLocations,
	}
}

// Regular jobs

func (hub *Hub) startLoginScheduler() *util.Scheduler {
	//	pause := 3 * time.Minute
	pause := 3 * time.Second
	return util.NewRunningScheduler(pause, hub.stop, true, func() {
		log.Debugf("starting to login to hub")
		err := hub.login()
		hub.didLoginCh <- err
	})
}

func (hub *Hub) startFetchProjectsScheduler(pause time.Duration) *util.Scheduler {
	return util.NewRunningScheduler(pause, hub.stop, true, func() {
		log.Debugf("starting to fetch all projects")
		result := hub.fetchAllProjects()
		hub.didFetchProjectsCh <- result
	})
}

func (hub *Hub) startFetchCodeLocationsScheduler(pause time.Duration) *util.Scheduler {
	return util.NewRunningScheduler(pause, hub.stop, true, func() {
		log.Debugf("starting to fetch all code locations")
		result := hub.fetchAllCodeLocations()
		hub.didFetchCodeLocationsCh <- result
	})
}

// Hub api calls

type fetchCodeLocationsResult struct {
	codeLocations map[string]string
	err           error
}

func (hub *Hub) fetchAllCodeLocations() *fetchCodeLocationsResult {
	codeLocationList, err := hub.fetcher.ListAllCodeLocations()
	log.Debugf("fetched all code locations: %+v, %+v", codeLocationList, err)
	if err != nil {
		return &fetchCodeLocationsResult{codeLocations: nil, err: err}
	}
	cls := map[string]string{}
	for _, cl := range codeLocationList.Items {
		cls[cl.Name] = cl.MappedProjectVersion
	}
	return &fetchCodeLocationsResult{codeLocations: cls, err: nil}
}

type fetchProjectsResult struct {
	projects map[string]string
	err      error
}

func (hub *Hub) fetchAllProjects() *fetchProjectsResult {
	projectList, err := hub.fetcher.ListAllProjects()
	log.Debugf("fetched all projects: %+v, %+v", projectList, err)
	if err != nil {
		return &fetchProjectsResult{projects: nil, err: err}
	}
	projects := map[string]string{}
	for _, proj := range projectList.Items {
		projects[proj.Name] = proj.Meta.Href
	}
	return &fetchProjectsResult{projects: projects, err: nil}
}

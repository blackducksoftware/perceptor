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
	hubDeleteTimeout                 = 1 * time.Hour
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
	// TODO critical vulnerabilities
	// schedulers
	loginScheduler             *util.Scheduler
	pullProjectsScheduler      *util.Scheduler
	pullCodeLocationsScheduler *util.Scheduler
	// channels
	stop                   <-chan struct{}
	resetCircuitBreakerCh  chan struct{}
	setTimeoutCh           chan time.Duration
	getProjectNamesCh      chan chan []string
	getCodeLocationNamesCh chan chan []string
}

// NewHub returns a new, logged-in Hub.
// It will instead return an error if any of the following happen:
//  - unable to instantiate an API client
//  - unable to sign in to the Hub
//  - unable to get hub version from the Hub
func NewHub(username string, password string, host string, port int, hubClientTimeout time.Duration, stop <-chan struct{}) (*Hub, error) {
	baseURL := fmt.Sprintf("https://%s:%d", host, port)
	client, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings, hubClientTimeout)
	if err != nil {
		return nil, err
	}
	hf := Hub{
		client: client,
		// circuitBreaker: NewCircuitBreaker(maxHubExponentialBackoffDuration, client),
		username:          username,
		password:          password,
		port:              port,
		baseURL:           baseURL,
		codeLocationNames: nil,
		projectNames:      nil,
		stop:              stop}
	err = hf.login()
	if err != nil {
		return nil, err
	}
	err = hf.fetchHubVersion()
	if err != nil {
		return nil, err
	}
	// action processing
	go func() {
		for {
			select {
			case timeout := <-hf.setTimeoutCh:
				hf.client.SetTimeout(timeout)
			case <-hf.resetCircuitBreakerCh:
				// TODO hf.circuitBreaker.Reset()
			case ch := <-hf.getProjectNamesCh:
				ch <- hf.projectNames
			case ch := <-hf.getCodeLocationNamesCh:
				ch <- hf.codeLocationNames
			}
		}
	}()
	// TODO start up schedulers
	return &hf, nil
}

func (hf *Hub) fetchHubVersion() error {
	// start := time.Now()
	currentVersion, err := hf.client.CurrentVersion()
	// recordHubResponse("version", err == nil)
	// recordHubResponseTime("version", time.Now().Sub(start))
	if err != nil {
		log.Errorf("unable to get hub version: %s", err.Error())
		return err
	}

	hf.HubVersion = currentVersion.Version
	log.Infof("successfully got hub version %s", hf.HubVersion)
	return nil
}

// // ResetCircuitBreaker ...
// func (hf *Hub) ResetCircuitBreaker() {
//   hf.resetCircuitBreakerCh <- struct{}
// }
//
// // IsEnabled returns whether the Hub is currently enabled
// // example: the circuit breaker is disabled -> the Hub is disabled
// func (hf *Hub) IsEnabled() <-chan bool {
// 	return hf.circuitBreaker.IsEnabledChannel
// }

func (hf *Hub) login() error {
	// start := time.Now()
	err := hf.client.Login(hf.username, hf.password)
	// recordHubResponse("login", err == nil)
	// recordHubResponseTime("login", time.Now().Sub(start))
	if err != nil {
		return err
	}
	return err
}

// SetTimeout ...
func (hf *Hub) SetTimeout(timeout time.Duration) {
	hf.setTimeoutCh <- timeout
}

// CodeLocationNames ...
func (hf *Hub) CodeLocationNames() []string {
	ch := make(chan []string)
	hf.getCodeLocationNamesCh <- ch
	return <-ch
}

// ProjectNames ...
func (hf *Hub) ProjectNames() []string {
	ch := make(chan []string)
	hf.getProjectNamesCh <- ch
	return <-ch
}

// Regular jobs

func (hf *Hub) startLoginScheduler() *util.Scheduler {
	pause := 30 * time.Minute
	return util.NewScheduler(pause, hf.stop, func() {
		err := hf.login()
		if err != nil {
			log.Errorf("unable to re-login to hub: %s", err.Error())
		}
		log.Infof("successfully re-logged in to hub")
	})
}

func (hf *Hub) fetchAllCodeLocationNames() ([]string, error) {
	options := &hubapi.GetListOptions{}
	codeLocationList, err := hf.client.ListAllCodeLocations(options) //circuitBreaker.ListAllCodeLocations()
	if err != nil {
		return nil, err
	}
	scanNames := make([]string, len(codeLocationList.Items))
	for i, cl := range codeLocationList.Items {
		scanNames[i] = cl.Name
	}
	return scanNames, nil
}

func (hf *Hub) fetchAllProjectNames() ([]string, error) {
	options := &hubapi.GetListOptions{}
	projectList, err := hf.client.ListProjects(options) //circuitBreaker.ListAllProjects()
	if err != nil {
		return nil, err
	}
	projectNames := make([]string, len(projectList.Items))
	for i, proj := range projectList.Items {
		projectNames[i] = proj.Name
	}
	return projectNames, nil
}

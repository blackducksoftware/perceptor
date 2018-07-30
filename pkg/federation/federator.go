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
	"os"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	actionChannelSize = 100
)

// Federator ...
type Federator struct {
	responder *HTTPResponder
	// model
	config      *Config
	hubPassword string
	hubs        map[string]*Hub
	// channels
	stop    chan struct{}
	actions chan FedAction
}

// NewFederator ...
func NewFederator(config *Config) (*Federator, error) {
	responder := NewHTTPResponder()
	SetupHTTPServer(responder)
	hubPassword, ok := os.LookupEnv(config.HubConfig.PasswordEnvVar)
	if !ok {
		return nil, fmt.Errorf("unable to get Hub password: environment variable %s not set", config.HubConfig.PasswordEnvVar)
	}
	actions := make(chan FedAction, actionChannelSize)
	go func() {
		for {
			select {
			case a := <-responder.RequestsCh:
				actions <- a
			}
		}
	}()
	fed := &Federator{
		responder:   responder,
		config:      config,
		hubPassword: hubPassword,
		hubs:        map[string]*Hub{},
		stop:        make(chan struct{}),
		actions:     actions}
	go func() {
		for {
			a := <-actions
			log.Debugf("received action %s", reflect.TypeOf(a))
			start := time.Now()
			a.FedApply(fed)
			stop := time.Now()
			log.Debugf("finished processing action -- %s", stop.Sub(start))
		}
	}()
	return fed, nil
}

func (fed *Federator) setHubs(hubURLs []string) {
	newHubURLs := map[string]bool{}
	for _, hubURL := range hubURLs {
		newHubURLs[hubURL] = true
	}
	// 1. create new hubs
	hubConfig := fed.config.HubConfig
	// TODO move this into a 'HubCreationManager' or something that can handle
	// retries and failures intelligently
	for hubURL := range newHubURLs {
		if _, ok := fed.hubs[hubURL]; !ok {
			hub, err := NewHub(hubConfig.User, fed.hubPassword, hubURL, hubConfig.Port, hubConfig.ClientTimeout())
			if err == nil {
				fed.hubs[hubURL] = hub
			} else {
				log.Errorf("unable to create Hub for URL %s: %s", hubURL, err.Error())
			}
		}
	}
	// 2. delete removed hubs
	// TODO separate this into a delete manager, handling failures and retries
	for hubURL, hub := range fed.hubs {
		if _, ok := newHubURLs[hubURL]; !ok {
			hub.Stop()
			delete(fed.hubs, hubURL)
			// TODO does any other clean up need to happen?
		}
	}
}

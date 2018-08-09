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
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	actionChannelSize = 100
)

// Federator ...
type Federator struct {
	responder  *HTTPResponder
	hubCreator *HubCreator
	// model
	config *Config
	hubs   map[string]*Hub
	// channels
	stop    chan struct{}
	actions chan FedAction
}

// NewFederator ...
func NewFederator(config *Config) (*Federator, error) {
	responder := NewHTTPResponder()
	SetupHTTPServer(responder)
	hubCreator, err := NewHubCreator(config.HubConfig)
	if err != nil {
		return nil, err
	}
	actions := make(chan FedAction, actionChannelSize)
	// dump events into 'actions' queue
	go func() {
		for {
			select {
			case a := <-responder.RequestsCh:
				actions <- a
			case d := <-hubCreator.didFinishHubCreation:
				actions <- d
			}
		}
	}()
	fed := &Federator{
		responder:  responder,
		hubCreator: hubCreator,
		config:     config,
		hubs:       map[string]*Hub{},
		stop:       make(chan struct{}),
		actions:    actions}
	// process actions
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
	hubsToCreate := map[string]bool{}
	for hubURL := range newHubURLs {
		if _, ok := fed.hubs[hubURL]; !ok {
			hubsToCreate[hubURL] = true
		}
	}
	// 1. create new hubs
	// TODO handle retries and failures intelligently
	go func() {
		fed.hubCreator.createHubs(hubsToCreate)
	}()
	// 2. delete removed hubs
	for hubURL, hub := range fed.hubs {
		if _, ok := newHubURLs[hubURL]; !ok {
			hub.Stop()
			delete(fed.hubs, hubURL)
			// TODO does any other clean up need to happen?
		}
	}
}

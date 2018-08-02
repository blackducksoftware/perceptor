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

import log "github.com/sirupsen/logrus"

// FedAction ...
type FedAction interface {
	FedApply(federator *Federator)
}

// FedGetModel ...
type FedGetModel struct {
	Done chan *APIModel
}

// NewFedGetModel ...
func NewFedGetModel() *FedGetModel {
	return &FedGetModel{Done: make(chan *APIModel)}
}

// FedApply ...
func (fgm *FedGetModel) FedApply(federator *Federator) {
	hubs := map[string]*APIModelHub{}
	for hubURL, hub := range federator.hubs {
		errors := []string{}
		for ix, err := range hub.errors {
			errors[ix] = err.Error()
		}
		projects := map[string]string{}
		for name, url := range hub.Projects() {
			projects[name] = url
		}
		codeLocations := map[string]string{}
		for name, url := range hub.CodeLocations() {
			codeLocations[name] = url
		}
		hubs[hubURL] = &APIModelHub{
			Errors:                  errors,
			HasLoadedAllProjects:    hub.projects != nil,
			Status:                  hub.hubStatus.String(),
			IsCircuitBreakerEnabled: false, // TODO
			IsLoggedIn:              false, // TODO
			Projects:                projects,
			CodeLocations:           codeLocations,
		}
	}
	model := &APIModel{Hubs: hubs}
	fgm.Done <- model
}

// FedSetHubs ...
type FedSetHubs struct {
	HubBaseURLs []string
}

// FedApply ...
func (fsh *FedSetHubs) FedApply(federator *Federator) {
	federator.setHubs(fsh.HubBaseURLs)
}

// FedFindProject ...
type FedFindProject struct {
	Request  APIProjectSearchRequest
	Response chan *APIProjectSearchResponse
}

// NewFedFindProject ...
func NewFedFindProject(request APIProjectSearchRequest) *FedFindProject {
	return &FedFindProject{Request: request, Response: make(chan *APIProjectSearchResponse)}
}

// FedApply ...
func (ffp *FedFindProject) FedApply(federator *Federator) {
	// TODO talk to all the hubs, ask them for their projects;
	// get back:
	//  - a list of each matching project
	//  - a list of hubs with problems or which are not yet initialized
	//    (in case that would be relevant to the user -- i.e. that their project
	//     *might* be present, but we don't know)
}

// FedUpdateConfig ...
type FedUpdateConfig struct {
}

// FedApply ...
func (fconf *FedUpdateConfig) FedApply(federator *Federator) {
	// TODO
}

// HubCreationResult ...
type HubCreationResult struct {
	hub *Hub
}

// FedApply ...
func (hcr *HubCreationResult) FedApply(federator *Federator) {
	if _, ok := federator.hubs[hcr.hub.baseURL]; ok {
		log.Errorf("cannot add hub %s: already present", hcr.hub.baseURL)
		return
	}
	federator.hubs[hcr.hub.baseURL] = hcr.hub
}

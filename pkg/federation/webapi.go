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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// SetupHTTPServer .....
func SetupHTTPServer(responder Responder) {
	// state of the program
	http.HandleFunc("/model", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			jsonBytes, err := json.MarshalIndent(responder.GetModel(), "", "  ")
			if err != nil {
				responder.Error(w, r, err, 500)
				return
			}
			header := w.Header()
			header.Set(http.CanonicalHeaderKey("content-type"), "application/json")
			fmt.Fprint(w, string(jsonBytes))
		} else {
			responder.NotFound(w, r)
		}
	})

	http.HandleFunc("/sethubs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			var setHubs APISetHubsRequest
			err = json.Unmarshal(body, &setHubs)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			responder.SetHubs(&setHubs)
		} else {
			responder.NotFound(w, r)
		}
	})

	http.HandleFunc("/findproject", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			var request APIProjectSearchRequest
			err = json.Unmarshal(body, &request)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			projects := responder.FindProject(request)
			jsonBytes, err := json.MarshalIndent(projects, "", "  ")
			if err != nil {
				responder.Error(w, r, err, 500)
			} else {
				header := w.Header()
				header.Set(http.CanonicalHeaderKey("content-type"), "application/json")
				fmt.Fprint(w, string(jsonBytes))
			}
		} else {
			responder.NotFound(w, r)
		}
	})

	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			var request APIUpdateConfigRequest
			err = json.Unmarshal(body, &request)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			responder.UpdateConfig(&request)
			fmt.Fprint(w, "")
		} else {
			responder.NotFound(w, r)
		}
	})
}

// Responder .....
type Responder interface {
	GetModel() *APIModel
	//
	SetHubs(hubs *APISetHubsRequest)
	UpdateConfig(config *APIUpdateConfigRequest)
	//
	FindProject(request APIProjectSearchRequest) *APIProjectSearchResponse
	// errors
	NotFound(w http.ResponseWriter, r *http.Request)
	Error(w http.ResponseWriter, r *http.Request, err error, statusCode int)
}

type MockResponder struct{}

func NewMockResponder() *MockResponder {
	return &MockResponder{}
}

func (mr *MockResponder) GetModel() *APIModel {
	return &APIModel{Hubs: map[string]*APIModelHub{
		"http://blackducksoftware/com": &APIModelHub{
			HasLoadedAllProjects:    false,
			IsCircuitBreakerEnabled: false,
			IsLoggedIn:              false,
			ProjectCount:            0,
			Projects:                map[string]string{},
		},
	}}
}

func (mr *MockResponder) SetHubs(hubs *APISetHubsRequest)             {}
func (mr *MockResponder) UpdateConfig(config *APIUpdateConfigRequest) {}

func (mr *MockResponder) FindProject(request APIProjectSearchRequest) *APIProjectSearchResponse {
	return &APIProjectSearchResponse{}
}

func (mr *MockResponder) NotFound(w http.ResponseWriter, r *http.Request)                         {}
func (mr *MockResponder) Error(w http.ResponseWriter, r *http.Request, err error, statusCode int) {}

type APIModel struct {
	// map of hub URL to ... ? hub info?
	Hubs map[string]*APIModelHub
}

type APISetHubsRequest struct {
	HubURLs []string
}

type APIUpdateConfigRequest struct {
	// TODO anything we need here?
}

type APIProjectSearchRequest struct {
	ProjectName string
}
type APIProjectSearchResponse struct {
	Projects []*APIProject
}

type APIProject struct {
	Name string
	URL  string
}

type APIModelHub struct {
	// can we log in to the hub?
	IsLoggedIn bool
	// have all the projects been sucked in?
	HasLoadedAllProjects bool
	ProjectCount         int
	// is circuit breaker enabled?
	IsCircuitBreakerEnabled bool
	// map of project name to ... ? hub URL?
	Projects map[string]string
	// bad things that have happened
	Errors []string
}

// HTTPResponder ...
type HTTPResponder struct {
	RequestsCh chan FedAction
}

// NewHTTPResponder .....
func NewHTTPResponder() *HTTPResponder {
	return &HTTPResponder{RequestsCh: make(chan FedAction)}
}

// GetModel .....
func (hr *HTTPResponder) GetModel() *APIModel {
	get := &FedGetModel{}
	hr.RequestsCh <- get
	return <-get.Done
}

func (hr *HTTPResponder) SetHubs(hubs *APISetHubsRequest) {
	hr.RequestsCh <- &FedSetHubs{HubBaseURLs: hubs.HubURLs}
}
func (hr *HTTPResponder) UpdateConfig(config *APIUpdateConfigRequest) {}

func (hr *HTTPResponder) FindProject(request APIProjectSearchRequest) *APIProjectSearchResponse {
	return &APIProjectSearchResponse{}
}

// errors

// NotFound .....
func (hr *HTTPResponder) NotFound(w http.ResponseWriter, r *http.Request) {
	log.Errorf("HTTPResponder not found from request %+v", r)
	recordHTTPNotFound(r)
	http.NotFound(w, r)
}

// Error .....
func (hr *HTTPResponder) Error(w http.ResponseWriter, r *http.Request, err error, statusCode int) {
	log.Errorf("HTTPResponder error %s with code %d from request %+v", err.Error(), statusCode, r)
	recordHTTPError(r, err, statusCode)
	http.Error(w, err.Error(), statusCode)
}

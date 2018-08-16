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

	"github.com/blackducksoftware/perceptor/pkg/api"
	"github.com/blackducksoftware/perceptor/pkg/util"

	log "github.com/sirupsen/logrus"
)

// SetupHTTPServer .....
func SetupHTTPServer(responder Responder) {
	// state of the program
	http.HandleFunc("/model", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			log.Debugf("http request: GET model")
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
			log.Debugf("http request: PUT sethubs")
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
			log.Debugf("http request: POST findproject")
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

	http.HandleFunc("/stackdump", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			log.Debugf("http request: GET stackdump")
			runtimeStack := util.DumpRuntimeStack()
			pprofStack, grsCount := util.DumpPProfStack()
			heap, heapCount := util.DumpHeap()
			dict := map[string]interface{}{
				"runtime":    runtimeStack,
				"pprof":      pprofStack,
				"pprofCount": grsCount,
				"heap":       heap,
				"heapCount":  heapCount,
			}
			fmt.Printf("runtime:\n%s\n\n", runtimeStack)
			fmt.Printf("pprof: %d\n%s\n\n", grsCount, pprofStack)
			fmt.Printf("heap: %d\n%s\n\n", heapCount, heap)
			//			log.Printf()
			//			log.Debugf("runtime: %s", )
			jsonBytes, err := json.MarshalIndent(dict, "", "  ")
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
			log.Debugf("http request: POST config")
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

// MockResponder ...
type MockResponder struct{}

// NewMockResponder ...
func NewMockResponder() *MockResponder {
	return &MockResponder{}
}

// GetModel ...
func (mr *MockResponder) GetModel() *APIModel {
	return &APIModel{Hubs: map[string]*api.HubModel{
		"http://blackducksoftware/com": {
			//			HasLoadedAllProjects:    false,
			IsCircuitBreakerEnabled: false,
			IsLoggedIn:              false,
			//			Projects:                map[string]string{},
		},
	}}
}

// SetHubs ...
func (mr *MockResponder) SetHubs(hubs *APISetHubsRequest) {}

// UpdateConfig ...
func (mr *MockResponder) UpdateConfig(config *APIUpdateConfigRequest) {}

// FindProject ...
func (mr *MockResponder) FindProject(request APIProjectSearchRequest) *APIProjectSearchResponse {
	return &APIProjectSearchResponse{}
}

// NotFound ...
func (mr *MockResponder) NotFound(w http.ResponseWriter, r *http.Request) {}

// Error ...
func (mr *MockResponder) Error(w http.ResponseWriter, r *http.Request, err error, statusCode int) {}

// APIModel ...
type APIModel struct {
	// map of hub URL to ... ? hub info?
	Hubs map[string]*api.HubModel
}

// APISetHubsRequest ...
type APISetHubsRequest struct {
	HubURLs []string
}

// APIUpdateConfigRequest ...
type APIUpdateConfigRequest struct {
	// TODO anything we need here?
}

// APIProjectSearchRequest ...
type APIProjectSearchRequest struct {
	ProjectName string
}

// APIProjectSearchResponse ...
type APIProjectSearchResponse struct {
	// map of hubBaseURL to project
	Projects map[string]*APIProject
}

// APIProject ...
type APIProject struct {
	Name string
	URL  string
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
	get := NewFedGetModel()
	go func() {
		hr.RequestsCh <- get
	}()
	return <-get.Done
}

// SetHubs ...
func (hr *HTTPResponder) SetHubs(hubs *APISetHubsRequest) {
	hr.RequestsCh <- &FedSetHubs{HubBaseURLs: hubs.HubURLs}
}

// UpdateConfig ...
func (hr *HTTPResponder) UpdateConfig(config *APIUpdateConfigRequest) {
	hr.RequestsCh <- &FedUpdateConfig{}
}

// FindProject ...
func (hr *HTTPResponder) FindProject(request APIProjectSearchRequest) *APIProjectSearchResponse {
	req := NewFedFindProject(request)
	hr.RequestsCh <- req
	return <-req.Response
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

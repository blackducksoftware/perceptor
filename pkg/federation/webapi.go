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
			responder.PostFinishScan(scanResults)
			fmt.Fprint(w, "")
		} else {
			responder.NotFound(w, r)
		}
	})
}

// Responder .....
type Responder interface {
	GetModel() APIModel
	//
	SetHubs(hubs *APISetHubsRequest)
	UpdateConfig(config *APIUpdateConfigRequest)
	//
	FindProject(request APIProjectSearchRequest) APIProjectSearchResponse
	// errors
	NotFound(w http.ResponseWriter, r *http.Request)
	Error(w http.ResponseWriter, r *http.Request, err error, statusCode int)
}

type APIModel struct {
	// map of project name to ... ? hub URL?
	Projects map[string]string
	// map of hub URL to ... ? hub info?
	Hubs map[string]*APIModelHub
}

type APISetHubsRequest struct{}

type APIUpdateConfigRequest struct{}

type APIProjectSearchRequest struct{}
type APIProjectSearchResponse struct{}

type APIModelHub struct {
	// can we log in to the hub?
	IsLoggedIn bool
	// have all the projects been sucked in?
	HasLoadedAllProjects bool
	ProjectCount         int
	// is circuit breaker enabled?
	IsCircuitBreakerEnabled bool
	//
	Projects map[string]string
}

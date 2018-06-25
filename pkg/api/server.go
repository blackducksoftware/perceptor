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

package api

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

	// for receiving data from perceiver
	http.HandleFunc("/pod", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Errorf("unable to read body for pod POST: %s", err.Error())
				responder.Error(w, r, err, 400)
				return
			}
			var pod Pod
			err = json.Unmarshal(body, &pod)
			if err != nil {
				log.Errorf("unable to ummarshal JSON for pod POST: %s", err.Error())
				responder.Error(w, r, err, 400)
				return
			}
			responder.AddPod(pod)
			fmt.Fprint(w, "")
		case "PUT":
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			var pod Pod
			err = json.Unmarshal(body, &pod)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			responder.UpdatePod(pod)
			fmt.Fprint(w, "")
		case "DELETE":
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			responder.DeletePod(string(body))
			fmt.Fprint(w, "")
		default:
			responder.NotFound(w, r)
		}
	})
	http.HandleFunc("/allpods", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			var allPods AllPods
			err = json.Unmarshal(body, &allPods)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			responder.UpdateAllPods(allPods)
		} else {
			responder.NotFound(w, r)
		}
	})
	http.HandleFunc("/allimages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			var allImages AllImages
			err = json.Unmarshal(body, &allImages)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			responder.UpdateAllImages(allImages)
		} else {
			responder.NotFound(w, r)
		}
	})
	http.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			var image Image
			err = json.Unmarshal(body, &image)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			responder.AddImage(image)
		} else {
			responder.NotFound(w, r)
		}
	})

	// for providing data to perceiver
	http.HandleFunc("/scanresults", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			scanResults := responder.GetScanResults()
			jsonBytes, err := json.MarshalIndent(scanResults, "", "  ")
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

	// for handling messages
	http.HandleFunc("/command", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			var commands PostCommand
			err = json.Unmarshal(body, &commands)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			responder.PostCommand(&commands)
		} else {
			responder.NotFound(w, r)
		}
	})

	// for providing data to scanners
	http.HandleFunc("/nextimage", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			nextImage := responder.GetNextImage()
			jsonBytes, err := json.MarshalIndent(nextImage, "", "  ")
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

	http.HandleFunc("/finishedscan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			var scanResults FinishedScanClientJob
			err = json.Unmarshal(body, &scanResults)
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

	// internal use
	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			var config PostConfig
			err = json.Unmarshal(body, &config)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			responder.PostConfig(&config)
			fmt.Fprint(w, "")
		} else {
			responder.NotFound(w, r)
		}
	})
}

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

// Perceptor API.
//
// Perceptor core REST API
//
// Terms Of Service:
//
// https://www.blackducksoftware.com/
//
// Host: perceptor
// BasePath: /perceptor
// Version: 1.0.0
// License: MIT https://opensource.org/licenses/MIT
// Contact: Black Duck Software<blackduck@blackducksoftware.com>
//
// Consumes:
// - application/json
//
// Produces:
// - application/json
//
// swagger:meta
package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func SetupHTTPServer(responder Responder) {
	// state of the program
	http.Handle("/metrics", prometheus.Handler())
	http.HandleFunc("/model", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			fmt.Fprint(w, responder.GetModel())
		} else {
			responder.NotFound(w, r)
		}
	})

	// for receiving data from perceiver
	http.HandleFunc("/pod", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		// swagger:operation POST /pod perceiver addPod
		//
		// Add a new pod
		//
		// ---
		// parameters:
		// - name: body
		//   in: body
		//   description: New pod object
		//   required: true
		//   schema:
		//     "$ref": "#/definitions/Pod"
		//
		// responses:
		//   '200':
		//     description: success
		//   '400':
		//     description: request problem
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
				log.Infof("unable to ummarshal JSON for pod POST: %s", err.Error())
				responder.Error(w, r, err, 400)
				return
			}
			responder.AddPod(pod)
			fmt.Fprint(w, "")

			// swagger:operation PUT /pod perceiver updatePod
			//
			// Update an existing pod or add if neccessary
			//
			// ---
			// parameters:
			// - name: body
			//   in: body
			//   description: Pod object
			//   required: true
			//   schema:
			//     "$ref": "#/definitions/Pod"
			//
			// responses:
			//   '200':
			//     description: success
			//   '400':
			//     description: request problem
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

		// swagger:operation DELETE /pod/{podName} perceiver deletePod
		//
		// Delete a pod
		//
		// ---
		// parameters:
		// - name: "podName"
		//   in: path
		//   description: Qualified name of the pod, in the format namespace/name
		//   required: true
		//   type: string
		//
		// responses:
		//   '200':
		//     description: success
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
		// swagger:operation PUT /allPods perceiver allPods
		//
		// Updates all pods
		//
		// ---
		// parameters:
		// - name: body
		//   in: body
		//   description: AllPods object
		//   required: true
		//   schema:
		//     "$ref": "#/definitions/AllPods"
		//
		// responses:
		//   '200':
		//     description: success
		//   '400':
		//     description: request problem
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
		// swagger:operation PUT /allImages perceiver allImages
		//
		// Update all images
		//
		// ---
		// parameters:
		// - name: body
		//   in: body
		//   description: AllImages object
		//   required: true
		//   schema:
		//     "$ref": "#/definitions/AllImages"
		//
		// responses:
		//   '200':
		//     description: success
		//   '400':
		//     description: request problem
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
		// swagger:operation POST /image perceiver addImage
		//
		// Add a new image
		//
		// ---
		// parameters:
		// - name: body
		//   in: body
		//   description: New image object
		//   required: true
		//   schema:
		//     "$ref": "#/definitions/Image"
		//
		// responses:
		//   '200':
		//     description: success
		//   '400':
		//     description: request problem
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
		// swagger:operation GET /scanresults perceiver getScanResults
		//
		// Get scan results for all pods and images
		//
		// ---
		// responses:
		//   '200':
		//     description: success
		if r.Method == "GET" {
			scanResults := responder.GetScanResults()
			jsonBytes, err := json.Marshal(scanResults)
			if err != nil {
				responder.Error(w, r, err, 500)
				return
			}
			fmt.Fprint(w, string(jsonBytes))
		} else {
			responder.NotFound(w, r)
		}
	})

	// for providing data to scanners
	http.HandleFunc("/nextimage", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			nextImage := responder.GetNextImage()
			jsonBytes, err := json.Marshal(nextImage)
			if err != nil {
				responder.Error(w, r, err, 500)
			} else {
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
			responder.PostFinishScan(scanResults)
			fmt.Fprint(w, "")
		} else {
			responder.NotFound(w, r)
		}
	})

	// internal use
	http.HandleFunc("/concurrentscanlimit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				responder.Error(w, r, err, 400)
				return
			}
			var limit SetConcurrentScanLimit
			err = json.Unmarshal(body, &limit)
			responder.SetConcurrentScanLimit(limit)
			fmt.Fprint(w, "")
		} else {
			responder.NotFound(w, r)
		}
	})
}

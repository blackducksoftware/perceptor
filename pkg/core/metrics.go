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

package core

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var statusGauge *prometheus.GaugeVec
var handledHTTPRequest *prometheus.CounterVec
var reducerActivityCounter *prometheus.CounterVec
var reducerMessageCounter *prometheus.CounterVec

// prometheus' terminology is so confusing ... a histogram isn't a histogram.  sometimes.
var statusHistogram *prometheus.GaugeVec

// modelMetrics is called periodically -- but NOT every time the model
// is updated -- in case generating the metrics is computationally expensive.
// And the metrics don't need to be updated all that often, since they'll only
// get scraped every now and then by prometheus.
func recordModelMetrics(modelMetrics *ModelMetrics) {
	// log.Info("generating status metrics")

	keys := []ScanStatus{
		ScanStatusUnknown,
		ScanStatusInHubCheckQueue,
		ScanStatusCheckingHub,
		ScanStatusInQueue,
		ScanStatusRunningScanClient,
		ScanStatusRunningHubScan,
		ScanStatusComplete,
		ScanStatusError}
	for _, key := range keys {
		val := modelMetrics.ScanStatusCounts[key]
		status := fmt.Sprintf("image_status_%s", key.String())
		statusGauge.With(prometheus.Labels{"name": status}).Set(float64(val))
	}

	statusGauge.With(prometheus.Labels{"name": "number_of_pods"}).Set(float64(modelMetrics.NumberOfPods))
	statusGauge.With(prometheus.Labels{"name": "number_of_images"}).Set(float64(modelMetrics.NumberOfImages))

	// number of containers per pod (as a histgram, but not a prometheus histogram ???)
	for numberOfContainers, numberOfPods := range modelMetrics.ContainerCounts {
		strCount := fmt.Sprintf("%d", numberOfContainers)
		statusHistogram.With(prometheus.Labels{"name": "containers_per_pod", "count": strCount}).Set(float64(numberOfPods))
	}

	// number of times each image is referenced from a pod's container
	for numberOfReferences, occurences := range modelMetrics.ImageCountHistogram {
		strCount := fmt.Sprintf("%d", numberOfReferences)
		statusHistogram.With(prometheus.Labels{"name": "references_per_image", "count": strCount}).Set(float64(occurences))
	}

	// TODO
	// number of images without a pod pointing to them
}

// successful http requests received

func recordAddPod() {
	handledHTTPRequest.With(prometheus.Labels{"path": "pod", "method": "POST", "code": "200"}).Inc()
}

func recordUpdatePod() {
	handledHTTPRequest.With(prometheus.Labels{"path": "pod", "method": "PUT", "code": "200"}).Inc()
}

func recordDeletePod() {
	handledHTTPRequest.With(prometheus.Labels{"path": "pod", "method": "DELETE", "code": "200"}).Inc()
}

func recordAddImage() {
	handledHTTPRequest.With(prometheus.Labels{"path": "image", "method": "POST", "code": "200"}).Inc()
}

func recordAllPods() {
	handledHTTPRequest.With(prometheus.Labels{"path": "allpods", "method": "PUT", "code": "200"}).Inc()
}

func recordAllImages() {
	handledHTTPRequest.With(prometheus.Labels{"path": "allimages", "method": "PUT", "code": "200"}).Inc()
}

func recordGetNextImage() {
	handledHTTPRequest.With(prometheus.Labels{"path": "nextimage", "method": "POST", "code": "200"}).Inc()
}

func recordPostFinishedScan() {
	handledHTTPRequest.With(prometheus.Labels{"path": "finishedscan", "method": "POST", "code": "200"}).Inc()
}

func recordGetScanResults() {
	handledHTTPRequest.With(prometheus.Labels{"path": "scanresults", "method": "GET", "code": "200"}).Inc()
}

// unsuccessful http requests received

func recordHTTPNotFound(request *http.Request) {
	path := request.URL.Path
	method := request.Method
	handledHTTPRequest.With(prometheus.Labels{"path": path, "method": method, "code": "404"}).Inc()
}

func recordHTTPError(request *http.Request, err error, statusCode int) {
	path := request.URL.Path
	method := request.Method
	statusCodeString := fmt.Sprintf("%d", statusCode)
	handledHTTPRequest.With(prometheus.Labels{"path": path, "method": method, "code": statusCodeString}).Inc()
}

// reducer loop

func recordReducerActivity(isActive bool, duration time.Duration) {
	state := "idle"
	if isActive {
		state = "active"
	}
	reducerActivityCounter.With(prometheus.Labels{"state": state}).Add(duration.Seconds())
}

func recordNumberOfMessagesInQueue(messageCount int) {
	statusGauge.With(prometheus.Labels{"name": "number_of_messages_in_reducer_queue"}).Set(float64(messageCount))
}

func recordMessageType(message string) {
	reducerMessageCounter.With(prometheus.Labels{"message": message}).Inc()
}

// http requests issued

// results from checking hub for completed projects (errors, unexpected things, etc.)

// TODO

func init() {
	statusGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "status_gauge",
		Help:      "a gauge of statuses for perceptor core's current state",
	}, []string{"name"})

	statusHistogram = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "status_histogram",
		Help:      "a histogram of statuses for perceptor core's current state",
	}, []string{"name", "count"})

	handledHTTPRequest = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Subsystem:   "core",
		Name:        "http_handled_status_codes",
		Help:        "status codes for HTTP requests handled by perceptor core",
		ConstLabels: map[string]string{},
	}, []string{"path", "method", "code"})

	reducerActivityCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "reducer_activity",
		Help:      "activity of the reducer -- how much time it's been idle and active, in seconds",
	}, []string{"state"})

	reducerMessageCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "reducer_message",
		Help:      "count of the message types processed by the reducer",
	}, []string{"message"})

	prometheus.MustRegister(handledHTTPRequest)
	prometheus.MustRegister(statusGauge)
	prometheus.MustRegister(statusHistogram)
	prometheus.MustRegister(reducerActivityCounter)
	prometheus.MustRegister(reducerMessageCounter)
}

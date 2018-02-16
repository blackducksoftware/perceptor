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
	"os"

	"github.com/prometheus/client_golang/prometheus"
)

var metricsHandler *Metrics

func init() {
	metricsHandler = newMetrics()
}

type Metrics struct {
	httpHandler http.Handler

	handledHTTPRequest *prometheus.CounterVec
	statusGauge        *prometheus.GaugeVec
	// prometheus' terminology is so confusing ... a histogram isn't a histogram.  sometimes.
	statusHistogram *prometheus.GaugeVec
}

func newMetrics() *Metrics {
	m := Metrics{}
	m.setup()
	m.httpHandler = prometheus.Handler()
	return &m
}

// successful http requests received

func (m *Metrics) addPod(pod Pod) {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "pod", "method": "POST", "code": "200"}).Inc()
}

func (m *Metrics) updatePod(pod Pod) {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "pod", "method": "PUT", "code": "200"}).Inc()
}

func (m *Metrics) deletePod(podName string) {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "pod", "method": "DELETE", "code": "200"}).Inc()
}

func (m *Metrics) addImage(image Image) {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "image", "method": "POST", "code": "200"}).Inc()
}

func (m *Metrics) allPods(pods []Pod) {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "allpods", "method": "PUT", "code": "200"}).Inc()
}

func (m *Metrics) allImages(images []Image) {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "allimages", "method": "PUT", "code": "200"}).Inc()
}

func (m *Metrics) getNextImage() {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "nextimage", "method": "POST", "code": "200"}).Inc()
}

func (m *Metrics) postFinishedScan() {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "finishedscan", "method": "POST", "code": "200"}).Inc()
}

func (m *Metrics) getScanResults() {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "scanresults", "method": "GET", "code": "200"}).Inc()
}

// unsuccessful http requests received

func (m *Metrics) httpNotFound(request *http.Request) {
	path := request.URL.Path
	method := request.Method
	m.handledHTTPRequest.With(prometheus.Labels{"path": path, "method": method, "code": "404"}).Inc()
}

func (m *Metrics) httpError(request *http.Request, err error) {
	path := request.URL.Path
	method := request.Method
	m.handledHTTPRequest.With(prometheus.Labels{"path": path, "method": method, "code": "500"}).Inc()
}

// model

// modelMetrics is called periodically -- but NOT every time the model
// is updated -- in case generating the metrics is computationally expensive.
// And the metrics don't need to be updated all that often, since they'll only
// get scraped every now and then by prometheus.
func (m *Metrics) modelMetrics(modelMetrics *ModelMetrics) {
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
		m.statusGauge.With(prometheus.Labels{"name": status}).Set(float64(val))
	}

	m.statusGauge.With(prometheus.Labels{"name": "number_of_pods"}).Set(float64(modelMetrics.NumberOfPods))
	m.statusGauge.With(prometheus.Labels{"name": "number_of_images"}).Set(float64(modelMetrics.NumberOfImages))

	// number of containers per pod (as a histgram, but not a prometheus histogram ???)
	for numberOfContainers, numberOfPods := range modelMetrics.ContainerCounts {
		strCount := fmt.Sprintf("%d", numberOfContainers)
		m.statusHistogram.With(prometheus.Labels{"name": "containers_per_pod", "count": strCount}).Set(float64(numberOfPods))
	}

	// number of times each image is referenced from a pod's container
	for numberOfReferences, occurences := range modelMetrics.ImageCountHistogram {
		strCount := fmt.Sprintf("%d", numberOfReferences)
		m.statusHistogram.With(prometheus.Labels{"name": "references_per_image", "count": strCount}).Set(float64(occurences))
	}

	// TODO
	// number of images without a pod pointing to them
}

// http requests issued

// results from checking hub for completed projects (errors, unexpected things, etc.)

// TODO

// setup

func (m *Metrics) setup() {
	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())

	m.handledHTTPRequest = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Subsystem:   "core",
		Name:        "http_handled_status_codes",
		Help:        "status codes for HTTP requests handled by perceptor core",
		ConstLabels: map[string]string{},
	}, []string{"path", "method", "code"})

	m.statusGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "status_gauge",
		Help:      "a gauge of statuses for perceptor core's current state",
	}, []string{"name"})

	m.statusHistogram = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "status_histogram",
		Help:      "a histogram of statuses for perceptor core's current state",
	}, []string{"name", "count"})

	prometheus.MustRegister(m.handledHTTPRequest)
	prometheus.MustRegister(m.statusGauge)
	prometheus.MustRegister(m.statusHistogram)
}

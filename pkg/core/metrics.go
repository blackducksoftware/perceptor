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

	model "github.com/blackducksoftware/perceptor/pkg/core/model"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	statusLabel           = "status"
	vulnerabilitiesLabel  = "vulnerability_count"
	policyViolationsLabel = "policy_violation_count"
)

var handledHTTPRequest *prometheus.CounterVec
var statusGauge *prometheus.GaugeVec

var podStatusGauge *prometheus.GaugeVec
var podPolicyViolationsGauge *prometheus.GaugeVec
var podVulnerabilitiesGauge *prometheus.GaugeVec

var imageStatusGauge *prometheus.GaugeVec
var imagePolicyViolationsGauge *prometheus.GaugeVec
var imageVulnerabilitiesGauge *prometheus.GaugeVec

var eventCounter *prometheus.CounterVec

// prometheus' terminology is so confusing ... a histogram isn't a histogram.  sometimes.
var statusHistogram *prometheus.GaugeVec

func recordEvent(subsystem string, name string) {
	eventCounter.With(prometheus.Labels{"subsystem": subsystem, "name": name}).Inc()
}

func recordModelMetrics(modelMetrics *model.Metrics) {
	keys := []model.ScanStatus{
		model.ScanStatusUnknown,
		model.ScanStatusInQueue,
		model.ScanStatusRunningScanClient,
		model.ScanStatusRunningHubScan,
		model.ScanStatusComplete}
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

	for podStatus, count := range modelMetrics.PodStatus {
		podStatusGauge.With(prometheus.Labels{statusLabel: podStatus}).Set(float64(count))
	}
	for podVulnerabilities, count := range modelMetrics.PodVulnerabilities {
		value := fmt.Sprintf("%d", podVulnerabilities)
		podVulnerabilitiesGauge.With(prometheus.Labels{vulnerabilitiesLabel: value}).Set(float64(count))
	}
	for podPolicyViolations, count := range modelMetrics.PodPolicyViolations {
		value := fmt.Sprintf("%d", podPolicyViolations)
		podPolicyViolationsGauge.With(prometheus.Labels{policyViolationsLabel: value}).Set(float64(count))
	}

	for imageStatus, count := range modelMetrics.ImageStatus {
		imageStatusGauge.With(prometheus.Labels{statusLabel: imageStatus}).Set(float64(count))
	}
	for imageVulnerabilities, count := range modelMetrics.ImageVulnerabilities {
		value := fmt.Sprintf("%d", imageVulnerabilities)
		imageVulnerabilitiesGauge.With(prometheus.Labels{vulnerabilitiesLabel: value}).Set(float64(count))
	}
	for imagePolicyViolations, count := range modelMetrics.ImagePolicyViolations {
		value := fmt.Sprintf("%d", imagePolicyViolations)
		imagePolicyViolationsGauge.With(prometheus.Labels{policyViolationsLabel: value}).Set(float64(count))
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

func init() {
	statusHistogram = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "status_histogram",
		Help:      "a histogram of statuses for perceptor core's current state",
	}, []string{"name", "count"})
	prometheus.MustRegister(statusHistogram)

	handledHTTPRequest = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Subsystem:   "core",
		Name:        "http_handled_status_codes",
		Help:        "status codes for HTTP requests handled by perceptor core",
		ConstLabels: map[string]string{},
	}, []string{"path", "method", "code"})
	prometheus.MustRegister(handledHTTPRequest)

	podStatusGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "pod_status",
		Help:      "buckets of pod status ('Unknown' means not yet scanned)",
	}, []string{statusLabel})
	prometheus.MustRegister(podStatusGauge)

	podVulnerabilitiesGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "pod_vulnerabilities",
		Help:      "buckets of pod vulnerability counts (-1 means not yet scanned)",
	}, []string{vulnerabilitiesLabel})
	prometheus.MustRegister(podVulnerabilitiesGauge)

	podPolicyViolationsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "pod_policy_violations",
		Help:      "buckets of pod policy violation counts (-1 means not yet scanned)",
	}, []string{policyViolationsLabel})
	prometheus.MustRegister(podPolicyViolationsGauge)

	imageStatusGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "image_status",
		Help:      "buckets of image status ('Unknown' means not yet scanned)",
	}, []string{statusLabel})
	prometheus.MustRegister(imageStatusGauge)

	imageVulnerabilitiesGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "image_vulnerabilities",
		Help:      "buckets of image vulnerability counts (-1 means not yet scanned)",
	}, []string{vulnerabilitiesLabel})
	prometheus.MustRegister(imageVulnerabilitiesGauge)

	statusGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "status_gauge",
		Help:      "a gauge of statuses for perceptor core's current state",
	}, []string{"name"})
	prometheus.MustRegister(statusGauge)

	imagePolicyViolationsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "image_policy_violations",
		Help:      "buckets of image policy violation counts (-1 means not yet scanned)",
	}, []string{policyViolationsLabel})
	prometheus.MustRegister(imagePolicyViolationsGauge)

	eventCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "event_counter",
		Help:      "various events happening in perceptor core",
	}, []string{"subsystem", "name"})
	prometheus.MustRegister(eventCounter)
}

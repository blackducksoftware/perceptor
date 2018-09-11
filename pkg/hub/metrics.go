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

package hub

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var hubResponse *prometheus.CounterVec
var hubData *prometheus.CounterVec
var hubResponseTime *prometheus.HistogramVec
var circuitBreakerState *prometheus.GaugeVec
var hubRequestIsCircuitBreakerEnabled *prometheus.CounterVec
var circuitBreakerTransitions *prometheus.CounterVec
var scanStageGauge *prometheus.GaugeVec
var eventCounter *prometheus.CounterVec

func recordHubResponse(host string, name string, isSuccessful bool) {
	isSuccessString := fmt.Sprintf("%t", isSuccessful)
	hubResponse.With(prometheus.Labels{"host": host, "name": name, "isSuccess": isSuccessString}).Inc()
}

func recordHubData(host string, name string, isOkay bool) {
	isOkayString := fmt.Sprintf("%t", isOkay)
	hubData.With(prometheus.Labels{"host": host, "name": name, "okay": isOkayString}).Inc()
}

func recordHubResponseTime(host string, name string, duration time.Duration) {
	milliseconds := float64(duration / time.Millisecond)
	hubResponseTime.With(prometheus.Labels{"host": host, "name": name}).Observe(milliseconds)
}

func recordCircuitBreakerState(host string, state CircuitBreakerState) {
	circuitBreakerState.With(prometheus.Labels{"host": host}).Set(float64(state))
}

func recordCircuitBreakerIsEnabled(host string, isEnabled bool) {
	isEnabledString := fmt.Sprintf("%t", isEnabled)
	hubRequestIsCircuitBreakerEnabled.With(prometheus.Labels{"host": host, "isEnabled": isEnabledString}).Inc()
}

func recordCircuitBreakerTransition(host string, from CircuitBreakerState, to CircuitBreakerState) {
	circuitBreakerTransitions.With(prometheus.Labels{"host": host, "from": from.String(), "to": to.String()}).Inc()
}

func recordClientState(host string, metrics *clientStateMetrics) {
	keys := []ScanStage{
		ScanStageUnknown,
		ScanStageScanClient,
		ScanStageHubScan,
		ScanStageComplete,
		ScanStageFailure,
	}
	for _, key := range keys {
		val := metrics.scanStageCounts[key]
		status := fmt.Sprintf("scan_stage_%s", key.String())
		scanStageGauge.With(prometheus.Labels{"host": host, "name": status}).Set(float64(val))
	}
	// TODO errors?
}

func recordEvent(host string, event string) {
	eventCounter.With(prometheus.Labels{"host": host, "event": event}).Inc()
}

func init() {
	hubResponse = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Subsystem:   "core",
		Name:        "http_hub_requests",
		Help:        "names and status codes for HTTP requests issued by perceptor to the hub",
		ConstLabels: map[string]string{},
	}, []string{"host", "name", "isSuccess"})
	prometheus.MustRegister(hubResponse)

	hubData = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Subsystem:   "core",
		Name:        "hub_data_integrity",
		Help:        "tracks hub data integrity: whether data fetched from the hub meets Perceptor's expectations",
		ConstLabels: map[string]string{},
	}, []string{"host", "name", "okay"})
	prometheus.MustRegister(hubData)

	hubResponseTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "hub_response_time",
		Help:      "tracks the response times of Hub requests in milliseconds",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 20),
	}, []string{"host", "name"})
	prometheus.MustRegister(hubResponseTime)

	circuitBreakerState = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "hub_circuit_breaker_state",
		Help:      "tracks the state of the circuit breaker; 0 = disabled; 1 = enabled; 2 = checking;",
	}, []string{"host"})
	prometheus.MustRegister(circuitBreakerState)

	hubRequestIsCircuitBreakerEnabled = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Subsystem:   "core",
		Name:        "hub_request_is_circuit_breaker_enabled",
		Help:        "tracks whether the circuit breaker is enabled or disabled when a Hub http request is issued",
		ConstLabels: map[string]string{},
	}, []string{"host", "isEnabled"})
	prometheus.MustRegister(hubRequestIsCircuitBreakerEnabled)

	circuitBreakerTransitions = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Subsystem:   "core",
		Name:        "hub_circuit_breaker_transitions",
		Help:        "tracks circuit breaker state transitions",
		ConstLabels: map[string]string{},
	}, []string{"host", "from", "to"})
	prometheus.MustRegister(circuitBreakerTransitions)

	scanStageGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "hub_scan_stage_gauge",
		Help:      "a gauge of scan stages for the hub client",
	}, []string{"host", "name"})
	prometheus.MustRegister(scanStageGauge)

	eventCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "hub_event_counter",
		Help:      "a counter of interesting events happening to each client",
	}, []string{"host", "event"})
	prometheus.MustRegister(eventCounter)
}

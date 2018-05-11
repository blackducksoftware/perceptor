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

package actions

import (
	"github.com/prometheus/client_golang/prometheus"
)

var requeueStalledScanCounter *prometheus.CounterVec
var errorCounter *prometheus.CounterVec

func recordRequeueStalledScan(imageState string) {
	requeueStalledScanCounter.With(prometheus.Labels{"imageState": imageState}).Inc()
}

func recordError(action string, name string) {
	errorCounter.With(prometheus.Labels{"action": action, "name": name}).Inc()
}

func init() {
	requeueStalledScanCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "requeue_stalled_scan_counter",
		Help:      "records when image scans are timed-out and requeued",
	}, []string{"imageState"})
	prometheus.MustRegister(requeueStalledScanCounter)

	errorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "action_errors_counter",
		Help:      "records errors encounted during core action processing",
	}, []string{"action", "name"})
	prometheus.MustRegister(errorCounter)
}

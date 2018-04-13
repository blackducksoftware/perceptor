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

package model

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

var eventsCounter *prometheus.CounterVec
var stateTransitionCounter *prometheus.CounterVec

func recordStateTransition(from ScanStatus, to ScanStatus, isLegal bool) {
	stateTransitionCounter.With(prometheus.Labels{
		"from":  from.String(),
		"to":    to.String(),
		"legal": fmt.Sprintf("%t", isLegal)}).Inc()
}

func recordEvent(event string) {
	eventsCounter.With(prometheus.Labels{"event": event}).Inc()
}

func init() {
	stateTransitionCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Subsystem:   "core",
		Name:        "model_image_state_transitions",
		Help:        "state transitions for images in the perceptor model",
		ConstLabels: map[string]string{},
	}, []string{"from", "to", "legal"})
	prometheus.MustRegister(stateTransitionCounter)

	eventsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "events",
		Help:      "counters for events happening in the core",
	}, []string{"event"})
	prometheus.MustRegister(eventsCounter)
}

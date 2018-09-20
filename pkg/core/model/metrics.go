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
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var eventsCounter *prometheus.CounterVec
var stateTransitionCounter *prometheus.CounterVec
var actionErrorCounter *prometheus.CounterVec

// var errorCounter *prometheus.CounterVec

var statusGauge *prometheus.GaugeVec
var reducerActivityCounter *prometheus.CounterVec
var reducerMessageCounter *prometheus.CounterVec
var setImagePriorityCounter *prometheus.CounterVec

func recordActionError(action string) {
	actionErrorCounter.With(prometheus.Labels{"action": action}).Inc()
}

// TODO
// func recordError(action string, name string) {
// 	errorCounter.With(prometheus.Labels{"action": action, "name": name}).Inc()
// }

func recordSetImagePriority(from int, to int) {
	setImagePriorityCounter.With(prometheus.Labels{
		"from": fmt.Sprintf("%d", from),
		"to":   fmt.Sprintf("%d", to)}).Inc()
}

func recordStateTransition(from ScanStatus, to ScanStatus, isLegal bool) {
	stateTransitionCounter.With(prometheus.Labels{
		"from":  from.String(),
		"to":    to.String(),
		"legal": fmt.Sprintf("%t", isLegal)}).Inc()
}

func recordEvent(event string) {
	eventsCounter.With(prometheus.Labels{"event": event}).Inc()
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

	// errorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	// 	Namespace: "perceptor",
	// 	Subsystem: "core",
	// 	Name:      "model_errors_counter",
	// 	Help:      "records errors encounted in the model package",
	// }, []string{"action", "name"})
	// prometheus.MustRegister(errorCounter)

	actionErrorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "action_errors_counter",
		Help:      "records errors encounted during model action processing",
	}, []string{"action"})
	prometheus.MustRegister(actionErrorCounter)

	setImagePriorityCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "set_image_priority",
		Help:      "records image priority changes",
	}, []string{"from", "to"})
	prometheus.MustRegister(setImagePriorityCounter)

	statusGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "model_status_gauge",
		Help:      "a gauge of statuses for perceptor core model's current state",
	}, []string{"name"})
	prometheus.MustRegister(statusGauge)

	reducerActivityCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "reducer_activity",
		Help:      "activity of the reducer -- how much time it's been idle and active, in seconds",
	}, []string{"state"})
	prometheus.MustRegister(reducerActivityCounter)

	reducerMessageCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "reducer_message",
		Help:      "count of the message types processed by the reducer",
	}, []string{"message"})
	prometheus.MustRegister(reducerMessageCounter)
}

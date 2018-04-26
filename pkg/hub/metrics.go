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

func recordHubResponse(name string, isSuccessful bool) {
	isSuccessString := fmt.Sprintf("%t", isSuccessful)
	hubResponse.With(prometheus.Labels{"name": name, "isSuccess": isSuccessString}).Inc()
}

func recordHubData(name string, isOkay bool) {
	isOkayString := fmt.Sprintf("%t", isOkay)
	hubData.With(prometheus.Labels{"name": name, "okay": isOkayString}).Inc()
}

func recordHubResponseTime(name string, duration time.Duration) {
	milliseconds := float64(duration / time.Millisecond)
	hubResponseTime.With(prometheus.Labels{"name": name}).Observe(milliseconds)
}

func init() {
	hubResponse = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Subsystem:   "core",
		Name:        "http_hub_requests",
		Help:        "names and status codes for HTTP requests issued by perceptor to the hub",
		ConstLabels: map[string]string{},
	}, []string{"name", "isSuccess"})
	prometheus.MustRegister(hubResponse)

	hubData = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Subsystem:   "core",
		Name:        "hub_data_integrity",
		Help:        "tracks hub data integrity: whether data fetched from the hub meets Perceptor's expectations",
		ConstLabels: map[string]string{},
	}, []string{"name", "okay"})
	prometheus.MustRegister(hubData)

	hubResponseTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "hub_response_time",
		Help:      "tracks the response times of Hub requests in milliseconds",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 20),
	}, []string{"name"})
	prometheus.MustRegister(hubResponseTime)
}

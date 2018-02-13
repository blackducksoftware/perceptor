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

package laserbeak

// Lazerbeak is a logging library that intercepts everyhing
// and reports it upstream.

import (
	"bytes"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type MetricType string

const (
	LZ_PERCEIVER_IN            = MetricType("log_perceiver_provided_input")
	LZ_HUB_OUT                 = MetricType("log_hub_finished_something")
	LZ_HUB_POLLING             = MetricType("log_hub_waiting_on_hub")
	LZ_PERCEPTOR_SCAN_PROGRESS = MetricType("log_perceptor_scan_progress")
	// should be none of these.
	LZ_CATCH_ALL = MetricType("log_perceptor_uncategorized")

	//... To be continued ...
)

var cv *prometheus.CounterVec

func init() {
	formatter := &log.TextFormatter{
		FullTimestamp:   false,
		TimestampFormat: "15:04",
	}
	log.SetFormatter(formatter)

	cv = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "perceptor",
			Subsystem: "lazerlog",
			Name:      "log",
			Help:      "Counter",
		},
		[]string{"log_type"})
}

func concat(a string, b string) string {
	var buffer bytes.Buffer
	buffer.WriteString("[ ")
	buffer.WriteString(a)
	buffer.WriteString(" ]")
	buffer.WriteString(" ~ ")
	buffer.WriteString(b)
	return buffer.String()
}

func ZLogInfo(base MetricType, message string) {
	cv.WithLabelValues(string(base)).Inc()
	log.Infof(concat(string(base), message))
}

func ZLogInfof(base MetricType, message string, args interface{}) {
	cv.WithLabelValues(string(base)).Inc()
	log.Infof(concat(string(base), message), args)
}

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

package federation

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

var handledHTTPRequest *prometheus.CounterVec

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
	handledHTTPRequest = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Subsystem:   "core",
		Name:        "http_handled_status_codes",
		Help:        "status codes for HTTP requests handled by perceptor core",
		ConstLabels: map[string]string{},
	}, []string{"path", "method", "code"})
	prometheus.MustRegister(handledHTTPRequest)
}

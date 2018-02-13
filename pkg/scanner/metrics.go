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

package scanner

import (
	"fmt"
	"net/http"
	"os"

	"github.com/blackducksoftware/perceptor/pkg/docker"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// ScannerMetricsHandler handles http requests to get prometheus metrics
// for image scanning
func ScannerMetricsHandler(hostName string, imageScanStats <-chan ScanClientJobResults, httpStats <-chan HttpResult) http.Handler {
	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())

	tarballSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "perceptor",
			Subsystem:   "scanner",
			Name:        "tarballsize",
			Help:        "tarball file size in MBs",
			Buckets:     prometheus.ExponentialBuckets(1, 2, 15),
			ConstLabels: map[string]string{"hostName": hostName},
		},
		[]string{"tarballSize"})

	durations := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "perceptor",
			Subsystem:   "scanner",
			Name:        "timings",
			Help:        "time durations of scanner operations",
			Buckets:     prometheus.ExponentialBuckets(0.25, 2, 20),
			ConstLabels: map[string]string{"hostName": hostName},
		},
		[]string{"stage"})

	errors := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Subsystem:   "scanner",
		Name:        "scannerErrors",
		Help:        "error codes from image pulling and scanning",
		ConstLabels: map[string]string{"hostName": hostName},
	}, []string{"stage", "errorName"})

	httpResults := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Subsystem:   "scanner",
		Name:        "http_response_status_codes",
		Help:        "status codes for responses from HTTP requests issued by scanner",
		ConstLabels: map[string]string{"hostName": hostName},
	},
		[]string{"request", "code"})

	go func() {
		for {
			select {
			case stats := <-imageScanStats:
				log.Infof("got new image scan stats: %+v", stats)
				// durations
				if stats.ScanClientDuration != nil {
					durations.With(prometheus.Labels{"stage": "scan client"}).Observe(stats.ScanClientDuration.Seconds())
				}
				if stats.TotalDuration != nil {
					durations.With(prometheus.Labels{"stage": "scan total"}).Observe(stats.TotalDuration.Seconds())
				}
				if stats.DockerStats.CreateDuration != nil {
					durations.With(prometheus.Labels{"stage": "docker create"}).Observe(stats.DockerStats.CreateDuration.Seconds())
				}
				if stats.DockerStats.SaveDuration != nil {
					durations.With(prometheus.Labels{"stage": "docker save"}).Observe(stats.DockerStats.SaveDuration.Seconds())
				}
				if stats.DockerStats.TotalDuration != nil {
					durations.With(prometheus.Labels{"stage": "docker get image total"}).Observe(stats.DockerStats.TotalDuration.Seconds())
				}
				// file size
				if stats.DockerStats.TarFileSizeMBs != nil {
					tarballSize.WithLabelValues("tarballSize").Observe(float64(*stats.DockerStats.TarFileSizeMBs))
				}
				// errors
				err := stats.Err
				if err != nil {
					var stage string
					var errorName string
					switch e := err.RootCause.(type) {
					case docker.ImagePullError:
						stage = "docker pull"
						errorName = e.Code.String()
					default:
						stage = "running scan client"
						errorName = err.Code.String()
					}
					errors.With(prometheus.Labels{"stage": stage, "errorName": errorName}).Inc()
				}
			case httpStats := <-httpStats:
				var request string
				switch httpStats.Path {
				case PathGetNextImage:
					request = "getNextImage"
				case PathPostScanResults:
					request = "finishScan"
				}
				httpResults.With(prometheus.Labels{"request": request, "code": fmt.Sprintf("%d", httpStats.StatusCode)}).Inc()
			}
		}
	}()
	prometheus.MustRegister(tarballSize)
	prometheus.MustRegister(durations)
	prometheus.MustRegister(errors)
	prometheus.MustRegister(httpResults)

	return prometheus.Handler()
}

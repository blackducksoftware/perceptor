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
	"testing"
	"time"

	"github.com/blackducksoftware/perceptor/pkg/docker"
	log "github.com/sirupsen/logrus"
)

func TestMetrics(t *testing.T) {
	scanResults := make(chan ScanClientJobResults)
	httpResults := make(chan HttpResult)
	m := ScannerMetricsHandler("hostName", scanResults, httpResults)
	if m == nil {
		t.Error("expected m to be non-nil")
	}

	duration := time.Duration(4078 * time.Millisecond)
	createDuration := time.Duration(16384 * time.Millisecond)
	saveDuration := time.Duration(32768 * time.Millisecond)
	totalDuration := time.Duration(createDuration.Nanoseconds() + saveDuration.Nanoseconds())
	fileSize := 123423
	scanResults <- ScanClientJobResults{
		DockerStats: docker.ImagePullStats{
			CreateDuration: &createDuration,
			Err:            nil,
			SaveDuration:   &saveDuration,
			TotalDuration:  &totalDuration,
			TarFileSizeMBs: &fileSize,
		},
		Err:                &ScanError{Code: ErrorTypeFailedToRunJavaScanner, RootCause: fmt.Errorf("oops")},
		ScanClientDuration: &duration,
	}

	httpResults <- HttpResult{
		Path:       PathGetNextImage,
		StatusCode: 200,
	}

	message := "finished test case"
	t.Log(message)
	log.Info(message)
}

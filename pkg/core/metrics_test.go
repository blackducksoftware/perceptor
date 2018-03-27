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
	"net/url"
	"testing"

	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	log "github.com/sirupsen/logrus"
)

func TestMetrics(t *testing.T) {
	recordAddPod()
	recordAllPods()
	recordAddImage()
	recordDeletePod()
	recordAllImages()
	recordHTTPError(&http.Request{URL: &url.URL{}}, fmt.Errorf("oops"), 500)
	recordAllImages()
	recordGetNextImage()
	recordHTTPNotFound(&http.Request{URL: &url.URL{}})
	recordModelMetrics(&m.ModelMetrics{
		ContainerCounts:       map[int]int{3: 4},
		ImageCountHistogram:   map[int]int{8: 5},
		ImagePolicyViolations: map[int]int{2: 2},
		ImageStatus:           map[string]int{"abc": 4},
		ImageVulnerabilities:  map[int]int{9: 3},
		NumberOfImages:        4,
		NumberOfPods:          8,
		PodPolicyViolations:   map[int]int{13: 16},
		PodStatus:             map[string]int{"zzz": 8},
		PodVulnerabilities:    map[int]int{9: 1},
		ScanStatusCounts:      map[m.ScanStatus]int{m.ScanStatusComplete: 31},
	})
	recordGetScanResults()
	recordPostFinishedScan()

	message := "finished test case"
	t.Log(message)
	log.Info(message)
}

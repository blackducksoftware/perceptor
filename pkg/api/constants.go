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

package api

// Three things that should work:
// curl -X GET http://perceptor.bds-perceptor.svc.cluster.local:3001/metrics
// curl -X GET http://perceptor.bds-perceptor:3001/metrics
// curl -X GET http://perceptor:3001/metrics
const (
	PerceptorBaseURL = "http://perceptor"
	// perceptor-scanner paths
	NextImagePath    = "nextimage"
	FinishedScanPath = "finishedscan"
	// perceiver paths
	PodPath         = "pod"
	ImagePath       = "image"
	ScanResultsPath = "scanresults"
	AllImagesPath   = "allimages"
	AllPodsPath     = "allpods"
	// Internal
	ConcurrentScanLimitPath = "concurrentscanlimit"
	// ports (basically so that you can run these locally without them stomping on each other -- for testing)
	PerceptorPort        = "3001"
	PerceiverPort        = "3002"
	PerceptorScannerPort = "3003"
)

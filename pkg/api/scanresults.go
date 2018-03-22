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

// swagger:model
type ScanResults struct {
	// The scan client version used in the scan
	// required: true
	HubScanClientVersion string

	// The version of the Hub used for analysis
	// required: true
	HubVersion string

	// Collection of pods scanned
	// required: true
	Pods []ScannedPod

	// Collection of images scanned
	// required: true
	Images []ScannedImage
}

func NewScanResults(hubScanClientVersion string, hubVersion string, pods []ScannedPod, images []ScannedImage) *ScanResults {
	return &ScanResults{
		HubScanClientVersion: hubScanClientVersion,
		HubVersion:           hubVersion,
		Pods:                 pods,
		Images:               images}
}

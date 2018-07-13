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

import "time"

// Timings manages values for regularly scheduled internal tasks
type Timings struct {
	HubClientTimeout time.Duration

	CheckHubForCompletedScansPause time.Duration
	CheckHubThrottle               time.Duration

	CheckForStalledScansPause time.Duration
	StalledScanClientTimeout  time.Duration

	RefreshImagePause time.Duration

	EnqueueLayersForRefreshPause time.Duration
	RefreshThresholdDuration     time.Duration

	ModelMetricsPause time.Duration

	HubReloginPause time.Duration
}

// DefaultTimings supplies reasonable default values for TaskTimingConfig
var DefaultTimings = &Timings{
	HubClientTimeout:               20 * time.Second,
	CheckHubForCompletedScansPause: 20 * time.Second,
	CheckHubThrottle:               1 * time.Second,
	CheckForStalledScansPause:      1 * time.Minute,
	StalledScanClientTimeout:       2 * time.Hour,
	RefreshImagePause:              1 * time.Minute,
	EnqueueLayersForRefreshPause:   5 * time.Minute,
	RefreshThresholdDuration:       30 * time.Minute,
	ModelMetricsPause:              15 * time.Second,
	HubReloginPause:                30 * time.Minute,
}

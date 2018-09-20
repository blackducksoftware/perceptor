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

import (
	"time"
)

// Model ...
type Model struct {
	Hubs      map[string]*ModelHub
	CoreModel *CoreModel
	Config    *ModelConfig
	Scheduler *ModelScanScheduler
}

// ModelScanScheduler ...
type ModelScanScheduler struct {
	ConcurrentScanLimit int
	TotalScanLimit      int
}

// CoreModel .....
type CoreModel struct {
	Pods             map[string]*Pod
	Images           map[string]*ModelImageInfo
	ImageScanQueue   []map[string]interface{}
	ImageTransitions []*ModelImageTransition
}

// ModelImageTransition .....
type ModelImageTransition struct {
	Sha  string
	From string
	To   string
	Err  string
	Time string
}

// ModelHubConfig ...
type ModelHubConfig struct {
	User                string
	PasswordEnvVar      string
	ClientTimeout       ModelTime
	Port                int
	ConcurrentScanLimit int
	TotalScanLimit      int
}

// ModelConfig .....
type ModelConfig struct {
	Timings  *ModelTimings
	Hub      *ModelHubConfig
	Port     int
	LogLevel string
}

// ModelTime ...
type ModelTime struct {
	duration     time.Duration
	Minutes      float64
	Seconds      float64
	Milliseconds float64
}

// NewModelTime consumes a time.Duration and calculates the minutes, seconds,
// and milliseconds
func NewModelTime(duration time.Duration) *ModelTime {
	return &ModelTime{
		duration:     duration,
		Minutes:      float64(duration) / float64(time.Minute),
		Seconds:      float64(duration) / float64(time.Second),
		Milliseconds: float64(duration) / float64(time.Millisecond),
	}
}

// ModelTimings ...
type ModelTimings struct {
	CheckForStalledScansPause ModelTime
	StalledScanClientTimeout  ModelTime
	ModelMetricsPause         ModelTime
	UnknownImagePause         ModelTime
}

// ModelImageInfo .....
type ModelImageInfo struct {
	ScanStatus             string
	TimeOfLastStatusChange string
	ScanResults            interface{}
	ImageSha               string
	RepoTags               []*ModelRepoTag
	Priority               int
}

// ModelRepoTag ...
type ModelRepoTag struct {
	Repository string
	Tag        string
}

// ModelCircuitBreaker ...
type ModelCircuitBreaker struct {
	State               string
	NextCheckTime       *time.Time
	MaxBackoffDuration  ModelTime
	ConsecutiveFailures int
}

// ModelHub describes a hub client model
type ModelHub struct {
	// can we log in to the hub?
	//	IsLoggedIn bool
	// have all the projects been sucked in?
	HasLoadedAllCodeLocations bool
	// map of project name to ... ? hub URL?
	//	Projects map[string]string
	// map of code location name to mapped project version url
	CodeLocations  map[string]*ModelCodeLocation
	Errors         []string
	Status         string
	CircuitBreaker *ModelCircuitBreaker
	Host           string
}

// ModelCodeLocation ...
type ModelCodeLocation struct {
	Stage                string
	Href                 string
	URL                  string
	MappedProjectVersion string
	UpdatedAt            string
	ComponentsHref       string
}

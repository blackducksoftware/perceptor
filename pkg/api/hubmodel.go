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

import "time"

// HubModel describes a hub client model
type HubModel struct {
	// can we log in to the hub?
	IsLoggedIn bool
	// have all the projects been sucked in?
	HasLoadedAllProjects bool
	// is circuit breaker enabled?
	IsCircuitBreakerEnabled bool
	// map of project name to ... ? hub URL?
	Projects map[string]string
	// map of code location name to mapped project version url
	CodeLocations map[string]string
	// bad things that have happened
	Errors []string
	// status
	Status string

	// more stuff
	CircuitBreakerState string
	NextCheckTime       *time.Time
	ConsecutiveFailures int
}

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

package hub

import (
	"fmt"
	"math"
	"time"

	"github.com/blackducksoftware/hub-client-go/hubapi"
	"github.com/blackducksoftware/hub-client-go/hubclient"
)

// CircuitBreaker .....
type CircuitBreaker struct {
	Client              *hubclient.Client
	State               CircuitBreakerState
	NextCheckTime       *time.Time
	ConsecutiveFailures int
}

// NewCircuitBreaker .....
func NewCircuitBreaker(client *hubclient.Client) *CircuitBreaker {
	cb := &CircuitBreaker{
		Client:              client,
		NextCheckTime:       nil,
		ConsecutiveFailures: 0,
	}
	cb.setState(CircuitBreakerStateEnabled)
	return cb
}

func (cb *CircuitBreaker) setState(state CircuitBreakerState) {
	recordCircuitBreakerState(state)
	cb.State = state
}

// IsEnabled .....
func (cb *CircuitBreaker) IsEnabled() bool {
	return cb.State != CircuitBreakerStateDisabled
}

// isAbleToIssueRequest does 2 things:
// 1. changes the state to `Checking` if necessary
// 2. returns whether the circuit breaker is enabled
func (cb *CircuitBreaker) isAbleToIssueRequest() bool {
	if cb.State == CircuitBreakerStateDisabled && time.Now().After(*cb.NextCheckTime) {
		cb.setState(CircuitBreakerStateChecking)
	}
	return cb.IsEnabled()
}

func (cb *CircuitBreaker) failure() {
	switch cb.State {
	case CircuitBreakerStateEnabled:
		cb.setState(CircuitBreakerStateDisabled)
		cb.ConsecutiveFailures = 1
		cb.setNextCheckTime()
	case CircuitBreakerStateDisabled:
		break
	case CircuitBreakerStateChecking:
		cb.setState(CircuitBreakerStateDisabled)
		cb.ConsecutiveFailures++
		cb.setNextCheckTime()
	}
}

func (cb *CircuitBreaker) success() {
	switch cb.State {
	case CircuitBreakerStateEnabled:
		break
	case CircuitBreakerStateDisabled:
		break
	case CircuitBreakerStateChecking:
		cb.setState(CircuitBreakerStateEnabled)
		cb.ConsecutiveFailures = 0
		cb.NextCheckTime = nil
	}
}

func (cb *CircuitBreaker) setNextCheckTime() {
	nextCheckTime := time.Now().Add(time.Duration(math.Pow(2, float64(cb.ConsecutiveFailures))) * time.Second)
	cb.NextCheckTime = &nextCheckTime
}

// ListProjects ...
func (cb *CircuitBreaker) ListProjects(projectName string) (*hubapi.ProjectList, error) {
	if !cb.isAbleToIssueRequest() {
		return nil, fmt.Errorf("unable to fetch project list: circuit breaker disabled")
	}
	queryString := fmt.Sprintf("name:%s", projectName)
	val, err := cb.Client.ListProjects(&hubapi.GetListOptions{Q: &queryString})
	if err == nil {
		cb.success()
	} else {
		cb.failure()
	}
	return val, err
}

// ListProjectVersions ...
func (cb *CircuitBreaker) ListProjectVersions(link hubapi.ResourceLink, versionName string) (*hubapi.ProjectVersionList, error) {
	if !cb.isAbleToIssueRequest() {
		return nil, fmt.Errorf("unable to fetch project version list: circuit breaker disabled")
	}
	q := fmt.Sprintf("versionName:%s", versionName)
	val, err := cb.Client.ListProjectVersions(link, &hubapi.GetListOptions{Q: &q})
	if err == nil {
		cb.success()
	} else {
		cb.failure()
	}
	return val, err
}

// GetProjectVersionRiskProfile ...
func (cb *CircuitBreaker) GetProjectVersionRiskProfile(link hubapi.ResourceLink) (*hubapi.ProjectVersionRiskProfile, error) {
	if !cb.isAbleToIssueRequest() {
		return nil, fmt.Errorf("unable to fetch project version risk profile: circuit breaker disabled")
	}
	val, err := cb.Client.GetProjectVersionRiskProfile(link)
	if err == nil {
		cb.success()
	} else {
		cb.failure()
	}
	return val, err
}

// GetProjectVersionPolicyStatus ...
func (cb *CircuitBreaker) GetProjectVersionPolicyStatus(link hubapi.ResourceLink) (*hubapi.ProjectVersionPolicyStatus, error) {
	if !cb.isAbleToIssueRequest() {
		return nil, fmt.Errorf("unable to fetch project version policy status: circuit breaker disabled")
	}
	val, err := cb.Client.GetProjectVersionPolicyStatus(link)
	if err == nil {
		cb.success()
	} else {
		cb.failure()
	}
	return val, err
}

// ListCodeLocations ...
func (cb *CircuitBreaker) ListCodeLocations(link hubapi.ResourceLink) (*hubapi.CodeLocationList, error) {
	if !cb.isAbleToIssueRequest() {
		return nil, fmt.Errorf("unable to fetch code location list: circuit breaker disabled")
	}
	val, err := cb.Client.ListCodeLocations(link, nil)
	if err == nil {
		cb.success()
	} else {
		cb.failure()
	}
	return val, err
}

// ListScanSummaries ...
func (cb *CircuitBreaker) ListScanSummaries(link hubapi.ResourceLink) (*hubapi.ScanSummaryList, error) {
	if !cb.isAbleToIssueRequest() {
		return nil, fmt.Errorf("unable to fetch scan summary list: circuit breaker disabled")
	}
	val, err := cb.Client.ListScanSummaries(link)
	if err == nil {
		cb.success()
	} else {
		cb.failure()
	}
	return val, err
}

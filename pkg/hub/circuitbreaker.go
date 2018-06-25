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
)

// CircuitBreaker .....
type CircuitBreaker struct {
	Client              ClientInterface
	State               CircuitBreakerState
	NextCheckTime       *time.Time
	MaxBackoffDuration  time.Duration
	ConsecutiveFailures int
	IsEnabledChannel    chan bool
}

// NewCircuitBreaker .....
func NewCircuitBreaker(maxBackoffDuration time.Duration, client ClientInterface) *CircuitBreaker {
	cb := &CircuitBreaker{
		Client:              client,
		NextCheckTime:       nil,
		MaxBackoffDuration:  maxBackoffDuration,
		ConsecutiveFailures: 0,
		IsEnabledChannel:    make(chan bool),
	}
	cb.setState(CircuitBreakerStateEnabled)
	return cb
}

// Reset reenables the circuit breaker regardless of its current state,
// and clears out ConsecutiveFailures and NextCheckTime
func (cb *CircuitBreaker) Reset() {
	cb.setState(CircuitBreakerStateEnabled)
	cb.ConsecutiveFailures = 0
	cb.NextCheckTime = nil
}

func (cb *CircuitBreaker) setState(state CircuitBreakerState) {
	recordCircuitBreakerState(state)
	recordCircuitBreakerTransition(cb.State, state)
	cb.State = state
	go func() {
		switch state {
		case CircuitBreakerStateEnabled:
			cb.IsEnabledChannel <- true
		case CircuitBreakerStateDisabled:
			cb.IsEnabledChannel <- false
		case CircuitBreakerStateChecking:
			break
		}
	}()
}

// IsEnabled .....
func (cb *CircuitBreaker) IsEnabled() bool {
	return cb.State != CircuitBreakerStateDisabled
}

// isAbleToIssueRequest does 3 things:
// 1. changes the state to `Checking` if necessary
// 2. increments a metric of the circuit breaker state
// 3. returns whether the circuit breaker is enabled
func (cb *CircuitBreaker) isAbleToIssueRequest() bool {
	if cb.State == CircuitBreakerStateDisabled && time.Now().After(*cb.NextCheckTime) {
		cb.setState(CircuitBreakerStateChecking)
	}
	isEnabled := cb.IsEnabled()
	recordCircuitBreakerIsEnabled(isEnabled)
	return isEnabled
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
	nextExponentialSeconds := math.Pow(2, float64(cb.ConsecutiveFailures))
	nextCheckDuration := MinDuration(cb.MaxBackoffDuration, time.Duration(nextExponentialSeconds)*time.Second)
	nextCheckTime := time.Now().Add(nextCheckDuration)
	cb.NextCheckTime = &nextCheckTime
}

// // ListAllCodeLocations ...
// func (cb *CircuitBreaker) ListAllCodeLocations() (*hubapi.CodeLocationList, error) {
// 	if !cb.isAbleToIssueRequest() {
// 		return nil, fmt.Errorf("unable to fetch code location list: circuit breaker disabled")
// 	}
// 	start := time.Now()
// 	val, err := cb.Client.ListAllCodeLocations(nil)
// 	recordHubResponseTime("allCodeLocations", time.Now().Sub(start))
// 	recordHubResponse("allCodeLocations", err == nil)
// 	if err == nil {
// 		cb.success()
// 	} else {
// 		cb.failure()
// 	}
// 	return val, err
// }

// ListCodeLocations ...
func (cb *CircuitBreaker) ListCodeLocations(codeLocationName string) (*hubapi.CodeLocationList, error) {
	if !cb.isAbleToIssueRequest() {
		return nil, fmt.Errorf("unable to fetch code location list: circuit breaker disabled")
	}
	queryString := fmt.Sprintf("name:%s", codeLocationName)
	start := time.Now()
	val, err := cb.Client.ListAllCodeLocations(&hubapi.GetListOptions{Q: &queryString})
	recordHubResponseTime("allCodeLocations", time.Now().Sub(start))
	recordHubResponse("allCodeLocations", err == nil)
	if err == nil {
		cb.success()
	} else {
		cb.failure()
	}
	return val, err
}

// GetProjectVersion ...
func (cb *CircuitBreaker) GetProjectVersion(link hubapi.ResourceLink) (*hubapi.ProjectVersion, error) {
	if !cb.isAbleToIssueRequest() {
		return nil, fmt.Errorf("unable to fetch project version: circuit breaker disabled")
	}
	start := time.Now()
	val, err := cb.Client.GetProjectVersion(link)
	recordHubResponseTime("projectVersion", time.Now().Sub(start))
	recordHubResponse("projectVersion", err == nil)
	if err == nil {
		cb.success()
	} else {
		cb.failure()
	}
	return val, err
}

// GetProject ...
func (cb *CircuitBreaker) GetProject(link hubapi.ResourceLink) (*hubapi.Project, error) {
	if !cb.isAbleToIssueRequest() {
		return nil, fmt.Errorf("unable to fetch project: circuit breaker disabled")
	}
	start := time.Now()
	val, err := cb.Client.GetProject(link)
	recordHubResponseTime("project", time.Now().Sub(start))
	recordHubResponse("project", err == nil)
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
	startGetRiskProfile := time.Now()
	val, err := cb.Client.GetProjectVersionRiskProfile(link)
	recordHubResponseTime("projectVersionRiskProfile", time.Now().Sub(startGetRiskProfile))
	recordHubResponse("projectVersionRiskProfile", err == nil)
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
	startGetPolicyStatus := time.Now()
	val, err := cb.Client.GetProjectVersionPolicyStatus(link)
	recordHubResponseTime("projectVersionPolicyStatus", time.Now().Sub(startGetPolicyStatus))
	recordHubResponse("projectVersionPolicyStatus", err == nil)
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
	startGetScanSummaries := time.Now()
	val, err := cb.Client.ListScanSummaries(link)
	recordHubResponseTime("scanSummaries", time.Now().Sub(startGetScanSummaries))
	recordHubResponse("scanSummaries", err == nil)
	if err == nil {
		cb.success()
	} else {
		cb.failure()
	}
	return val, err
}

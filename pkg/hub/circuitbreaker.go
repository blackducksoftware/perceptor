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

	"github.com/blackducksoftware/perceptor/pkg/api"
)

// CircuitBreaker .....
type CircuitBreaker struct {
	state               CircuitBreakerState
	nextCheckTime       *time.Time
	maxBackoffDuration  time.Duration
	consecutiveFailures int
}

// NewCircuitBreaker .....
func NewCircuitBreaker(maxBackoffDuration time.Duration) *CircuitBreaker {
	cb := &CircuitBreaker{
		nextCheckTime:       nil,
		maxBackoffDuration:  maxBackoffDuration,
		consecutiveFailures: 0,
	}
	cb.setState(CircuitBreakerStateEnabled)
	return cb
}

// Model dumps the current state of the circuit breaker
func (cb *CircuitBreaker) Model() *api.ModelCircuitBreaker {
	return &api.ModelCircuitBreaker{
		State:               cb.state.String(),
		ConsecutiveFailures: cb.consecutiveFailures,
		MaxBackoffDuration:  *api.NewModelTime(cb.maxBackoffDuration),
		NextCheckTime:       cb.nextCheckTime,
	}
}

// Reset reenables the circuit breaker regardless of its current state,
// and clears out ConsecutiveFailures and NextCheckTime
func (cb *CircuitBreaker) Reset() {
	cb.setState(CircuitBreakerStateEnabled)
	cb.consecutiveFailures = 0
	cb.nextCheckTime = nil
}

func (cb *CircuitBreaker) setState(state CircuitBreakerState) {
	recordCircuitBreakerState(state)
	recordCircuitBreakerTransition(cb.state, state)
	cb.state = state
}

// IsEnabled .....
func (cb *CircuitBreaker) IsEnabled() bool {
	return cb.state != CircuitBreakerStateDisabled
}

// isAbleToIssueRequest does 3 things:
// 1. changes the state to `Checking` if necessary
// 2. increments a metric of the circuit breaker state
// 3. returns whether the circuit breaker is enabled
func (cb *CircuitBreaker) isAbleToIssueRequest() bool {
	if cb.state == CircuitBreakerStateDisabled && time.Now().After(*cb.nextCheckTime) {
		cb.setState(CircuitBreakerStateChecking)
	}
	isEnabled := cb.IsEnabled()
	recordCircuitBreakerIsEnabled(isEnabled)
	return isEnabled
}

func (cb *CircuitBreaker) failure() {
	switch cb.state {
	case CircuitBreakerStateEnabled:
		cb.setState(CircuitBreakerStateDisabled)
		cb.consecutiveFailures = 1
		cb.setNextCheckTime()
	case CircuitBreakerStateDisabled:
		break
	case CircuitBreakerStateChecking:
		cb.setState(CircuitBreakerStateDisabled)
		cb.consecutiveFailures++
		cb.setNextCheckTime()
	}
}

func (cb *CircuitBreaker) success() {
	switch cb.state {
	case CircuitBreakerStateEnabled:
		break
	case CircuitBreakerStateDisabled:
		break
	case CircuitBreakerStateChecking:
		cb.setState(CircuitBreakerStateEnabled)
		cb.consecutiveFailures = 0
		cb.nextCheckTime = nil
	}
}

func (cb *CircuitBreaker) setNextCheckTime() {
	nextExponentialSeconds := math.Pow(2, float64(cb.consecutiveFailures))
	nextCheckDuration := MinDuration(cb.maxBackoffDuration, time.Duration(nextExponentialSeconds)*time.Second)
	nextCheckTime := time.Now().Add(nextCheckDuration)
	cb.nextCheckTime = &nextCheckTime
}

// IssueRequest synchronously:
//  - checks whether it's enabled
//  - runs 'request'
//  - looks at the result of 'request', disabling itself on failure
func (cb *CircuitBreaker) IssueRequest(description string, request func() error) error {
	if !cb.isAbleToIssueRequest() {
		return fmt.Errorf("unable to issue request %s, circuit breaker is disabled", description)
	}
	start := time.Now()
	err := request()
	recordHubResponseTime(description, time.Now().Sub(start))
	recordHubResponse(description, err == nil)
	if err == nil {
		cb.success()
	} else {
		cb.failure()
	}
	return err
}

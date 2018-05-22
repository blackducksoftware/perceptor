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

import (
	"fmt"
	"math"
	"time"
)

// HubCircuitBreaker .....
type HubCircuitBreaker struct {
	State               HubCircuitBreakerState
	NextCheckTime       *time.Time
	ConsecutiveFailures int
}

// NewHubCircuitBreaker .....
func NewHubCircuitBreaker() *HubCircuitBreaker {
	return &HubCircuitBreaker{
		State:               HubCircuitBreakerStateEnabled,
		NextCheckTime:       nil,
		ConsecutiveFailures: 0,
	}
}

// IsEnabled .....
func (hcb *HubCircuitBreaker) IsEnabled() bool {
	return hcb.State != HubCircuitBreakerStateDisabled
}

// HubFailure .....
func (hcb *HubCircuitBreaker) HubFailure() {
	switch hcb.State {
	case HubCircuitBreakerStateEnabled:
		hcb.State = HubCircuitBreakerStateDisabled
		hcb.ConsecutiveFailures = 1
		hcb.setNextCheckTime()
	case HubCircuitBreakerStateDisabled:
		break
	case HubCircuitBreakerStateChecking:
		hcb.State = HubCircuitBreakerStateDisabled
		hcb.ConsecutiveFailures++
		hcb.setNextCheckTime()
	}
}

// HubSuccess .....
func (hcb *HubCircuitBreaker) HubSuccess() {
	switch hcb.State {
	case HubCircuitBreakerStateEnabled:
		break
	case HubCircuitBreakerStateDisabled:
		break
	case HubCircuitBreakerStateChecking:
		hcb.State = HubCircuitBreakerStateEnabled
		hcb.ConsecutiveFailures = 0
		hcb.NextCheckTime = nil
	}
}

func (hcb *HubCircuitBreaker) setNextCheckTime() {
	nextCheckTime := time.Now().Add(time.Duration(math.Pow(2, float64(hcb.ConsecutiveFailures))) * time.Second)
	hcb.NextCheckTime = &nextCheckTime
}

// MoveToCheckingState .....
func (hcb *HubCircuitBreaker) MoveToCheckingState() error {
	var err error
	switch hcb.State {
	case HubCircuitBreakerStateDisabled:
		hcb.State = HubCircuitBreakerStateChecking
	default:
		err = fmt.Errorf("unable to transition from state %s to HubCircuitBreakerStateChecking", hcb.State.String())
	}
	return err
}

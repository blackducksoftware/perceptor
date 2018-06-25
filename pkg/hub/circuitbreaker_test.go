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
	"testing"
	"time"
)

// TestCircuitBreaker .....
func TestCircuitBreaker(t *testing.T) {
	hubClient := &MockHubClient{ShouldFail: false}
	cb := NewCircuitBreaker(10*time.Minute, hubClient)
	if cb.State != CircuitBreakerStateEnabled {
		t.Errorf("expected CircuitBreakerStateEnabled, found %s", cb.State)
	}

	// API working -> cb remains enabled
	cb.ListCodeLocations("abc")
	if cb.State != CircuitBreakerStateEnabled {
		t.Errorf("expected CircuitBreakerStateEnabled, found %s", cb.State)
	}

	// API fails -> cb gets disabled
	hubClient.ShouldFail = true
	cb.ListCodeLocations("abc")
	if cb.State != CircuitBreakerStateDisabled {
		t.Errorf("expected CircuitBreakerStateDisabled, found %s", cb.State)
	}
	if cb.ConsecutiveFailures != 1 {
		t.Errorf("expected 1, got %d", cb.ConsecutiveFailures)
	}

	// cb disabled -> API calls fail
	clList, err := cb.ListCodeLocations("abc")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if clList != nil {
		t.Errorf("expected nil cl list, got %+v", clList)
	}
	if cb.State != CircuitBreakerStateDisabled {
		t.Errorf("expected CircuitBreakerStateDisabled, found %s", cb.State)
	}
	if cb.ConsecutiveFailures != 1 {
		t.Errorf("expected 1, got %d", cb.ConsecutiveFailures)
	}

	// disabled -> checks -> disabled
	time.Sleep(2 * time.Second)
	clList, err = cb.ListCodeLocations("abc")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if clList != nil {
		t.Errorf("expected nil cl list, got %+v", clList)
	}
	if cb.State != CircuitBreakerStateDisabled {
		t.Errorf("expected CircuitBreakerStateDisabled, found %s", cb.State)
	}
	if cb.ConsecutiveFailures != 2 {
		t.Errorf("expected 1, got %d", cb.ConsecutiveFailures)
	}

	// disabled -> checks -> enabled
	time.Sleep(4 * time.Second)
	hubClient.ShouldFail = false
	clList, err = cb.ListCodeLocations("abc")
	if err != nil {
		t.Errorf("expected nil error, got: %s", err.Error())
	}
	if clList == nil {
		t.Errorf("expected cl list, got nil")
	}
	if cb.State != CircuitBreakerStateEnabled {
		t.Errorf("expected CircuitBreakerStateEnabled, found %s", cb.State)
	}
}

// TestCircuitBreakerConsecutiveFailures .....
func TestCircuitBreakerConsecutiveFailures(t *testing.T) {
	cb := NewCircuitBreaker(10*time.Minute, &MockHubClient{})
	if cb.State != CircuitBreakerStateEnabled {
		t.Errorf("expected CircuitBreakerStateEnabled, found %s", cb.State)
	}

	// cb.HubFailure()
	// if cb.State != CircuitBreakerStateDisabled {
	// 	t.Errorf("expected CircuitBreakerStateDisabled, found %s", cb.State)
	// }
	// assertEqual(t, "state", cb.State, CircuitBreakerStateDisabled)
	// assertEqual(t, "consecutive failures", cb.ConsecutiveFailures, 1)
	// assertEqual(t, "is enabled", cb.IsEnabled(), false)
	//
	// err := cb.MoveToCheckingState()
	// assertEqual(t, "error", err, nil)
	// cb.HubFailure()
	// assertEqual(t, "state", cb.State, CircuitBreakerStateDisabled)
	// assertEqual(t, "consecutive failures", cb.ConsecutiveFailures, 2)
	// assertEqual(t, "is enabled", cb.IsEnabled(), false)
	//
	// err = cb.MoveToCheckingState()
	// assertEqual(t, "error", err, nil)
	// cb.HubFailure()
	// assertEqual(t, "state", cb.State, CircuitBreakerStateDisabled)
	// assertEqual(t, "consecutive failures", cb.ConsecutiveFailures, 3)
	// assertEqual(t, "is enabled", cb.IsEnabled(), false)
}

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
	"testing"
	"time"
)

// TestCircuitBreaker .....
func TestCircuitBreaker(t *testing.T) {
	hubClient := &MockRawClient{ShouldFail: false}
	cb := NewCircuitBreaker("testhost", 10*time.Minute)
	if cb.state != CircuitBreakerStateEnabled {
		t.Errorf("expected CircuitBreakerStateEnabled, found %s", cb.state)
	}

	// API working -> cb remains enabled
	cb.IssueRequest("abc", func() error {
		return nil
	})
	if cb.state != CircuitBreakerStateEnabled {
		t.Errorf("expected CircuitBreakerStateEnabled, found %s", cb.state)
	}

	// API fails -> cb gets disabled
	cb.IssueRequest("abc", func() error {
		return fmt.Errorf("planned failure")
	})
	if cb.state != CircuitBreakerStateDisabled {
		t.Errorf("expected CircuitBreakerStateDisabled, found %s", cb.state)
	}
	if cb.consecutiveFailures != 1 {
		t.Errorf("expected 1, got %d", cb.consecutiveFailures)
	}

	// cb disabled -> API calls fail
	err := cb.IssueRequest("abc", func() error {
		panic("this should never be called!")
	})
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if cb.state != CircuitBreakerStateDisabled {
		t.Errorf("expected CircuitBreakerStateDisabled, found %s", cb.state)
	}
	if cb.consecutiveFailures != 1 {
		t.Errorf("expected 1, got %d", cb.consecutiveFailures)
	}

	// disabled -> checks -> disabled
	time.Sleep(2 * time.Second)
	err = cb.IssueRequest("abc", func() error {
		return fmt.Errorf("planned failure")
	})
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if cb.state != CircuitBreakerStateDisabled {
		t.Errorf("expected CircuitBreakerStateDisabled, found %s", cb.state)
	}
	if cb.consecutiveFailures != 2 {
		t.Errorf("expected 1, got %d", cb.consecutiveFailures)
	}

	// disabled -> checks -> enabled
	time.Sleep(4 * time.Second)
	hubClient.ShouldFail = false
	err = cb.IssueRequest("abc", func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err.Error())
	}
	if cb.state != CircuitBreakerStateEnabled {
		t.Errorf("expected CircuitBreakerStateEnabled, found %s", cb.state)
	}
}

// TestCircuitBreakerConsecutiveFailures .....
func TestCircuitBreakerConsecutiveFailures(t *testing.T) {
	cb := NewCircuitBreaker("testhost", 10*time.Minute)
	if cb.state != CircuitBreakerStateEnabled {
		t.Errorf("expected CircuitBreakerStateEnabled, found %s", cb.state)
	}

	// cb.HubFailure()
	// if cb.state != CircuitBreakerStateDisabled {
	// 	t.Errorf("expected CircuitBreakerStateDisabled, found %s", cb.state)
	// }
	// assertEqual(t, "state", cb.state, CircuitBreakerStateDisabled)
	// assertEqual(t, "consecutive failures", cb.consecutiveFailures, 1)
	// assertEqual(t, "is enabled", cb.IsEnabled(), false)
	//
	// err := cb.MoveToCheckingState()
	// assertEqual(t, "error", err, nil)
	// cb.HubFailure()
	// assertEqual(t, "state", cb.state, CircuitBreakerStateDisabled)
	// assertEqual(t, "consecutive failures", cb.consecutiveFailures, 2)
	// assertEqual(t, "is enabled", cb.IsEnabled(), false)
	//
	// err = cb.MoveToCheckingState()
	// assertEqual(t, "error", err, nil)
	// cb.HubFailure()
	// assertEqual(t, "state", cb.state, CircuitBreakerStateDisabled)
	// assertEqual(t, "consecutive failures", cb.consecutiveFailures, 3)
	// assertEqual(t, "is enabled", cb.IsEnabled(), false)
}

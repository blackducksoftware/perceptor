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

/* TODO maybe define interface for hubclient.Client type, and mock implementation?
import (
	"testing"
)

// TestCircuitBreaker .....
func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker()
	if cb.State != CircuitBreakerStateEnabled {
		t.Errorf("expected CircuitBreakerStateEnabled, found %s", cb.State)
	}
	cb.HubSuccess()
	if cb.State != CircuitBreakerStateEnabled {
		t.Errorf("expected CircuitBreakerStateEnabled, found %s", cb.State)
	}
	cb.HubFailure()
	if cb.State != CircuitBreakerStateDisabled {
		t.Errorf("expected CircuitBreakerStateDisabled, found %s", cb.State)
	}
	cb.HubFailure()
	if cb.State != CircuitBreakerStateDisabled {
		t.Errorf("expected CircuitBreakerStateDisabled, found %s", cb.State)
	}
	cb.HubSuccess()
	if cb.State != CircuitBreakerStateDisabled {
		t.Errorf("expected CircuitBreakerStateDisabled, found %s", cb.State)
	}

	// disabled -> checking -> disabled
	err := cb.MoveToCheckingState()
	if err != nil {
		t.Errorf("unable to change to checking state: %s", err.Error())
	}
	if cb.State != CircuitBreakerStateChecking {
		t.Errorf("expected CircuitBreakerStateChecking, found %s", cb.State)
	}
	cb.HubFailure()
	if cb.State != CircuitBreakerStateDisabled {
		t.Errorf("expected CircuitBreakerStateDisabled, found %s", cb.State)
	}

	// disabled -> checking -> enabled
	err = cb.MoveToCheckingState()
	if err != nil {
		t.Errorf("unable to change to checking state: %s", err.Error())
	}
	if cb.State != CircuitBreakerStateChecking {
		t.Errorf("expected CircuitBreakerStateChecking, found %s", cb.State)
	}
	cb.HubSuccess()
	if cb.State != CircuitBreakerStateEnabled {
		t.Errorf("expected CircuitBreakerStateEnabled, found %s", cb.State)
	}
}

// TestCircuitBreakerConsecutiveFailures .....
func TestCircuitBreakerConsecutiveFailures(t *testing.T) {
	cb := NewCircuitBreaker()
	if cb.State != CircuitBreakerStateEnabled {
		t.Errorf("expected CircuitBreakerStateEnabled, found %s", cb.State)
	}

	cb.HubFailure()
	if cb.State != CircuitBreakerStateDisabled {
		t.Errorf("expected CircuitBreakerStateDisabled, found %s", cb.State)
	}
	assertEqual(t, "state", cb.State, CircuitBreakerStateDisabled)
	assertEqual(t, "consecutive failures", cb.ConsecutiveFailures, 1)
	assertEqual(t, "is enabled", cb.IsEnabled(), false)

	err := cb.MoveToCheckingState()
	assertEqual(t, "error", err, nil)
	cb.HubFailure()
	assertEqual(t, "state", cb.State, CircuitBreakerStateDisabled)
	assertEqual(t, "consecutive failures", cb.ConsecutiveFailures, 2)
	assertEqual(t, "is enabled", cb.IsEnabled(), false)

	err = cb.MoveToCheckingState()
	assertEqual(t, "error", err, nil)
	cb.HubFailure()
	assertEqual(t, "state", cb.State, CircuitBreakerStateDisabled)
	assertEqual(t, "consecutive failures", cb.ConsecutiveFailures, 3)
	assertEqual(t, "is enabled", cb.IsEnabled(), false)
}
*/

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
	"testing"
)

// TestHubCircuitBreaker .....
func TestHubCircuitBreaker(t *testing.T) {
	cb := NewHubCircuitBreaker()
	if cb.State != HubCircuitBreakerStateEnabled {
		t.Errorf("expected HubCircuitBreakerStateEnabled, found %s", cb.State)
	}
	cb.HubSuccess()
	if cb.State != HubCircuitBreakerStateEnabled {
		t.Errorf("expected HubCircuitBreakerStateEnabled, found %s", cb.State)
	}
	cb.HubFailure()
	if cb.State != HubCircuitBreakerStateDisabled {
		t.Errorf("expected HubCircuitBreakerStateDisabled, found %s", cb.State)
	}
	cb.HubFailure()
	if cb.State != HubCircuitBreakerStateDisabled {
		t.Errorf("expected HubCircuitBreakerStateDisabled, found %s", cb.State)
	}
	cb.HubSuccess()
	if cb.State != HubCircuitBreakerStateDisabled {
		t.Errorf("expected HubCircuitBreakerStateDisabled, found %s", cb.State)
	}

	// disabled -> checking -> disabled
	err := cb.MoveToCheckingState()
	if err != nil {
		t.Errorf("unable to change to checking state: %s", err.Error())
	}
	if cb.State != HubCircuitBreakerStateChecking {
		t.Errorf("expected HubCircuitBreakerStateChecking, found %s", cb.State)
	}
	cb.HubFailure()
	if cb.State != HubCircuitBreakerStateDisabled {
		t.Errorf("expected HubCircuitBreakerStateDisabled, found %s", cb.State)
	}

	// disabled -> checking -> enabled
	err = cb.MoveToCheckingState()
	if err != nil {
		t.Errorf("unable to change to checking state: %s", err.Error())
	}
	if cb.State != HubCircuitBreakerStateChecking {
		t.Errorf("expected HubCircuitBreakerStateChecking, found %s", cb.State)
	}
	cb.HubSuccess()
	if cb.State != HubCircuitBreakerStateEnabled {
		t.Errorf("expected HubCircuitBreakerStateEnabled, found %s", cb.State)
	}
}

// TestHubCircuitBreakerConsecutiveFailures .....
func TestHubCircuitBreakerConsecutiveFailures(t *testing.T) {
	cb := NewHubCircuitBreaker()
	if cb.State != HubCircuitBreakerStateEnabled {
		t.Errorf("expected HubCircuitBreakerStateEnabled, found %s", cb.State)
	}

	cb.HubFailure()
	if cb.State != HubCircuitBreakerStateDisabled {
		t.Errorf("expected HubCircuitBreakerStateDisabled, found %s", cb.State)
	}
	assertEqual(t, "state", cb.State, HubCircuitBreakerStateDisabled)
	assertEqual(t, "consecutive failures", cb.ConsecutiveFailures, 1)
	assertEqual(t, "is enabled", cb.IsEnabled(), false)

	err := cb.MoveToCheckingState()
	assertEqual(t, "error", err, nil)
	cb.HubFailure()
	assertEqual(t, "state", cb.State, HubCircuitBreakerStateDisabled)
	assertEqual(t, "consecutive failures", cb.ConsecutiveFailures, 2)
	assertEqual(t, "is enabled", cb.IsEnabled(), false)

	err = cb.MoveToCheckingState()
	assertEqual(t, "error", err, nil)
	cb.HubFailure()
	assertEqual(t, "state", cb.State, HubCircuitBreakerStateDisabled)
	assertEqual(t, "consecutive failures", cb.ConsecutiveFailures, 3)
	assertEqual(t, "is enabled", cb.IsEnabled(), false)
}

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

import "fmt"

// HubCircuitBreakerState .....
type HubCircuitBreakerState int

// .....
const (
	HubCircuitBreakerStateDisabled HubCircuitBreakerState = iota
	HubCircuitBreakerStateEnabled  HubCircuitBreakerState = iota
	HubCircuitBreakerStateChecking HubCircuitBreakerState = iota
)

// String .....
func (state HubCircuitBreakerState) String() string {
	switch state {
	case HubCircuitBreakerStateDisabled:
		return "HubCircuitBreakerStateDisabled"
	case HubCircuitBreakerStateEnabled:
		return "HubCircuitBreakerStateEnabled"
	case HubCircuitBreakerStateChecking:
		return "HubCircuitBreakerStateChecking"
	}
	panic(fmt.Errorf("invalid HubCircuitBreakerState value: %d", state))
}

// MarshalJSON .....
func (state HubCircuitBreakerState) MarshalJSON() ([]byte, error) {
	jsonString := fmt.Sprintf(`"%s"`, state.String())
	return []byte(jsonString), nil
}

// MarshalText .....
func (state HubCircuitBreakerState) MarshalText() (text []byte, err error) {
	return []byte(state.String()), nil
}

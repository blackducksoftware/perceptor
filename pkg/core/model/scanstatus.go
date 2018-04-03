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

// ScanStatus describes the state of an image in perceptor
type ScanStatus int

const (
	ScanStatusUnknown           ScanStatus = iota
	ScanStatusInHubCheckQueue   ScanStatus = iota
	ScanStatusInQueue           ScanStatus = iota
	ScanStatusRunningScanClient ScanStatus = iota
	ScanStatusRunningHubScan    ScanStatus = iota
	ScanStatusComplete          ScanStatus = iota
)

func (status ScanStatus) String() string {
	switch status {
	case ScanStatusUnknown:
		return "ScanStatusUnknown"
	case ScanStatusInHubCheckQueue:
		return "ScanStatusInHubCheckQueue"
	case ScanStatusInQueue:
		return "ScanStatusInQueue"
	case ScanStatusRunningScanClient:
		return "ScanStatusRunningScanClient"
	case ScanStatusRunningHubScan:
		return "ScanStatusRunningHubScan"
	case ScanStatusComplete:
		return "ScanStatusComplete"
	}
	panic(fmt.Errorf("invalid ScanStatus value: %d", status))
}

func (s ScanStatus) MarshalJSON() ([]byte, error) {
	jsonString := fmt.Sprintf(`"%s"`, s.String())
	return []byte(jsonString), nil
}

func (s ScanStatus) MarshalText() (text []byte, err error) {
	return []byte(s.String()), nil
}

var legalTransitions = map[ScanStatus]map[ScanStatus]bool{
	ScanStatusUnknown: {
		ScanStatusInHubCheckQueue: true,
		ScanStatusRunningHubScan:  true,
	},
	ScanStatusInHubCheckQueue: {
		ScanStatusInQueue:        true,
		ScanStatusRunningHubScan: true,
		ScanStatusComplete:       true,
	},
	ScanStatusInQueue: {
		ScanStatusRunningScanClient: true,
		ScanStatusRunningHubScan:    true,
	},
	ScanStatusRunningScanClient: {
		ScanStatusInQueue:        true,
		ScanStatusRunningHubScan: true,
	},
	ScanStatusRunningHubScan: {
		ScanStatusInQueue:  true,
		ScanStatusComplete: true,
	},
	// we never expect to transition FROM complete
	ScanStatusComplete: {},
}

func IsLegalTransition(from ScanStatus, to ScanStatus) bool {
	stateMap, ok := legalTransitions[from]
	if !ok {
		panic(fmt.Errorf("expected to find state transition map for %s but did not", from))
	}
	return stateMap[to]
}

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

type transitionCase struct {
	from    ScanStatus
	to      ScanStatus
	isLegal bool
}

var cases = []transitionCase{
	transitionCase{from: ScanStatusUnknown, to: ScanStatusUnknown, isLegal: false},
	transitionCase{from: ScanStatusUnknown, to: ScanStatusInHubCheckQueue, isLegal: true},
	transitionCase{from: ScanStatusUnknown, to: ScanStatusInQueue, isLegal: false},
	transitionCase{from: ScanStatusUnknown, to: ScanStatusRunningScanClient, isLegal: false},
	transitionCase{from: ScanStatusUnknown, to: ScanStatusRunningHubScan, isLegal: true},
	transitionCase{from: ScanStatusUnknown, to: ScanStatusComplete, isLegal: false},

	transitionCase{from: ScanStatusInHubCheckQueue, to: ScanStatusUnknown, isLegal: false},
	transitionCase{from: ScanStatusInHubCheckQueue, to: ScanStatusInHubCheckQueue, isLegal: false},
	transitionCase{from: ScanStatusInHubCheckQueue, to: ScanStatusInQueue, isLegal: true},
	transitionCase{from: ScanStatusInHubCheckQueue, to: ScanStatusRunningScanClient, isLegal: false},
	transitionCase{from: ScanStatusInHubCheckQueue, to: ScanStatusRunningHubScan, isLegal: true},
	transitionCase{from: ScanStatusInHubCheckQueue, to: ScanStatusComplete, isLegal: true},

	transitionCase{from: ScanStatusInQueue, to: ScanStatusUnknown, isLegal: false},
	transitionCase{from: ScanStatusInQueue, to: ScanStatusInHubCheckQueue, isLegal: false},
	transitionCase{from: ScanStatusInQueue, to: ScanStatusInQueue, isLegal: false},
	transitionCase{from: ScanStatusInQueue, to: ScanStatusRunningScanClient, isLegal: true},
	transitionCase{from: ScanStatusInQueue, to: ScanStatusRunningHubScan, isLegal: true},
	transitionCase{from: ScanStatusInQueue, to: ScanStatusComplete, isLegal: false},

	transitionCase{from: ScanStatusRunningScanClient, to: ScanStatusUnknown, isLegal: false},
	transitionCase{from: ScanStatusRunningScanClient, to: ScanStatusInHubCheckQueue, isLegal: false},
	transitionCase{from: ScanStatusRunningScanClient, to: ScanStatusInQueue, isLegal: true},
	transitionCase{from: ScanStatusRunningScanClient, to: ScanStatusRunningScanClient, isLegal: false},
	transitionCase{from: ScanStatusRunningScanClient, to: ScanStatusRunningHubScan, isLegal: true},
	transitionCase{from: ScanStatusRunningScanClient, to: ScanStatusComplete, isLegal: false},

	transitionCase{from: ScanStatusRunningHubScan, to: ScanStatusUnknown, isLegal: false},
	transitionCase{from: ScanStatusRunningHubScan, to: ScanStatusInHubCheckQueue, isLegal: false},
	transitionCase{from: ScanStatusRunningHubScan, to: ScanStatusInQueue, isLegal: true},
	transitionCase{from: ScanStatusRunningHubScan, to: ScanStatusRunningScanClient, isLegal: false},
	transitionCase{from: ScanStatusRunningHubScan, to: ScanStatusRunningHubScan, isLegal: false},
	transitionCase{from: ScanStatusRunningHubScan, to: ScanStatusComplete, isLegal: true},

	transitionCase{from: ScanStatusComplete, to: ScanStatusUnknown, isLegal: false},
	transitionCase{from: ScanStatusComplete, to: ScanStatusInHubCheckQueue, isLegal: false},
	transitionCase{from: ScanStatusComplete, to: ScanStatusInQueue, isLegal: false},
	transitionCase{from: ScanStatusComplete, to: ScanStatusRunningScanClient, isLegal: false},
	transitionCase{from: ScanStatusComplete, to: ScanStatusRunningHubScan, isLegal: false},
	transitionCase{from: ScanStatusComplete, to: ScanStatusComplete, isLegal: false},
}

func TestLegalTransitions(t *testing.T) {
	for _, testCase := range cases {
		actual := IsLegalTransition(testCase.from, testCase.to)
		expected := testCase.isLegal
		if actual != expected {
			t.Errorf("expected %t for %s to %s, got %t", expected, testCase.from, testCase.to, actual)
		}
	}
}

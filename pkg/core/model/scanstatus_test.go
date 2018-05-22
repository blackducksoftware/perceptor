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
	{from: ScanStatusUnknown, to: ScanStatusUnknown, isLegal: false},
	{from: ScanStatusUnknown, to: ScanStatusInHubCheckQueue, isLegal: true},
	{from: ScanStatusUnknown, to: ScanStatusInQueue, isLegal: false},
	{from: ScanStatusUnknown, to: ScanStatusRunningScanClient, isLegal: false},
	{from: ScanStatusUnknown, to: ScanStatusRunningHubScan, isLegal: true},
	{from: ScanStatusUnknown, to: ScanStatusComplete, isLegal: false},

	{from: ScanStatusInHubCheckQueue, to: ScanStatusUnknown, isLegal: false},
	{from: ScanStatusInHubCheckQueue, to: ScanStatusInHubCheckQueue, isLegal: false},
	{from: ScanStatusInHubCheckQueue, to: ScanStatusInQueue, isLegal: true},
	{from: ScanStatusInHubCheckQueue, to: ScanStatusRunningScanClient, isLegal: false},
	{from: ScanStatusInHubCheckQueue, to: ScanStatusRunningHubScan, isLegal: true},
	{from: ScanStatusInHubCheckQueue, to: ScanStatusComplete, isLegal: true},

	{from: ScanStatusInQueue, to: ScanStatusUnknown, isLegal: false},
	{from: ScanStatusInQueue, to: ScanStatusInHubCheckQueue, isLegal: false},
	{from: ScanStatusInQueue, to: ScanStatusInQueue, isLegal: false},
	{from: ScanStatusInQueue, to: ScanStatusRunningScanClient, isLegal: true},
	{from: ScanStatusInQueue, to: ScanStatusRunningHubScan, isLegal: true},
	{from: ScanStatusInQueue, to: ScanStatusComplete, isLegal: false},

	{from: ScanStatusRunningScanClient, to: ScanStatusUnknown, isLegal: false},
	{from: ScanStatusRunningScanClient, to: ScanStatusInHubCheckQueue, isLegal: false},
	{from: ScanStatusRunningScanClient, to: ScanStatusInQueue, isLegal: true},
	{from: ScanStatusRunningScanClient, to: ScanStatusRunningScanClient, isLegal: false},
	{from: ScanStatusRunningScanClient, to: ScanStatusRunningHubScan, isLegal: true},
	{from: ScanStatusRunningScanClient, to: ScanStatusComplete, isLegal: false},

	{from: ScanStatusRunningHubScan, to: ScanStatusUnknown, isLegal: false},
	{from: ScanStatusRunningHubScan, to: ScanStatusInHubCheckQueue, isLegal: false},
	{from: ScanStatusRunningHubScan, to: ScanStatusInQueue, isLegal: true},
	{from: ScanStatusRunningHubScan, to: ScanStatusRunningScanClient, isLegal: false},
	{from: ScanStatusRunningHubScan, to: ScanStatusRunningHubScan, isLegal: false},
	{from: ScanStatusRunningHubScan, to: ScanStatusComplete, isLegal: true},

	{from: ScanStatusComplete, to: ScanStatusUnknown, isLegal: false},
	{from: ScanStatusComplete, to: ScanStatusInHubCheckQueue, isLegal: false},
	{from: ScanStatusComplete, to: ScanStatusInQueue, isLegal: false},
	{from: ScanStatusComplete, to: ScanStatusRunningScanClient, isLegal: false},
	{from: ScanStatusComplete, to: ScanStatusRunningHubScan, isLegal: false},
	{from: ScanStatusComplete, to: ScanStatusComplete, isLegal: false},
}

// TestLegalTransitions .....
func TestLegalTransitions(t *testing.T) {
	for _, testCase := range cases {
		actual := IsLegalTransition(testCase.from, testCase.to)
		expected := testCase.isLegal
		if actual != expected {
			t.Errorf("expected %t for %s to %s, got %t", expected, testCase.from, testCase.to, actual)
		}
	}
}

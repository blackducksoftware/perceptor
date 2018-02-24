/*
Copyright (C) 2018 Black Duck Software, Inc.

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

package core


import "testing"

func TestString(t *testing.T) {
	if ScanStatusUnknown.String() != "ScanStatusUnknown" {
		t.Errorf("Expected ScanStatusUnknown")
	}

	if ScanStatusInHubCheckQueue.String() != "ScanStatusInHubCheckQueue" {
		t.Errorf("Expected ScanStatusInHubCheckQueue")
	}

	if ScanStatusCheckingHub.String() != "ScanStatusCheckingHub" {
		t.Errorf("Expected ScanStatusCheckingHub")
	}

	if ScanStatusInQueue.String() != "ScanStatusInQueue" {
		t.Errorf("Expected ScanStatusInQueue")
	}

	if ScanStatusRunningScanClient.String() != "ScanStatusRunningScanClient" {
		t.Errorf("Expected ScanStatusRunningScanClient")
	}

	if ScanStatusRunningHubScan.String() != "ScanStatusRunningHubScan" {
		t.Errorf("Expected ScanStatusRunningHubScan")
	}

	if ScanStatusComplete.String() != "ScanStatusComplete" {
		t.Errorf("Expected ScanStatusComplete")
	}

	if ScanStatusError.String() != "ScanStatusError" {
		t.Errorf("Expected ScanStatusError")
	}
}

func TestInvalidScanStatus(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected to cause a panic")
		}
	}()

	ScanStatus(-1).String()
}
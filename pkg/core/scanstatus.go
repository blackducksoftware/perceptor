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

package core

import "fmt"

// ScanStatus describes the state of an image -- have we checked the hub for it?
// Have we scanned it?  Are we scanning it?
type ScanStatus int

// Allowed transitions:
//  - Unknown -> InHubCheckQueue
//  - InHubCheckQueue -> CheckingHub
//  - CheckingHub -> InQueue
//  - CheckingHub -> Complete
//  - InQueue -> RunningScanClient
//  - RunningScanClient -> Error
//  - RunningScanClient -> RunningHubScan
//  - RunningHubScan -> Error
//  - RunningHubScan -> Complete
//  - Error -> ??? throw it back into the queue?
const (
	ScanStatusUnknown           ScanStatus = iota
	ScanStatusInHubCheckQueue   ScanStatus = iota
	ScanStatusCheckingHub       ScanStatus = iota
	ScanStatusInQueue           ScanStatus = iota
	ScanStatusRunningScanClient ScanStatus = iota
	ScanStatusRunningHubScan    ScanStatus = iota
	ScanStatusComplete          ScanStatus = iota
	ScanStatusError             ScanStatus = iota
)

func (status ScanStatus) String() string {
	switch status {
	case ScanStatusUnknown:
		return "ScanStatusUnknown"
	case ScanStatusInHubCheckQueue:
		return "ScanStatusInHubCheckQueue"
	case ScanStatusCheckingHub:
		return "ScanStatusCheckingHub"
	case ScanStatusInQueue:
		return "ScanStatusInQueue"
	case ScanStatusRunningScanClient:
		return "ScanStatusRunningScanClient"
	case ScanStatusRunningHubScan:
		return "ScanStatusRunningHubScan"
	case ScanStatusComplete:
		return "ScanStatusComplete"
	case ScanStatusError:
		return "ScanStatusError"
	}
	panic(fmt.Errorf("invalid ScanStatus value: %d", status))
}

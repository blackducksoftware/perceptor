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

import "fmt"

// ScanSummaryStatus .....
type ScanSummaryStatus int

// .....
const (
	ScanSummaryStatusInProgress ScanSummaryStatus = iota
	ScanSummaryStatusSuccess    ScanSummaryStatus = iota
	ScanSummaryStatusFailure    ScanSummaryStatus = iota
)

// String .....
func (status ScanSummaryStatus) String() string {
	switch status {
	case ScanSummaryStatusInProgress:
		return "ScanSummaryStatusInProgress"
	case ScanSummaryStatusSuccess:
		return "ScanSummaryStatusSuccess"
	case ScanSummaryStatusFailure:
		return "ScanSummaryStatusFailure"
	}
	panic(fmt.Errorf("invalid ScanSummaryStatus value: %d", status))
}

func parseScanSummaryStatus(statusString string) ScanSummaryStatus {
	switch statusString {
	case "COMPLETE":
		return ScanSummaryStatusSuccess
	case "ERROR", "ERROR_BUILDING_BOM", "ERROR_MATCHING", "ERROR_SAVING_SCAN_DATA", "ERROR_SCANNING", "CANCELLED":
		return ScanSummaryStatusFailure
	default:
		return ScanSummaryStatusInProgress
	}
}

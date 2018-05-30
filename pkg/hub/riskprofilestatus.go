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
	"encoding/json"
	"fmt"
)

// RiskProfileStatus .....
type RiskProfileStatus int

// .....
const (
	RiskProfileStatusHigh    RiskProfileStatus = iota
	RiskProfileStatusMedium  RiskProfileStatus = iota
	RiskProfileStatusLow     RiskProfileStatus = iota
	RiskProfileStatusOK      RiskProfileStatus = iota
	RiskProfileStatusUnknown RiskProfileStatus = iota
)

// String .....
func (r RiskProfileStatus) String() string {
	switch r {
	case RiskProfileStatusHigh:
		return "HIGH"
	case RiskProfileStatusMedium:
		return "MEDIUM"
	case RiskProfileStatusLow:
		return "LOW"
	case RiskProfileStatusOK:
		return "OK"
	case RiskProfileStatusUnknown:
		return "UNKNOWN"
	default:
		panic(fmt.Errorf("invalid RiskProfileStatus value: %d", r))
	}
}

// MarshalJSON .....
func (r RiskProfileStatus) MarshalJSON() ([]byte, error) {
	jsonString := fmt.Sprintf(`"%s"`, r.String())
	return []byte(jsonString), nil
}

// UnmarshalJSON .....
func (r *RiskProfileStatus) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	status, err := parseHubRiskProfileStatus(str)
	if err != nil {
		return err
	}
	*r = status
	return nil
}

// MarshalText .....
func (r RiskProfileStatus) MarshalText() (text []byte, err error) {
	return []byte(r.String()), nil
}

// UnmarshalText .....
func (r *RiskProfileStatus) UnmarshalText(text []byte) (err error) {
	status, err := parseHubRiskProfileStatus(string(text))
	if err != nil {
		return err
	}
	*r = status
	return nil
}

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

package api

import "fmt"

// ShouldScanLayer .....
type ShouldScanLayer int

// .....
const (
	ShouldScanLayerNo   ShouldScanLayer = iota
	ShouldScanLayerYes  ShouldScanLayer = iota
	ShouldScanLayerWait ShouldScanLayer = iota
)

// String .....
func (s ShouldScanLayer) String() string {
	switch s {
	case ShouldScanLayerNo:
		return "ShouldScanLayerNo"
	case ShouldScanLayerYes:
		return "ShouldScanLayerYes"
	case ShouldScanLayerWait:
		return "ShouldScanLayerWait"
	}
	panic(fmt.Errorf("invalid ShouldScanLayer value: %d", s))
}

// MarshalJSON .....
func (s ShouldScanLayer) MarshalJSON() ([]byte, error) {
	jsonString := fmt.Sprintf(`"%s"`, s.String())
	return []byte(jsonString), nil
}

// MarshalText .....
func (s ShouldScanLayer) MarshalText() (text []byte, err error) {
	return []byte(s.String()), nil
}

// UnmarshalText .....
func (s *ShouldScanLayer) UnmarshalText(text []byte) (err error) {
	answer, err := parseShouldScanLayer(string(text))
	if err != nil {
		return err
	}
	*s = answer
	return nil
}

func parseShouldScanLayer(value string) (ShouldScanLayer, error) {
	switch value {
	case "ShouldScanLayerNo":
		return ShouldScanLayerNo, nil
	case "ShouldScanLayerYes":
		return ShouldScanLayerYes, nil
	case "ShouldScanLayerWait":
		return ShouldScanLayerWait, nil
	default:
		return ShouldScanLayerNo, fmt.Errorf("invalid value for ShouldScanLayer: %s", value)
	}
}

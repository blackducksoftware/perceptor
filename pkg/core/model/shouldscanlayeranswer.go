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

// ShouldScanLayerAnswer .....
type ShouldScanLayerAnswer int

// .....
const (
	ShouldScanLayerAnswerNo       ShouldScanLayerAnswer = iota
	ShouldScanLayerAnswerYes      ShouldScanLayerAnswer = iota
	ShouldScanLayerAnswerWait     ShouldScanLayerAnswer = iota
	ShouldScanLayerAnswerDontKnow ShouldScanLayerAnswer = iota
)

// String .....
func (s ShouldScanLayerAnswer) String() string {
	switch s {
	case ShouldScanLayerAnswerNo:
		return "ShouldScanLayerAnswerNo"
	case ShouldScanLayerAnswerYes:
		return "ShouldScanLayerAnswerYes"
	case ShouldScanLayerAnswerWait:
		return "ShouldScanLayerAnswerWait"
	case ShouldScanLayerAnswerDontKnow:
		return "ShouldScanLayerAnswerDontKnow"
	}
	panic(fmt.Errorf("invalid ShouldScanLayerAnswer value: %d", s))
}

// MarshalJSON .....
func (s ShouldScanLayerAnswer) MarshalJSON() ([]byte, error) {
	jsonString := fmt.Sprintf(`"%s"`, s.String())
	return []byte(jsonString), nil
}

// MarshalText .....
func (s ShouldScanLayerAnswer) MarshalText() (text []byte, err error) {
	return []byte(s.String()), nil
}

// UnmarshalText .....
func (s *ShouldScanLayerAnswer) UnmarshalText(text []byte) (err error) {
	answer, err := parseShouldScanLayerAnswer(string(text))
	if err != nil {
		return err
	}
	*s = answer
	return nil
}

func parseShouldScanLayerAnswer(value string) (ShouldScanLayerAnswer, error) {
	switch value {
	case "ShouldScanLayerAnswerNo":
		return ShouldScanLayerAnswerNo, nil
	case "ShouldScanLayerAnswerYes":
		return ShouldScanLayerAnswerYes, nil
	case "ShouldScanLayerAnswerWait":
		return ShouldScanLayerAnswerWait, nil
	case "ShouldScanLayerAnswerDontKnow":
		return ShouldScanLayerAnswerDontKnow, nil
	default:
		return ShouldScanLayerAnswerNo, fmt.Errorf("invalid hub name for should scan layer answer: %s", value)
	}
}

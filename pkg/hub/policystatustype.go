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

// PolicyStatusType .....
type PolicyStatusType int

// .....
const (
	PolicyStatusTypeNotInViolation        PolicyStatusType = iota
	PolicyStatusTypeInViolation           PolicyStatusType = iota
	PolicyStatusTypeInViolationOverridden PolicyStatusType = iota
)

// String .....
func (p PolicyStatusType) String() string {
	switch p {
	case PolicyStatusTypeNotInViolation:
		return "NOT_IN_VIOLATION"
	case PolicyStatusTypeInViolation:
		return "IN_VIOLATION"
	case PolicyStatusTypeInViolationOverridden:
		return "IN_VIOLATION_OVERRIDDEN"
	default:
		panic(fmt.Errorf("invalid PolicyStatusType value: %d", p))
	}
}

// MarshalJSON .....
func (p PolicyStatusType) MarshalJSON() ([]byte, error) {
	jsonString := fmt.Sprintf(`"%s"`, p.String())
	return []byte(jsonString), nil
}

// MarshalText .....
func (p PolicyStatusType) MarshalText() (text []byte, err error) {
	return []byte(p.String()), nil
}

// UnmarshalText .....
func (p *PolicyStatusType) UnmarshalText(text []byte) (err error) {
	status, err := parseHubPolicyStatusType(string(text))
	if err != nil {
		return err
	}
	*p = status
	return nil
}

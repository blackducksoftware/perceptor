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

package clustermanager

// BlackDuckAnnotations describes the data model for pod annotation.
type BlackDuckAnnotations struct {
	// TODO remove KeyVals, this is just for testing, to be able
	// to jam random stuff somewhere
	KeyVals              map[string]string
	PolicyViolationCount int
	VulnerabilityCount   int
	OverallStatus        string
}

func NewBlackDuckAnnotations(policyViolationCount int, vulnerabilityCount int, overallStatus string) *BlackDuckAnnotations {
	return &BlackDuckAnnotations{
		PolicyViolationCount: policyViolationCount,
		VulnerabilityCount:   vulnerabilityCount,
		OverallStatus:        overallStatus,
		KeyVals:              make(map[string]string),
	}
}

func (bda *BlackDuckAnnotations) hasPolicyViolations() bool {
	return bda.PolicyViolationCount > 0
}

func (bda *BlackDuckAnnotations) hasVulnerabilities() bool {
	return bda.VulnerabilityCount > 0
}

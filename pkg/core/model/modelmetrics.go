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

type ModelMetrics struct {
	ScanStatusCounts      map[ScanStatus]int
	NumberOfPods          int
	NumberOfImages        int
	ContainerCounts       map[int]int
	ImageCountHistogram   map[int]int
	PodStatus             map[string]int
	ImageStatus           map[string]int
	PodPolicyViolations   map[int]int
	ImagePolicyViolations map[int]int
	PodVulnerabilities    map[int]int
	ImageVulnerabilities  map[int]int
}

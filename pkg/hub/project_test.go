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
	"testing"
)

// TestProjectJSONRoundtrip .....
func TestProjectJSONRoundtrip(t *testing.T) {
	project := Project{
		Name:   "abc",
		Source: "def",
		Versions: []Version{
			{
				CodeLocations: []CodeLocation{
					{
						CodeLocationType:     "yyy",
						CreatedAt:            "zzz",
						MappedProjectVersion: "mpv",
						Name:                 "nnnnm",
						UpdatedAt:            "upd",
						URL:                  "myurl",
						ScanSummaries: []ScanSummary{
							{
								CreatedAt: "crt",
								Status:    ScanSummaryStatusSuccess,
								UpdatedAt: "upd",
							},
						},
					},
				},
				VersionName:  "vn1",
				Distribution: "qrs",
				Nickname:     "nini",
				Phase:        "phs",
				PolicyStatus: PolicyStatus{
					ComponentVersionStatusCounts: map[PolicyStatusType]int{
						PolicyStatusTypeInViolationOverridden: 88,
					},
					OverallStatus: PolicyStatusTypeNotInViolation,
					UpdatedAt:     "updat",
				},
				ReleaseComments: "rcs",
				ReleasedOn:      "ron",
				RiskProfile:     riskProfile,
			},
		},
	}
	jsonBytes, err := json.Marshal(project)
	if err != nil {
		panic(err)
	}
	var unmarshaledProject Project
	err = json.Unmarshal(jsonBytes, &unmarshaledProject)
	if fmt.Sprintf("%+v", project) != fmt.Sprintf("%+v", unmarshaledProject) {
		t.Errorf("expected \n%+v, \ngot \n%+v", project, unmarshaledProject)
	}
}

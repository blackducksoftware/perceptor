// Copyright 2018 Synopsys, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hubapi

import (
	"encoding/json"
	"testing"
)

func parseJSON(jsonText string, result interface{}) error {
	err := json.Unmarshal([]byte(jsonText), result)
	return err
}

var policyStatusJSON = `{
    "overallStatus": "NOT_IN_VIOLATION",
    "updatedAt": "2017-12-13T22:44:20.087Z",
    "componentVersionStatusCounts": [
        {
            "name": "IN_VIOLATION_OVERRIDDEN",
            "value": 0
        },
        {
            "name": "NOT_IN_VIOLATION",
            "value": 14
        },
        {
            "name": "IN_VIOLATION",
            "value": 0
        }
    ],
    "_meta": {
        "allow": [
            "GET"
        ],
        "href": "https://localhost/api/projects/e93317e1-023c-45a8-89fd-19aea01a8d20/versions/5a775cd3-4542-47bc-8497-7eb7c0680430/policy-status",
        "links": []
    }
}`

func TestParsePolicyStatus(t *testing.T) {
	var policyStatus ProjectVersionPolicyStatus
	err := parseJSON(policyStatusJSON, &policyStatus)
	if err != nil {
		t.Log("unable to parse json: " + err.Error())
		t.Fail()
	}
	if policyStatus.OverallStatus != "NOT_IN_VIOLATION" {
		t.Log("incorrectly parsed overallStatus")
		t.Fail()
	}
	if policyStatus.UpdatedAt != "2017-12-13T22:44:20.087Z" {
		t.Log("incorrectly parsed updatedAt")
		t.Fail()
	}
	if len(policyStatus.Meta.Allow) != 1 {
		t.Log("incorrectly parsed _meta.allow")
		t.Fail()
	}
	if policyStatus.Meta.Href != "https://localhost/api/projects/e93317e1-023c-45a8-89fd-19aea01a8d20/versions/5a775cd3-4542-47bc-8497-7eb7c0680430/policy-status" {
		t.Log("incorrectly parsed _meta.href")
		t.Fail()
	}
}

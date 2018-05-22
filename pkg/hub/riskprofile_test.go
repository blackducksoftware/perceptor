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

var riskProfile = RiskProfile{
	BomLastUpdatedAt: "blua",
	Categories: map[RiskProfileCategory]RiskProfileStatusCounts{
		RiskProfileCategoryOperational: {
			StatusCounts: map[RiskProfileStatus]int{
				RiskProfileStatusLow: 31,
			},
		},
	},
}

// TestRiskProfileJSONRoundtrip .....
func TestRiskProfileJSONRoundtrip(t *testing.T) {
	jsonBytes, err := json.Marshal(riskProfile)
	if err != nil {
		panic(err)
	}
	var unmarshaledRiskProfile RiskProfile
	err = json.Unmarshal(jsonBytes, &unmarshaledRiskProfile)
	if err != nil {
		panic(err)
	}
	if fmt.Sprintf("%+v", riskProfile) != fmt.Sprintf("%+v", unmarshaledRiskProfile) {
		t.Errorf("expected \n%+v, \ngot \n%+v,\n json bytes\n%s", riskProfile, unmarshaledRiskProfile, string(jsonBytes))
	}
}

// TestRiskProfileUnmarshal .....
func TestRiskProfileUnmarshal(t *testing.T) {
	input := `{"Categories":{"OPERATIONAL":{"StatusCounts":{"LOW":31}}},"BomLastUpdatedAt":"blua"}`
	var rp RiskProfile
	err := json.Unmarshal([]byte(input), &rp)
	if err != nil {
		panic(err)
	}
	// expected := RiskProfile{
	// 	BomLastUpdatedAt: "blua",
	// 	Categories:       map[RiskProfileCategory]RiskProfileStatusCounts{},
	// }
	// if rp != expected {
	//
	// }
	if rp.BomLastUpdatedAt != "blua" {
		t.Errorf("expected")
	}
}

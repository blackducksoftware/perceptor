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
	"testing"
)

func TestGetListOptionsURLSerialization(t *testing.T) {
	limit := 3
	offset := 12
	q := "a?bc"
	gpo := GetListOptions{
		Limit:  &limit,
		Offset: &offset,
		// skip "Sort", meaning it will be nil, and not show up in the query string
		Q: &q,
	}
	actual := ParameterString(&gpo)
	expected := "limit=3&offset=12&q=a%3Fbc"
	if actual != expected {
		t.Errorf("URL parameters serialized incorrectly -- expected %s, got %s", expected, actual)
		t.Fail()
	}
}

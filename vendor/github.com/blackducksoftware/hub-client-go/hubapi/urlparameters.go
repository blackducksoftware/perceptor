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
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// URLParameters describes types used as parameter models
// for GET endpoints.
type URLParameters interface {
	Parameters() map[string]string
}

// ParameterString takes a URLParameters object
// and converts it to a string which can be added to
// a URL.
func ParameterString(params URLParameters) string {
	dict := params.Parameters()

	var keys []string
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := []string{}
	for _, key := range keys {
		val := dict[key]
		pairs = append(pairs, fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(val)))
	}
	return strings.Join(pairs, "&")
}

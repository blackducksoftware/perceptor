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
)

type Meta struct {
	Allow []string       `json:"allow"`
	Href  string         `json:"href"`
	Links []ResourceLink `json:"links"`
}

type ResourceLink struct {
	Rel   string `json:"rel"`
	Href  string `json:"href"`
	Label string `json:"label"`
	Name  string `json:"name"`
}

func (m *Meta) FindLinkByRel(rel string) (*ResourceLink, error) {

	for _, l := range m.Links {
		if l.Rel == rel {
			return &l, nil
		}
	}

	return nil, fmt.Errorf("no relation '%s' found", rel)
}

// GetListOptions describes the parameter model for the list GET endpoints.
type GetListOptions struct {
	Limit  *int
	Offset *int
	Sort   *string
	Q      *string
}

// Parameters implements the URLParameters interface.
func (glo *GetListOptions) Parameters() map[string]string {
	params := make(map[string]string)
	if glo.Limit != nil {
		params["limit"] = fmt.Sprintf("%d", *glo.Limit)
	}
	if glo.Offset != nil {
		params["offset"] = fmt.Sprintf("%d", *glo.Offset)
	}
	if glo.Sort != nil {
		params["sort"] = *glo.Sort
	}
	if glo.Q != nil {
		params["q"] = *glo.Q
	}
	return params
}

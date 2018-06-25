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

import "fmt"

type CodeLocationList struct {
	TotalCount uint32         `json:"totalCount"`
	Items      []CodeLocation `json:"items"`
	Meta       Meta           `json:"_meta"`
}

type CodeLocation struct {
	Name                 string `json:"name"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	MappedProjectVersion string `json:"mappedProjectVersion"`
	CreatedAt            string `json:"createdAt"`
	UpdatedAt            string `json:"updatedAt"`
	Meta                 Meta   `json:"_meta"`
}

type ScanSummaryList struct {
	TotalCount uint32        `json:"totalCount"`
	Items      []ScanSummary `json:"items"`
	Meta       Meta          `json:"_meta"`
}

type ScanSummary struct {
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Meta      Meta   `json:"_meta"`
}

// I wonder if these can exist to make request as well...
// Or maybe add something to the link itself to make the request?

func (c *CodeLocation) GetScanSummariesLink() (*ResourceLink, error) {
	return c.Meta.FindLinkByRel("scans")
}

func (c *CodeLocation) GetProjectVersionLink() (*ResourceLink, error) {
	if c.MappedProjectVersion == "" {
		return nil, fmt.Errorf("empty mapped project version")
	}
	return &ResourceLink{Href: c.MappedProjectVersion}, nil
}

func (s *ScanSummary) GetCodeLocationLink() (*ResourceLink, error) {
	return s.Meta.FindLinkByRel("codelocation")
}

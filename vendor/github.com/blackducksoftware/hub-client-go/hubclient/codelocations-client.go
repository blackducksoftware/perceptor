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

package hubclient

import (
	"github.com/blackducksoftware/hub-client-go/hubapi"
	log "github.com/sirupsen/logrus"
)

func (c *Client) ListCodeLocations(link hubapi.ResourceLink) (*hubapi.CodeLocationList, error) {

	// Need offset/limit
	// Should we abstract list fetching like we did with a single Get?

	var codeLocationList hubapi.CodeLocationList
	err := c.httpGetJSON(link.Href, &codeLocationList, 200)

	if err != nil {
		log.Errorf("Error trying to retrieve code location list: %+v.", err)
		return nil, err
	}

	return &codeLocationList, nil
}

func (c *Client) GetCodeLocation(link hubapi.ResourceLink) (*hubapi.CodeLocation, error) {

	var codeLocation hubapi.CodeLocation
	err := c.httpGetJSON(link.Href, &codeLocation, 200)

	if err != nil {
		log.Errorf("Error trying to retrieve a code location: %+v.", err)
		return nil, err
	}

	return &codeLocation, nil
}

func (c *Client) ListScanSummaries(link hubapi.ResourceLink) (*hubapi.ScanSummaryList, error) {

	// Need offset/limit
	// Should we abstract list fetching like we did with a single Get?

	var scanSummaryList hubapi.ScanSummaryList
	err := c.httpGetJSON(link.Href, &scanSummaryList, 200)

	if err != nil {
		log.Errorf("Error trying to retrieve scan summary list: %+v.", err)
		return nil, err
	}

	return &scanSummaryList, nil
}

func (c *Client) GetScanSummary(link hubapi.ResourceLink) (*hubapi.ScanSummary, error) {

	var scanSummary hubapi.ScanSummary
	err := c.httpGetJSON(link.Href, &scanSummary, 200)

	if err != nil {
		log.Errorf("Error trying to retrieve a scan summary: %+v.", err)
		return nil, err
	}

	return &scanSummary, nil
}

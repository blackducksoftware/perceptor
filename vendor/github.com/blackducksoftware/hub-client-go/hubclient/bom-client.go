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
	"fmt"

	"github.com/blackducksoftware/hub-client-go/hubapi"
	"github.com/juju/errors"

	log "github.com/sirupsen/logrus"
)

func (c *Client) ListProjectVersionComponents(link hubapi.ResourceLink) (*hubapi.BomComponentList, error) {

	// Need offset/limit
	// Should we abstract list fetching like we did with a single Get?

	var bomList hubapi.BomComponentList
	err := c.HttpGetJSON(link.Href+"?limit=2", &bomList, 200)

	if err != nil {
		return nil, errors.Annotate(err, "Error while trying to get Project Version Component list")
	}

	return &bomList, nil
}

// TODO: Should this be used?
func (c *Client) ListProjectVersionVulnerableComponents(link hubapi.ResourceLink) (*hubapi.BomVulnerableComponentList, error) {

	// Need offset/limit
	// Should we abstract list fetching like we did with a single Get?

	var bomList hubapi.BomVulnerableComponentList
	err := c.HttpGetJSON(link.Href+"?limit=2", &bomList, 200)

	if err != nil {
		return nil, errors.Annotate(err, "Error trying to retrieve vulnerable components list")
	}

	return &bomList, nil
}

func (c *Client) PageProjectVersionVulnerableComponents(link hubapi.ResourceLink, offset uint32, limit uint32) (*hubapi.BomVulnerableComponentList, error) {

	// Should we abstract list fetching like we did with a single Get?

	var bomList hubapi.BomVulnerableComponentList
	url := fmt.Sprintf("%s?offset=%d&limit=%d", link.Href, offset, limit)
	err := c.HttpGetJSON(url, &bomList, 200)

	if err != nil {
		return nil, errors.Annotate(err, "Error trying to retrieve vulnerable components page")
	}

	return &bomList, nil
}

func (c *Client) CountProjectVersionVulnerableComponents(link hubapi.ResourceLink) (uint32, error) {

	// Need offset/limit
	// Should we abstract list fetching like we did with a single Get?

	var bomList hubapi.BomVulnerableComponentList
	err := c.HttpGetJSON(link.Href+"?offset=0&limit=1", &bomList, 200)

	if err != nil {
		return 0, errors.Annotate(err, "Error trying to retrieve count of vulnerable components")
	}

	return bomList.TotalCount, nil
}

func (c *Client) ListAllProjectVersionVulnerableComponents(link hubapi.ResourceLink) ([]hubapi.BomVulnerableComponent, error) {

	log.Debugf("***** Getting total count.")
	//totalCount, err := c.CountProjectVersionVulnerableComponents(link)
	totalCount := uint32(100)
	log.Debugf("***** Got total count: %d", totalCount)

	// if err != nil {
	// 	log.Debugf("ERROR GETTING COUNT: %s\n", err)
	// }

	pageSize := uint32(100)
	result := make([]hubapi.BomVulnerableComponent, totalCount, totalCount)

	for offset := uint32(0); offset < totalCount; offset += pageSize {

		log.Debugf("***** Going to get vulnerable components. Offset: %d, Limit: %d ", offset, pageSize)
		bomPage, err := c.PageProjectVersionVulnerableComponents(link, offset, pageSize)

		if err != nil {
			log.Errorf("Error trying to retrieve vulnerable components list: %+v.", err)
		}

		result = append(result, bomPage.Items...)
	}

	return result, nil
}

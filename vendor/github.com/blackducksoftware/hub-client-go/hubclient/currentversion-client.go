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
)

func (c *Client) CurrentVersion() (*hubapi.CurrentVersion, error) {

	var currentVersion hubapi.CurrentVersion
	currentVersionURL := fmt.Sprintf("%s/api/current-version", c.baseURL)
	err := c.HttpGetJSON(currentVersionURL, &currentVersion, 200)

	if err != nil {
		return nil, errors.Annotate(err, "Error trying to get current version")
	}

	return &currentVersion, nil
}

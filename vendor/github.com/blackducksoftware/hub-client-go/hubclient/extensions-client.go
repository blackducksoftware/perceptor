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

func (c *Client) GetExternalExtension(link hubapi.ResourceLink) (*hubapi.ExternalExtension, error) {

	var extension hubapi.ExternalExtension
	err := c.httpGetJSON(link.Href, &extension, 200)

	if err != nil {
		log.Errorf("Error trying to retrieve an external extension: %+v.", err)
		return nil, err
	}

	return &extension, nil
}

func (c *Client) UpdateExternalExtension(extension *hubapi.ExternalExtension) error {

	err := c.httpPutJSON(extension.Meta.Href, &extension, hubapi.ContentTypeExtensionJSON, 200)

	if err != nil {
		log.Errorf("Error trying to update an external extension: %+v.", err)
		return err
	}

	return nil
}

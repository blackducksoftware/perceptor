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

func (c *Client) ListPolicyRules(options *hubapi.GetListOptions) (*hubapi.PolicyRuleList, error) {
	params := ""
	if options != nil {
		params = fmt.Sprintf("?%s", hubapi.ParameterString(options))
	}

	policyRuleURL := fmt.Sprintf("%s/api/policy-rules%s", c.baseURL, params)

	var policyRuleList hubapi.PolicyRuleList
	err := c.HttpGetJSON(policyRuleURL, &policyRuleList, 200)

	if err != nil {
		return nil, errors.Annotate(err, "Error trying to retrieve policy rule list")
	}

	return &policyRuleList, nil
}

func (c *Client) GetPolicyRule(link hubapi.ResourceLink) (*hubapi.PolicyRule, error) {
	var policyRule hubapi.PolicyRule
	err := c.HttpGetJSON(link.Href, &policyRule, 200)

	if err != nil {
		return nil, errors.Annotate(err, "Error trying to retrieve a policy rule")
	}

	return &policyRule, nil
}

func (c *Client) CreatePolicyRule(policyRuleRequest *hubapi.PolicyRuleRequest) (string, error) {
	policyRuleURL := fmt.Sprintf("%s/api/policy-rules", c.baseURL)
	location, err := c.HttpPostJSON(policyRuleURL, policyRuleRequest, "application/json", 201)

	if err != nil {
		return location, errors.Trace(err)
	}

	if location == "" {
		log.Warnf("Did not get a location header back for policy rule creation")
	}

	return location, err
}

func (c *Client) DeletePolicyRule(policyRuleURL string) error {
	return c.HttpDelete(policyRuleURL, "application/json", 204)
}

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

// What about continuation for these?
// Should we have something where user can pass in an optional continuation/next placeholder?
// Or maybe that is something more for RX?
// Or maybe a special return type that can keep querying for all of them when it runs out?
// Is there any iterator type in GoLang?

func (c *Client) ListProjects(options *hubapi.GetListOptions) (*hubapi.ProjectList, error) {

	params := ""
	if options != nil {
		params = fmt.Sprintf("?%s", hubapi.ParameterString(options))
	}

	projectsURL := fmt.Sprintf("%s/api/projects%s", c.baseURL, params)

	var projectList hubapi.ProjectList
	err := c.HttpGetJSON(projectsURL, &projectList, 200)

	if err != nil {
		return nil, errors.Annotate(err, "Error trying to retrieve project list")
	}

	return &projectList, nil
}

func (c *Client) GetProject(link hubapi.ResourceLink) (*hubapi.Project, error) {

	var project hubapi.Project
	err := c.HttpGetJSON(link.Href, &project, 200)

	if err != nil {
		return nil, errors.Annotate(err, "Error trying to retrieve a project")
	}

	return &project, nil
}

func (c *Client) CreateProject(projectRequest *hubapi.ProjectRequest) (string, error) {

	projectsURL := fmt.Sprintf("%s/api/projects", c.baseURL)
	location, err := c.HttpPostJSON(projectsURL, projectRequest, "application/json", 201)

	if err != nil {
		return location, errors.Trace(err)
	}

	if location == "" {
		log.Warnf("Did not get a location header back for project creation")
	}

	return location, err
}

func (c *Client) DeleteProject(projectURL string) error {
	return c.HttpDelete(projectURL, "application/json", 204)
}

// DeleteProjectVersion deletes a project version using
// https://<base_hub_URL>/api.html#!/project45version45rest45server/deleteVersionUsingDELETE
func (c *Client) DeleteProjectVersion(projectVersionURL string) error {
	return c.HttpDelete(projectVersionURL, "application/json", 204)
}

func (c *Client) ListProjectVersions(link hubapi.ResourceLink, options *hubapi.GetListOptions) (*hubapi.ProjectVersionList, error) {

	params := ""
	if options != nil {
		params = fmt.Sprintf("?%s", hubapi.ParameterString(options))
	}

	projectVersionsURL := fmt.Sprintf("%s%s", link.Href, params)

	var versionList hubapi.ProjectVersionList
	err := c.HttpGetJSON(projectVersionsURL, &versionList, 200)

	if err != nil {
		return nil, errors.Annotate(err, "Error trying to retrieve project version list")
	}

	return &versionList, nil
}

func (c *Client) GetProjectVersion(link hubapi.ResourceLink) (*hubapi.ProjectVersion, error) {

	var projectVersion hubapi.ProjectVersion
	err := c.HttpGetJSON(link.Href, &projectVersion, 200)

	if err != nil {
		return nil, errors.Annotate(err, "Error trying to retrieve a project version")
	}

	return &projectVersion, nil
}

func (c *Client) CreateProjectVersion(link hubapi.ResourceLink, projectVersionRequest *hubapi.ProjectVersionRequest) (string, error) {

	location, err := c.HttpPostJSON(link.Href, projectVersionRequest, "application/json", 201)

	if err != nil {
		return location, errors.Trace(err)
	}

	if location == "" {
		log.Warnf("Did not get a location header back for project version creation")
	}

	return location, err
}

func (c *Client) GetProjectVersionRiskProfile(link hubapi.ResourceLink) (*hubapi.ProjectVersionRiskProfile, error) {

	var riskProfile hubapi.ProjectVersionRiskProfile
	err := c.HttpGetJSON(link.Href, &riskProfile, 200)

	if err != nil {
		return nil, errors.Annotate(err, "Error trying to retrieve a project version risk profile")
	}

	return &riskProfile, nil
}

func (c *Client) GetProjectVersionPolicyStatus(link hubapi.ResourceLink) (*hubapi.ProjectVersionPolicyStatus, error) {

	var policyStatus hubapi.ProjectVersionPolicyStatus
	err := c.HttpGetJSON(link.Href, &policyStatus, 200)

	if err != nil {
		return nil, errors.Annotate(err, "Error trying to retrieve a project version policy status")
	}

	return &policyStatus, nil
}

func (c *Client) AssignUserToProject(link hubapi.ResourceLink, userAssignmentRequest *hubapi.UserAssignmentRequest) (string, error) {

	location, err := c.HttpPostJSON(link.Href, userAssignmentRequest, "application/json", 201)

	if err != nil {
		return location, errors.Trace(err)
	}

	if location == "" {
		log.Warnf("Did not get a location header back for project user assignment")
	}

	return location, err
}

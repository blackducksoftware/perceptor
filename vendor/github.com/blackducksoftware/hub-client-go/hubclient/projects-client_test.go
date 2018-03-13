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
	"testing"
	"time"

	"github.com/blackducksoftware/hub-client-go/hubapi"
)

// TestFetchPolicyStatus is a very brittle test because it requires:
//   1. a reachable hub backend
//   2. the hub backend to be located on localhost
//   3. a specific username and password to be able to log in
//   4. that there is at least one project, with a version, with a policy status
// It's actually an integration test, not a unit test.
func TestFetchPolicyStatus(t *testing.T) {
	client, err := NewWithSession("https://localhost", HubClientDebugTimings, 5*time.Second)
	client.Login("sysadmin", "blackduck")
	if err != nil {
		t.Log("unable to instantiate client: " + err.Error())
		t.Fail()
		return
	}
	projectList, err := client.ListProjects(nil)
	if err != nil {
		t.Log("unable to fetch project list: " + err.Error())
		t.Fail()
		return
	}
	if len((*projectList).Items) == 0 {
		t.Log("this test cannot continue without at least 1 project (found 0)")
		t.Fail()
		return
	}
	project := projectList.Items[0]
	projectVersionsLink, err := project.GetProjectVersionsLink()
	if err != nil {
		t.Log("unable to get project versions link: " + err.Error())
		t.Fail()
		return
	}
	versions, err := client.ListProjectVersions(*projectVersionsLink, nil)
	if err != nil {
		t.Log("unable to fetch project versions list: " + err.Error())
		t.Fail()
		return
	}
	if len((*versions).Items) == 0 {
		t.Log("this test requires at least one project version (found 0)")
		t.Fail()
		return
	}
	version := versions.Items[0]
	policyStatusLink, err := version.GetProjectVersionPolicyStatusLink()
	if err != nil {
		t.Log("unable to get project version policy status link: " + err.Error())
		t.Fail()
		return
	}

	policyStatus, err := client.GetProjectVersionPolicyStatus(*policyStatusLink)
	if err != nil {
		t.Log("unable to fetch policy status: " + err.Error())
		t.Fail()
		return
	}

	/* // uncomment if necessary to help with debugging
	t.Logf("\n policy status: %v\n", policyStatus)
	t.Fail()
	*/

	if len(policyStatus.OverallStatus) == 0 {
		t.Log("expected non-empty overall status")
		t.Fail()
		return
	}

	// check whether project version supports options
	qOption := fmt.Sprintf("versionName:%s", version.VersionName)
	versionsWithOptions, err := client.ListProjectVersions(*projectVersionsLink, &hubapi.GetListOptions{Q: &qOption})
	if err != nil {
		t.Log("unable to fetch project versions list: with options " + err.Error())
		t.Fail()
		return
	}
	if len(versionsWithOptions.Items) != 1 {
		t.Logf("expected one project version with name %s, found %d", version.VersionName, len(versionsWithOptions.Items))
		t.Fail()
		return
	}

}

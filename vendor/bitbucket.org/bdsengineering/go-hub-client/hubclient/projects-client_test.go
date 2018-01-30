package hubclient

import (
	"testing"
)

// TestFetchPolicyStatus is a very brittle test because it requires:
//   1. a reachable hub backend
//   2. the hub backend to be located on localhost
//   3. a specific username and password to be able to log in
//   4. that there is at least one project, with a version, with a policy status
// It's actually an integration test, not a unit test.
func TestFetchPolicyStatus(t *testing.T) {
	client, err := NewWithSession("https://localhost", HubClientDebugTimings)
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
	versions, err := client.ListProjectVersions(*projectVersionsLink)
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

}

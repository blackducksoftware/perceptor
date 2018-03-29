/*
Copyright (C) 2018 Black Duck Software, Inc.

Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements. See the NOTICE file
distributed with this work for additional information
regarding copyright ownership. The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied. See the License for the
specific language governing permissions and limitations
under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/blackducksoftware/hub-client-go/hubapi"
	"github.com/blackducksoftware/hub-client-go/hubclient"
	log "github.com/sirupsen/logrus"
)

func main() {
	url := os.Args[1]
	username := os.Args[2]
	password := os.Args[3]
	var baseURL = fmt.Sprintf("https://%s", url)

	hubClient, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings, 5000*time.Second)
	if err != nil {
		log.Errorf("unable to get hub client: %s", err.Error())
		panic(err)
	}
	err = hubClient.Login(username, password)
	if err != nil {
		log.Errorf("unable to log in to hub: %s", err.Error())
		panic(err)
	}

	limit := 20000 // totally arbitrary number
	projectList, err := hubClient.ListProjects(&hubapi.GetListOptions{Limit: &limit})
	if err != nil {
		panic(err)
	}
	projects := []*Project{}
	for _, project := range projectList.Items {
		project, err := dumpProject(&project, hubClient)
		if err != nil {
			panic(err)
		}
		projects = append(projects, project)
	}
	jsonBytes, err := json.Marshal(projects)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n%s\n", string(jsonBytes))
}

func dumpProject(hubProject *hubapi.Project, hubClient *hubclient.Client) (*Project, error) {
	log.Infof("looking for project %s at url %s", hubProject.Name, hubProject.Meta.Href)
	versions := []*Version{}
	versionsLink, err := hubProject.GetProjectVersionsLink()
	if err != nil {
		return nil, err
	}
	versionsList, err := hubClient.ListProjectVersions(*versionsLink, nil)
	if err != nil {
		return nil, err
	}
	for _, hubVersion := range versionsList.Items {
		version, err := dumpVersion(&hubVersion, hubClient)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	log.Infof("successfully dumped project %s at url %s", hubProject.Name, hubProject.Meta.Href)
	project := &Project{
		Name:        hubProject.Name,
		Versions:    versions,
		Description: hubProject.Description,
		Source:      hubProject.Source}
	return project, nil
}

func dumpVersion(hubVersion *hubapi.ProjectVersion, client *hubclient.Client) (*Version, error) {
	riskProfileLink, err := hubVersion.GetProjectVersionRiskProfileLink()
	if err != nil {
		return nil, err
	}
	hubRiskProfile, err := client.GetProjectVersionRiskProfile(*riskProfileLink)
	if err != nil {
		return nil, err
	}
	riskProfile, err := dumpRiskProfile(hubRiskProfile)

	codeLocations := []*CodeLocation{}
	codeLocationsLink, err := hubVersion.GetCodeLocationsLink()
	if err != nil {
		return nil, err
	}
	hubCodeLocations, err := client.ListCodeLocations(*codeLocationsLink)
	if err != nil {
		return nil, err
	}
	for _, hubCodeLocation := range hubCodeLocations.Items {
		codeLocation, err := dumpCodeLocation(&hubCodeLocation, client)
		if err != nil {
			return nil, err
		}
		codeLocations = append(codeLocations, codeLocation)
	}

	policyStatusLink, err := hubVersion.GetProjectVersionPolicyStatusLink()
	if err != nil {
		return nil, err
	}
	hubPolicyStatus, err := client.GetProjectVersionPolicyStatus(*policyStatusLink)
	if err != nil {
		return nil, err
	}
	policyStatus, err := dumpPolicyStatus(hubPolicyStatus)

	version := &Version{
		Name:            hubVersion.VersionName,
		CodeLocations:   codeLocations,
		RiskProfile:     riskProfile,
		Distribution:    hubVersion.Distribution,
		Meta:            hubVersion.Meta,
		Nickname:        hubVersion.Nickname,
		ReleasedOn:      hubVersion.ReleasedOn,
		ReleaseComments: hubVersion.ReleaseComments,
		PolicyStatus:    policyStatus,
		Phase:           hubVersion.Phase,
	}
	return version, nil
}

func dumpPolicyStatus(hubPolicyStatus *hubapi.ProjectVersionPolicyStatus) (*PolicyStatus, error) {
	statusCounts := []*ComponentVersionStatusCount{}
	for _, hubStatusCount := range hubPolicyStatus.ComponentVersionStatusCounts {
		statusCount := &ComponentVersionStatusCount{
			Name:  hubStatusCount.Name,
			Value: hubStatusCount.Value,
		}
		statusCounts = append(statusCounts, statusCount)
	}
	policyStatus := &PolicyStatus{
		ComponentVersionStatusCounts: statusCounts,
		Meta:          hubPolicyStatus.Meta,
		OverallStatus: hubPolicyStatus.OverallStatus,
		UpdatedAt:     hubPolicyStatus.UpdatedAt,
	}
	return policyStatus, nil
}

func dumpCodeLocation(hubCodeLocation *hubapi.CodeLocation, client *hubclient.Client) (*CodeLocation, error) {
	scanSummaries := []*ScanSummary{}
	link, err := hubCodeLocation.GetScanSummariesLink()
	if err != nil {
		return nil, err
	}
	hubScanSummaries, err := client.ListScanSummaries(*link)
	if err != nil {
		return nil, err
	}
	for _, hubScanSummary := range hubScanSummaries.Items {
		scanSummary, err := dumpScanSummary(&hubScanSummary, client)
		if err != nil {
			return nil, err
		}
		scanSummaries = append(scanSummaries, scanSummary)
	}
	codeLocation := &CodeLocation{
		CreatedAt:            hubCodeLocation.CreatedAt,
		MappedProjectVersion: hubCodeLocation.MappedProjectVersion,
		Meta:                 hubCodeLocation.Meta,
		Name:                 hubCodeLocation.Name,
		ScanSummaries:        scanSummaries,
		Type:                 hubCodeLocation.Type,
		URL:                  hubCodeLocation.URL,
		UpdatedAt:            hubCodeLocation.UpdatedAt,
	}
	return codeLocation, nil
}

func dumpRiskProfile(hubRiskProfile *hubapi.ProjectVersionRiskProfile) (*RiskProfile, error) {
	riskProfile := &RiskProfile{
		BomLastUpdatedAt: hubRiskProfile.BomLastUpdatedAt,
		Categories:       hubRiskProfile.Categories,
		Meta:             hubRiskProfile.Meta,
	}
	return riskProfile, nil
}

func dumpScanSummary(hubScanSummary *hubapi.ScanSummary, client *hubclient.Client) (*ScanSummary, error) {
	scanSummary := &ScanSummary{
		CreatedAt: hubScanSummary.CreatedAt,
		Meta:      hubScanSummary.Meta,
		Status:    hubScanSummary.Status,
		UpdatedAt: hubScanSummary.UpdatedAt,
	}
	return scanSummary, nil
}

//func dumpComponent(hubComponent *hubapi.)

type Project struct {
	Name        string
	Versions    []*Version
	Description string
	Source      string
}

type Version struct {
	Name            string
	CodeLocations   []*CodeLocation
	RiskProfile     *RiskProfile
	Distribution    string
	Meta            hubapi.Meta
	ReleasedOn      string
	ReleaseComments string
	Phase           string
	Nickname        string
	PolicyStatus    *PolicyStatus
}

type PolicyStatus struct {
	ComponentVersionStatusCounts []*ComponentVersionStatusCount
	Meta                         hubapi.Meta
	OverallStatus                string
	UpdatedAt                    string
}

type ComponentVersionStatusCount struct {
	Name  string
	Value int
}

type CodeLocation struct {
	ScanSummaries        []*ScanSummary
	CreatedAt            string
	MappedProjectVersion string
	Meta                 hubapi.Meta
	Name                 string
	Type                 string
	URL                  string
	UpdatedAt            string
}

type RiskProfile struct {
	BomLastUpdatedAt string
	Categories       map[string]map[string]int
	Meta             hubapi.Meta
}

type ScanSummary struct {
	CreatedAt string
	Meta      hubapi.Meta
	Status    string
	UpdatedAt string
}

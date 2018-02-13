/*
Copyright (C) 2018 Synopsys, Inc.

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

	"github.com/blackducksoftware/hub-client-go/hubclient"
	"github.com/blackducksoftware/perceptor/pkg/hub"
	log "github.com/sirupsen/logrus"
)

// var baseURL = "https://localhost"
var baseURL = "https://34.227.56.110.xip.io/"
var username = "sysadmin"
var password = "blackduck"

func main() {
	if len(os.Args) < 2 {
		panic("requires 'verb' arg -- list or get")
	}
	verb := os.Args[1]
	if verb == "list" {
		listProjects()
	} else if verb == "get" {
		projectName := os.Args[2]
		HitHubAPI(projectName)
	} else {
		panic("invalid verb")
	}
}

func listProjects() {
	client, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings)
	if err != nil {
		log.Fatalf("unable to create hub client %v", err)
		panic("oops, unable to create hub client " + err.Error())
	}
	err = client.Login(username, password)
	if err == nil {
		log.Info("success logging in!")
		projects, _ := client.ListProjects(nil)
		// log.Info("projects: %v", projects)
		for _, p := range projects.Items {
			log.Infof("project: %s: %s", p.Name, p.Description)
		}
	} else {
		log.Errorf("unable to log in, %v", err)
	}
}

func HitHubAPI(projectName string) {
	pf, err := hub.NewFetcher(username, password, baseURL)
	if err != nil {
		panic("unable to instantiate ProjectFetcher: " + err.Error())
	}
	project, err := pf.FetchProjectByName(projectName)
	if err != nil {
		panic("unable to fetch project " + projectName + "; " + err.Error())
	}
	bytes, _ := json.Marshal(project)
	log.Infof("fetched project: %v \n\nwith json: %v", project, string(bytes[:]))
	log.Infof("bytes: %d", len(bytes))
}

func exampleHubAPI() {
	client, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings)
	if err != nil {
		log.Fatalf("unable to create hub client %v", err)
		panic("oops, unable to create hub client " + err.Error())
	}
	err = client.Login(username, password)
	if err == nil {
		log.Info("success logging in!")
		projects, _ := client.ListProjects(nil)
		log.Info("projects: %v", projects)
	} else {
		log.Errorf("unable to log in, %v", err)
	}

	projs, err := client.ListProjects(nil)
	if err != nil {
		panic(fmt.Sprintf("error fetching project list: %v", err))
	}
	for _, p := range projs.Items {
		log.Info("proj: ", p)
		log.Info("proj href: ", p.Meta.Href)
		link, err := p.GetProjectVersionsLink()
		if err != nil {
			panic(fmt.Sprintf("error getting project versions link: %v", err))
		}
		versions, err := client.ListProjectVersions(*link, nil)
		if err != nil {
			panic(fmt.Sprintf("error fetching project version: %v", err))
		}
		log.Info("project versions for url: ", link.Href, ": ", versions, "\n\n")

		for _, v := range versions.Items {
			log.Info("version: ", v)
			log.Info("version href: ", v.Meta.Href)
			codeLocationsLink, err := v.GetCodeLocationsLink()
			if err != nil {
				panic(fmt.Sprintf("error getting code locations link: %v", err))
			}
			//codeLocations, err := client.GetCodeLocation(*codeLocationsLink)
			codeLocations, err := client.ListCodeLocations(*codeLocationsLink)
			//			client.
			if err != nil {
				panic(fmt.Sprintf("error fetching code locations: %v", err))
			}
			log.Info("code locations: ", codeLocations)
			for _, codeLocation := range codeLocations.Items {
				scanSummariesLink, err := codeLocation.GetScanSummariesLink()
				if err != nil {
					panic(fmt.Sprintf("error getting scan summaries link: %v", err))
				}
				scanSummaries, err := client.ListScanSummaries(*scanSummariesLink)
				if err != nil {
					panic(fmt.Sprintf("error fetching scan summaries: %v", err))
				}
				for _, scanSummary := range scanSummaries.Items {
					log.Info("scan summary: ", scanSummary)
				}
			}

			riskProfileLink, err := v.GetProjectVersionRiskProfileLink()
			if err != nil {
				panic(fmt.Sprintf("error getting risk profile link: %v", err))
			}
			riskProfile, err := client.GetProjectVersionRiskProfile(*riskProfileLink)
			if err != nil {
				panic(fmt.Sprintf("error fetching project version risk profile: %v", err))
			}
			log.Info("project version risk profile: ", riskProfile)

			// TODO can't get PolicyStatus for now
			// v.GetPolicyStatusLink()

			//scanSummaryLink, err := v.
			log.Info("\n\n")
		}
		log.Info("\n\n\n")
	}
	//	log.Info("projs", projs)
}

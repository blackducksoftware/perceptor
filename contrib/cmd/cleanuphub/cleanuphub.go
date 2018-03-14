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

	limit := 2000
	projectList, err := hubClient.ListProjects(&hubapi.GetListOptions{Limit: &limit})
	if err != nil {
		panic(err)
	}
	for _, project := range projectList.Items {
		deleteProject(project.Meta.Href, hubClient)
	}
}

func deleteProject(projectURL string, hubClient *hubclient.Client) {
	log.Infof("looking to delete project %s", projectURL)
	err := hubClient.DeleteProject(projectURL)
	if err != nil {
		log.Errorf("unable to DELETE project %s : %s", projectURL, err.Error())
		panic(err)
	}
	log.Infof("successfully deleted project %s", projectURL)
}

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
	"os/exec"
	"strings"
	"time"

	"bitbucket.org/bdsengineering/go-hub-client/hubapi"
	"bitbucket.org/bdsengineering/go-hub-client/hubclient"
	clustermanager "github.com/blackducksoftware/perceptor/pkg/clustermanager"
	"github.com/blackducksoftware/perceptor/pkg/common"
	log "github.com/sirupsen/logrus"
)

func main() {
	var baseURL = "https://34.227.56.110.xip.io"
	var insecureBaseURL = "http://34.227.56.110.xip.io"
	var username = "sysadmin"
	var password = "blackduck"

	clusterMasterURL := baseURL + ":8443"
	openshiftMasterUsername := "admin"
	openshiftMasterPassword := "123"

	err := loginToOpenshift(clusterMasterURL, openshiftMasterUsername, openshiftMasterPassword)

	if err != nil {
		log.Errorf("unable to log in to openshift: %s", err.Error())
		panic(err)
	}

	hubClient, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings)
	if err != nil {
		log.Errorf("unable to get hub client: %s", err.Error())
		panic(err)
	}
	err = hubClient.Login(username, password)
	if err != nil {
		log.Errorf("unable to log in to hub: %s", err.Error())
		panic(err)
	}

	kubeconfigPath := "/Users/mfenwick/.kube/config"

	kubeClient, err := clustermanager.NewKubeClient(clusterMasterURL, kubeconfigPath)

	if err != nil {
		log.Errorf("unable to get kube client: %s", err.Error())
		panic(err)
	}

	pods, err := kubeClient.GetAllPods()
	if err != nil {
		log.Errorf("unable to get pods: %s", err.Error())
		panic(err)
	}

	for _, pod := range pods {
		log.Infof("processing pod %+v", pod)
		for _, cont := range pod.Containers {
			deleteImageFromHub(insecureBaseURL, hubClient, cont.Image)
			time.Sleep(1 * time.Second)
		}
	}

}

func deleteImageFromHub(baseURL string, hubClient *hubclient.Client, image common.Image) {
	log.Infof("looking to delete image %+v", image)
	// 1. find a project with the same name
	q := fmt.Sprintf("name:%s", image.HubProjectName())
	projectList, err := hubClient.ListProjects(&hubapi.GetListOptions{Q: &q})
	if err != nil {
		log.Errorf("unable to list projects: %s", err.Error())
		panic(err)
	}

	projects := []hubapi.Project{}
	for _, proj := range projectList.Items {
		if proj.Name == image.HubProjectName() {
			projects = append(projects, proj)
		}
	}

	if len(projects) == 0 {
		return
	}
	if len(projects) > 1 {
		log.Errorf("expected 0 or 1 projects, found %d", len(projects))
		return
	}
	project := projects[0]
	// 4. delete the project
	err = hubClient.DeleteProject(project.Meta.Href)
	if err != nil {
		log.Errorf("unable to DELETE project %s : %s from status code", project.Meta.Href, err.Error())
		panic(err)
	}
}

func splitLast(str string) *string {
	split := strings.Split(str, "/")
	if len(split) > 0 {
		last := split[len(split)-1]
		return &last
	}
	return nil
}

func loginToOpenshift(host string, username string, password string) error {
	// TODO do we need to `oc logout` first?
	cmd := exec.Command("oc", "login", host, "--insecure-skip-tls-verify=true", "-u", username, "-p", password)
	fmt.Println("running command 'oc login ...'")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("unable to login to oc: %s, %s", stdoutStderr, err)
	}
	log.Infof("finished `oc login`: %s", stdoutStderr)
	return err
}

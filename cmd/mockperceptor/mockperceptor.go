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
	"fmt"
	"net/http"
	"time"

	api "github.com/blackducksoftware/perceptor/pkg/api"
	log "github.com/sirupsen/logrus"
)

func main() {
	// Read in Config File
	/*var configPath string
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	log.Infof("Config path: %s", configPath)
	configManager := core.NewConfigManager(configPath, nil)
	config, err := configManager.GetConfig()
	if err != nil {
		log.Errorf("Failed to load configuration: %s", err.Error())
		panic(err)
	}
	if config == nil {
		err = fmt.Errorf("expected non-nil config, but got nil")
		log.Errorf(err.Error())
		panic(err)
	}*/

	// Handle http Requests
	log.Info("Creating Mock Perceptor Responder")
	resp := MockPerceptorResponder{}

	log.Info("Setting Up Listeners")
	api.SetupHTTPServer(&resp)

	port := 8081
	log.Infof("Listening on port: %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

}

// MockResponder .....
type MockPerceptorResponder struct{}

// GetModel .....
func (mr *MockPerceptorResponder) GetModel() api.Model {
	return api.Model{}
}

// AddPod .....
func (mr *MockPerceptorResponder) AddPod(pod api.Pod) error {
	log.Infof("AddPod")
	return nil
}

// UpdatePod .....
func (mr *MockPerceptorResponder) UpdatePod(pod api.Pod) error {
	log.Infof("UpdatePod")
	return nil
}

// DeletePod .....
func (mr *MockPerceptorResponder) DeletePod(qualifiedName string) {
	log.Infof("delete pod: %s", qualifiedName)
}

// GetScanResults .....
func (mr *MockPerceptorResponder) GetScanResults() api.ScanResults {
	log.Info("GetScanResults")
	return api.ScanResults{
		Pods:   nil,
		Images: nil,
	}
}

// AddImage .....
func (mr *MockPerceptorResponder) AddImage(image api.Image) error {
	log.Info("AddImage")
	return nil
}

// UpdateAllPods .....
func (mr *MockPerceptorResponder) UpdateAllPods(allPods api.AllPods) error {
	log.Info("UpdateAllPods")
	return nil
}

// UpdateAllImages .....
func (mr *MockPerceptorResponder) UpdateAllImages(allImages api.AllImages) error {
	log.Info("UpdateAllImages")
	return nil
}

// GetNextImage .....
func (mr *MockPerceptorResponder) GetNextImage() api.NextImage {
	log.Info("GetNextImage")
	start := time.Now().String()
	imageSpec := api.ImageSpec{
		Repository:            "docker.io/alpine",
		Tag:                   "latest",
		Sha:                   "621c2f39f8133acb8e64023a94dbdf0d5ca81896102b9e57c0dc184cadaf5528",
		HubURL:                "",
		HubProjectName:        "string",
		HubProjectVersionName: "string",
		HubScanName:           start,
		Priority:              1,
	}
	return api.NextImage{ImageSpec: &imageSpec}
}

// PostFinishScan .....
func (mr *MockPerceptorResponder) PostFinishScan(job api.FinishedScanClientJob) error {
	log.Infof("PostFinishScan")
	return nil
}

// PostCommand ...
func (mr *MockPerceptorResponder) PostCommand(command *api.PostCommand) {
	// TODO
}

// NotFound .....
func (mr *MockPerceptorResponder) NotFound(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

// Error .....
func (mr *MockPerceptorResponder) Error(w http.ResponseWriter, r *http.Request, err error, statusCode int) {
	http.Error(w, err.Error(), statusCode)
}

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

package federation

import (
	"fmt"
	"os"
	"time"

	"github.com/blackducksoftware/hub-client-go/hubclient"
	"github.com/blackducksoftware/perceptor/pkg/hub"
)

// HubCreator ...
type HubCreator struct {
	hubPassword          string
	hubConfig            *HubConfig
	didFinishHubCreation chan *HubCreationResult
}

// NewHubCreator ...
func NewHubCreator(hubConfig *HubConfig) (*HubCreator, error) {
	hubPassword, ok := os.LookupEnv(hubConfig.PasswordEnvVar)
	if !ok {
		return nil, fmt.Errorf("unable to get Hub password: environment variable %s not set", hubConfig.PasswordEnvVar)
	}
	return &HubCreator{
		hubPassword:          hubPassword,
		hubConfig:            hubConfig,
		didFinishHubCreation: make(chan *HubCreationResult)}, nil
}

func (hc *HubCreator) createHubs(hubHosts map[string]bool) {
	for host := range hubHosts {
		user := hc.hubConfig.User
		port := hc.hubConfig.Port
		timeout := hc.hubConfig.ClientTimeout()
		fetchAllProjectsPause := hc.hubConfig.FetchAllProjectsPause()
		baseURL := fmt.Sprintf("https://%s:%d", host, port)
		rawClient, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings, timeout)
		if err != nil {
			panic(fmt.Errorf("TODO -- don't panic.  handle.  unable to create client for hub %s: %s", host, err.Error()))
		}

		timings := &hub.Timings{
			ScanCompletionPause:    30 * time.Second,
			FetchUnknownScansPause: 30 * time.Second,
			FetchAllScansPause:     fetchAllProjectsPause,
			GetMetricsPause:        hub.DefaultTimings.GetMetricsPause,
			LoginPause:             hub.DefaultTimings.LoginPause,
			RefreshScanThreshold:   hub.DefaultTimings.RefreshScanThreshold,
		}
		hubb := hub.NewHub(user, hc.hubPassword, host, rawClient, timings)
		go func() {
			hc.didFinishHubCreation <- &HubCreationResult{hub: hubb}
		}()
	}
}

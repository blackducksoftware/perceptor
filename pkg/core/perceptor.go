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

package core

import (
	"fmt"
	"os"

	api "github.com/blackducksoftware/perceptor/pkg/api"
	a "github.com/blackducksoftware/perceptor/pkg/core/actions"
	model "github.com/blackducksoftware/perceptor/pkg/core/model"
	"github.com/blackducksoftware/perceptor/pkg/hub"
	log "github.com/sirupsen/logrus"
)

const (
	actionChannelSize = 100
)

// Perceptor ties together: a cluster, scan clients, and a hub.
// It listens to the cluster to learn about new pods.
// It keeps track of pods, containers, images, and scan results in a model.
// It has the hub scan images that have never been seen before.
// It grabs the scan results from the hub and adds them to its model.
// It publishes vulnerabilities that the cluster can find out about.
type Perceptor struct {
	hubClient          hub.FetcherInterface
	httpResponder      *HTTPResponder
	reducer            *reducer
	routineTaskManager *RoutineTaskManager
	// channels
	actions chan a.Action
}

// NewMockedPerceptor creates a Perceptor which uses a mock hub
func NewMockedPerceptor() (*Perceptor, error) {
	mockConfig := model.Config{
		HubHost:             "mock host",
		HubUser:             "mock user",
		ConcurrentScanLimit: 2,
	}
	return newPerceptorHelper(hub.NewMockHub("mock hub version"), &mockConfig), nil
}

// NewPerceptor creates a Perceptor using a real hub client.
func NewPerceptor(config *model.Config) (*Perceptor, error) {
	log.Infof("instantiating perceptor with config %+v", config)
	hubPassword := os.Getenv(config.HubUserPasswordEnvVar)
	if hubPassword == "" {
		return nil, fmt.Errorf("unable to read hub password")
	}

	hubBaseURL := fmt.Sprintf("https://%s:%d", config.HubHost, config.HubPort)
	hubClient, err := hub.NewFetcher(config.HubUser, hubPassword, hubBaseURL, config.HubClientTimeoutMilliseconds)
	if err != nil {
		log.Errorf("unable to instantiate hub Fetcher: %s", err.Error())
		return nil, err
	}

	return newPerceptorHelper(hubClient, config), nil
}

func newPerceptorHelper(hubClient hub.FetcherInterface, config *model.Config) *Perceptor {
	// 1. http responder
	httpResponder := NewHTTPResponder()
	api.SetupHTTPServer(httpResponder)

	// 2. routine task manager
	stop := make(chan struct{})
	routineTaskManager := NewRoutineTaskManager(stop, hubClient, model.DefaultTaskTimingConfig)

	// 3. gather up all actions into a single channel
	actions := make(chan a.Action, actionChannelSize)
	go func() {
		for {
			select {
			case pod := <-httpResponder.AddPodChannel:
				actions <- &a.AddPod{Pod: pod}
			case pod := <-httpResponder.UpdatePodChannel:
				actions <- &a.UpdatePod{Pod: pod}
			case podName := <-httpResponder.DeletePodChannel:
				actions <- &a.DeletePod{PodName: podName}
			case image := <-httpResponder.AddImageChannel:
				actions <- &a.AddImage{Image: image}
			case pods := <-httpResponder.AllPodsChannel:
				actions <- &a.AllPods{Pods: pods}
			case images := <-httpResponder.AllImagesChannel:
				actions <- &a.AllImages{Images: images}
			case job := <-httpResponder.PostFinishScanJobChannel:
				actions <- job
			case continuation := <-httpResponder.PostNextImageChannel:
				actions <- &a.GetNextImage{Continuation: continuation}
			case config := <-httpResponder.PostConfigChannel:
				actions <- &a.SetConfig{ConcurrentScanLimit: config.ConcurrentScanLimit, HubClientTimeoutMilliseconds: config.HubClientTimeoutMilliseconds}
			case continuation := <-httpResponder.GetModelChannel:
				actions <- &a.GetModel{Continuation: continuation}
			case continuation := <-httpResponder.GetScanResultsChannel:
				actions <- &a.GetScanResults{Continuation: continuation}
			case action := <-routineTaskManager.actions:
				actions <- action
			}
		}
	}()

	// 4. now for the reducer
	reducer := newReducer(model.NewModel(config, hubClient.HubVersion()), actions)

	// 5. instantiate perceptor
	perceptor := Perceptor{
		hubClient:          hubClient,
		httpResponder:      httpResponder,
		reducer:            reducer,
		actions:            actions,
		routineTaskManager: routineTaskManager,
	}

	// 6. done
	return &perceptor
}

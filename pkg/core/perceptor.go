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
	mockConfig := Config{
		HubHost:             "mock host",
		HubUser:             "mock user",
		ConcurrentScanLimit: 2,
	}
	return newPerceptorHelper(hub.NewMockHub("mock hub version"), &mockConfig), nil
}

// NewPerceptor creates a Perceptor using a real hub client.
func NewPerceptor(config *Config) (*Perceptor, error) {
	log.Infof("instantiating perceptor with config %+v", config)
	hubPassword, ok := os.LookupEnv(config.HubUserPasswordEnvVar)
	if !ok {
		return nil, fmt.Errorf("unable to get Hub password: environment variable %s not set", config.HubUserPasswordEnvVar)
	}

	hubClient, err := hub.NewFetcher(config.HubUser, hubPassword, config.HubHost, config.HubPort, config.HubClientTimeoutMilliseconds)
	if err != nil {
		log.Errorf("unable to instantiate hub Fetcher: %s", err.Error())
		return nil, err
	}

	return newPerceptorHelper(hubClient, config), nil
}

func newPerceptorHelper(hubClient hub.FetcherInterface, config *Config) *Perceptor {
	// 1. http responder
	httpResponder := NewHTTPResponder()
	api.SetupHTTPServer(httpResponder)

	// 2. routine task manager
	stop := make(chan struct{})
	routineTaskManager := NewRoutineTaskManager(stop, hubClient, model.DefaultTimings)

	// 3. gather up all actions into a single channel
	actions := make(chan a.Action, actionChannelSize)
	go func() {
		for {
			select {
			case actions <- <-httpResponder.AddPodChannel:
			case actions <- <-httpResponder.UpdatePodChannel:
			case actions <- <-httpResponder.DeletePodChannel:
			case actions <- <-httpResponder.AddImageChannel:
			case actions <- <-httpResponder.AllPodsChannel:
			case actions <- <-httpResponder.AllImagesChannel:
			case job := <-httpResponder.PostFinishScanJobChannel:
				actions <- job
			case actions <- <-httpResponder.PostNextImageChannel:
			case config := <-httpResponder.PostConfigChannel:
				actions <- &a.SetConfig{
					ConcurrentScanLimit:                 config.ConcurrentScanLimit,
					HubClientTimeoutMilliseconds:        config.HubClientTimeoutMilliseconds,
					LogLevel:                            config.LogLevel,
					ImageRefreshThresholdSeconds:        config.ImageRefreshThresholdSeconds,
					EnqueueImagesForRefreshPauseSeconds: config.EnqueueImagesForRefreshPauseSeconds,
				}
			case get := <-httpResponder.GetModelChannel:
				// TODO wow, this is such a huge hack.  Root cause: circuit breaker model lives
				// outside of the main model.
				cbModel := hubClient.Model()
				get.HubCircuitBreaker = &api.ModelCircuitBreaker{
					ConsecutiveFailures: cbModel.ConsecutiveFailures,
					NextCheckTime:       cbModel.NextCheckTime,
					State:               cbModel.State.String(),
				}
				actions <- get
			case actions <- <-httpResponder.GetScanResultsChannel:
			case actions <- <-routineTaskManager.actions:
			case isEnabled := <-hubClient.IsEnabled():
				actions <- &a.SetIsHubEnabled{IsEnabled: isEnabled}
			case <-httpResponder.ResetCircuitBreakerChannel:
				hubClient.ResetCircuitBreaker()
			}
		}
	}()

	// 4. now for the reducer
	modelConfig := &model.Config{
		HubHost:               config.HubHost,
		HubPort:               config.HubPort,
		HubUser:               config.HubUser,
		HubUserPasswordEnvVar: config.HubUserPasswordEnvVar,
		LogLevel:              config.LogLevel,
		Port:                  config.Port,
		ConcurrentScanLimit:   config.ConcurrentScanLimit,
	}
	timings := &model.Timings{
		HubClientTimeout:               config.HubClientTimeout(),
		CheckForStalledScansPause:      model.DefaultTimings.CheckForStalledScansPause,
		CheckHubForCompletedScansPause: model.DefaultTimings.CheckHubForCompletedScansPause,
		CheckHubThrottle:               model.DefaultTimings.CheckHubThrottle,
		EnqueueImagesForRefreshPause:   model.DefaultTimings.EnqueueImagesForRefreshPause,
		HubReloginPause:                model.DefaultTimings.HubReloginPause,
		ModelMetricsPause:              model.DefaultTimings.ModelMetricsPause,
		RefreshImagePause:              model.DefaultTimings.RefreshImagePause,
		RefreshThresholdDuration:       model.DefaultTimings.RefreshThresholdDuration,
		StalledScanClientTimeout:       model.DefaultTimings.StalledScanClientTimeout,
	}
	reducer := newReducer(model.NewModel(hubClient.HubVersion(), modelConfig, timings), actions)

	// 5. connect reducer notifications to routine task manager
	go func() {
		for {
			select {
			case timings := <-reducer.Timings:
				routineTaskManager.SetTimings(timings)
			}
		}
	}()

	// 6. perceptor
	perceptor := Perceptor{
		hubClient:          hubClient,
		httpResponder:      httpResponder,
		reducer:            reducer,
		routineTaskManager: routineTaskManager,
		actions:            actions,
	}

	// 7. done
	return &perceptor
}

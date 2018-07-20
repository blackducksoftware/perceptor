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
	"time"

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
	httpResponder      *HTTPResponder
	reducer            *reducer
	routineTaskManager *RoutineTaskManager
	hubs               map[string]hub.FetcherInterface
	// channels
	actions chan a.Action
}

// NewMockedPerceptor creates a Perceptor which uses a mock hub
// TODO
// func NewMockedPerceptor() (*Perceptor, error) {
// 	mockConfig := Config{
// 		HubHost:             "mock host",
// 		HubUser:             "mock user",
// 		ConcurrentScanLimit: 2,
// 	}
// 	return newPerceptorHelper(hub.NewMockHub("mock hub version"), &mockConfig), nil
// }

// NewPerceptor creates a Perceptor using a real hub client.
func NewPerceptor(config *Config) (*Perceptor, error) {
	log.Infof("instantiating perceptor with config %+v", config)
	hubPassword, ok := os.LookupEnv(config.HubUserPasswordEnvVar)
	if !ok {
		return nil, fmt.Errorf("unable to get Hub password: environment variable %s not set", config.HubUserPasswordEnvVar)
	}

	return newPerceptorHelper(config, hubPassword), nil
}

func newPerceptorHelper(config *Config, hubPassword string) *Perceptor {
	// 1. http responder
	httpResponder := NewHTTPResponder()
	api.SetupHTTPServer(httpResponder)

	// 2. routine task manager
	stop := make(chan struct{})
	routineTaskManager := NewRoutineTaskManager(stop, model.DefaultTimings)

	hubFetchers := map[string]hub.FetcherInterface{}

	// 3. gather up all actions into a single channel
	actions := make(chan a.Action, actionChannelSize)
	go func() {
		for {
			select {
			case a := <-httpResponder.AddPodChannel:
				actions <- a
			case a := <-httpResponder.UpdatePodChannel:
				actions <- a
			case a := <-httpResponder.DeletePodChannel:
				actions <- a
			case a := <-httpResponder.AddImageChannel:
				actions <- a
			case a := <-httpResponder.AllPodsChannel:
				actions <- a
			case a := <-httpResponder.AllImagesChannel:
				actions <- a
			case a := <-httpResponder.PostFinishScanJobChannel:
				actions <- a
			case a := <-httpResponder.PostNextImageChannel:
				actions <- a
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
				// TODO extend this to handle ALL hubs
				// cbModel := hubClient.Model()
				// get.HubCircuitBreaker = &api.ModelCircuitBreaker{
				// 	ConsecutiveFailures: cbModel.ConsecutiveFailures,
				// 	NextCheckTime:       cbModel.NextCheckTime,
				// 	State:               cbModel.State.String(),
				// }
				actions <- get
			case a := <-httpResponder.SetHubsChannel:
				actions <- a
			case a := <-httpResponder.GetScanResultsChannel:
				actions <- a
			case a := <-routineTaskManager.actions:
				actions <- a
			// case isEnabled := <-hubClient.IsEnabled():
			// 	log.Warnf("TODO -- read isEnabled from hub managers")
			// actions <- &a.SetIsHubEnabled{IsEnabled: isEnabled}
			case <-httpResponder.ResetCircuitBreakerChannel:
				log.Warnf("TODO -- enable/disable circuit breaker per hub")
				// hubClient.ResetCircuitBreaker()
			}
		}
	}()

	// 4. now for the reducer
	modelConfig := &model.Config{
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
	coreModel := model.NewModel(modelConfig, timings)
	reducer := newReducer(coreModel, actions)
	go func() {
		updates := coreModel.Updates()
		for {
			next := <-updates
			switch u := next.(type) {
			case *model.StartScan:
				break
			case *model.CreateHub:
				if _, ok := hubFetchers[u.HubURL]; !ok {
					hubClientTimeout := time.Millisecond * time.Duration(config.HubClientTimeoutMilliseconds)
					reloginPause := model.DefaultTimings.HubReloginPause
					hubClient, err := hub.NewFetcher(config.HubUser, hubPassword, u.HubURL, config.HubPort, hubClientTimeout, model.DefaultTimings.CheckHubForCompletedScansPause, stop, reloginPause)
					if err != nil {
						panic("TODO handle this intelligently")
						log.Errorf("unable to instantiate hub Fetcher: %s", err.Error())
					}
					hubFetchers[u.HubURL] = hubClient
				} else {
					panic("TODO handle intelligently")
				}
			case *model.DeleteHub:
				// TODO do we need to call a .Stop() method?  Will things leak if we don't?
				// hubFetchers[hubURL].Stop()
				delete(hubFetchers, u.HubURL)
			default:
				panic("unexpected type ")
			}
		}
	}()

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
		httpResponder:      httpResponder,
		reducer:            reducer,
		routineTaskManager: routineTaskManager,
		actions:            actions,
		hubs:               hubFetchers,
	}

	// 7. done
	return &perceptor
}

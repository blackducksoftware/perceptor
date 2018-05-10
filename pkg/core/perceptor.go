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
	"sync"
	"time"

	api "github.com/blackducksoftware/perceptor/pkg/api"
	a "github.com/blackducksoftware/perceptor/pkg/core/actions"
	model "github.com/blackducksoftware/perceptor/pkg/core/model"
	"github.com/blackducksoftware/perceptor/pkg/hub"
	log "github.com/sirupsen/logrus"
)

const (
	checkHubForCompletedScansPause = 20 * time.Second
	checkHubThrottle               = 1 * time.Second

	checkForStalledScansPause = 10000 * time.Hour
	stalledScanClientTimeout  = 3000 * time.Hour

	refreshImagePause = 1000 * time.Hour

	checkHubAccessibilityPause = 5 * time.Second

	enqueueImagesForRefreshPause = 5000 * time.Hour

	modelMetricsPause = 15 * time.Second

	actionChannelSize = 100

	hubReloginPause = 2 * time.Hour
)

// Perceptor ties together: a cluster, scan clients, and a hub.
// It listens to the cluster to learn about new pods.
// It keeps track of pods, containers, images, and scan results in a model.
// It has the hub scan images that have never been seen before.
// It grabs the scan results from the hub and adds them to its model.
// It publishes vulnerabilities that the cluster can find out about.
type Perceptor struct {
	hubClient     hub.FetcherInterface
	httpResponder *HTTPResponder
	// reducer
	reducer *reducer
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
	hubClient, err := hub.NewFetcher(config.HubUser, hubPassword, hubBaseURL, config.HubClientTimeoutSeconds)
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

	// 2. combine actions
	actions := make(chan a.Action, actionChannelSize)
	go func() {
		for {
			select {
			case pod := <-httpResponder.AddPodChannel:
				actions <- &a.AddPod{pod}
			case pod := <-httpResponder.UpdatePodChannel:
				actions <- &a.UpdatePod{pod}
			case podName := <-httpResponder.DeletePodChannel:
				actions <- &a.DeletePod{podName}
			case image := <-httpResponder.AddImageChannel:
				actions <- &a.AddImage{image}
			case pods := <-httpResponder.AllPodsChannel:
				actions <- &a.AllPods{pods}
			case images := <-httpResponder.AllImagesChannel:
				actions <- &a.AllImages{images}
			case job := <-httpResponder.PostFinishScanJobChannel:
				actions <- job
			case continuation := <-httpResponder.PostNextImageChannel:
				actions <- &a.GetNextImage{continuation}
			case limit := <-httpResponder.SetConcurrentScanLimitChannel:
				actions <- &a.SetConcurrentScanLimit{limit}
			case continuation := <-httpResponder.GetModelChannel:
				actions <- &a.GetModel{continuation}
			case continuation := <-httpResponder.GetScanResultsChannel:
				actions <- &a.GetScanResults{continuation}
			}
		}
	}()

	// 3. now for the reducer
	reducer := newReducer(model.NewModel(config, hubClient.HubVersion()), actions)

	// 4. instantiate perceptor
	perceptor := Perceptor{
		hubClient:     hubClient,
		httpResponder: httpResponder,
		reducer:       reducer,
		actions:       actions,
	}

	// 5. start regular tasks -- hitting the hub for results, checking for
	//    stalled scans, model metrics
	go perceptor.startHubInitialScanChecking()
	go perceptor.startPollingHubForScanCompletion()
	go perceptor.startCheckingForStalledScanClientScans()
	go perceptor.startGeneratingModelMetrics()
	go perceptor.startCheckingForUpdatesForCompletedScans()
	go perceptor.startCheckingForHubAccessibility()
	go perceptor.startEnqueueingImagesNeedingRefreshing()
	go perceptor.startReloggingInToHub()

	// 6. done
	return &perceptor
}

func (perceptor *Perceptor) startHubInitialScanChecking() {
	for {
		var wg sync.WaitGroup
		wg.Add(1)
		var image *model.Image
		perceptor.actions <- &a.CheckScanInitial{func(i *model.Image) {
			image = i
			wg.Done()
		}}
		wg.Wait()

		if image != nil {
			scan, err := perceptor.hubClient.FetchScanFromImage(*image)
			perceptor.actions <- &a.FetchScanInitial{&model.HubImageScan{Sha: (*image).Sha, Scan: scan, Err: err}}
			time.Sleep(checkHubThrottle)
		} else {
			// slow down the chatter if we didn't find something
			time.Sleep(checkHubForCompletedScansPause)
		}
	}
}

func (perceptor *Perceptor) startPollingHubForScanCompletion() {
	log.Info("starting to poll hub for scan completion")
	for {
		time.Sleep(checkHubForCompletedScansPause)
		log.Debug("checking hub for completion of running hub scans")
		perceptor.actions <- &a.CheckScansCompletion{func(images *[]model.Image) {
			if images == nil {
				return
			}
			for _, image := range *images {
				scan, err := perceptor.hubClient.FetchScanFromImage(image)
				perceptor.actions <- &a.FetchScanCompletion{&model.HubImageScan{Sha: image.Sha, Scan: scan, Err: err}}
				time.Sleep(checkHubThrottle)
			}
		}}
	}
}

func (perceptor *Perceptor) startCheckingForStalledScanClientScans() {
	log.Info("starting checking for stalled scans")
	for {
		time.Sleep(checkForStalledScansPause)
		log.Info("checking for stalled scans")
		perceptor.actions <- &a.RequeueStalledScans{StalledScanClientTimeout: stalledScanClientTimeout}
	}
}

func (perceptor *Perceptor) startGeneratingModelMetrics() {
	for {
		time.Sleep(modelMetricsPause)

		perceptor.actions <- &a.GetMetrics{func(modelMetrics *model.ModelMetrics) {
			recordModelMetrics(modelMetrics)
		}}
	}
}

func (perceptor *Perceptor) startCheckingForUpdatesForCompletedScans() {
	for {
		log.Info("requesting completed scans for rechecking hub")

		var wg sync.WaitGroup
		wg.Add(1)
		perceptor.actions <- &a.CheckScanRefresh{func(image *model.Image) {
			if image != nil {
				log.Debugf("refreshing image %s", image.PullSpec())
				scan, err := perceptor.hubClient.FetchScanFromImage(*image)
				perceptor.actions <- &a.FetchScanRefresh{&model.HubImageScan{Sha: (*image).Sha, Scan: scan, Err: err}}
			}
			wg.Done()
		}}
		wg.Wait()

		// TODO what should this constant be set to?
		// it's okay if this gets run a lot, and really quickly -- since if the hub
		// becomes unreachable, the circuit breaker will trip
		// and if nothing needs to be refreshed, then this will be cheap
		time.Sleep(refreshImagePause)
	}
}

func (perceptor *Perceptor) startCheckingForHubAccessibility() {
	for {
		perceptor.actions <- &a.CheckHubAccessibility{}
		time.Sleep(checkHubAccessibilityPause)
	}
}

func (perceptor *Perceptor) startEnqueueingImagesNeedingRefreshing() {
	for {
		perceptor.actions <- &a.EnqueueImagesNeedingRefreshing{}
		time.Sleep(enqueueImagesForRefreshPause)
	}
}

func (perceptor *Perceptor) startReloggingInToHub() {
	for {
		time.Sleep(hubReloginPause)

		err := perceptor.hubClient.Login()
		if err != nil {
			log.Errorf("unable to re-login to hub: %s", err.Error())
		}
		log.Infof("successfully re-logged in to hub")
	}
}

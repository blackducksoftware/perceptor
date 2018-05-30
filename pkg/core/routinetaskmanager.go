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
	"sync"
	"time"

	"github.com/blackducksoftware/perceptor/pkg/core/actions"
	"github.com/blackducksoftware/perceptor/pkg/core/model"
	"github.com/blackducksoftware/perceptor/pkg/hub"
	log "github.com/sirupsen/logrus"
)

// RoutineTaskManager manages routine tasks
type RoutineTaskManager struct {
	actions      chan actions.Action
	stop         <-chan struct{}
	readTimings  chan func(model.Timings)
	writeTimings chan model.Timings
	hubClient    hub.FetcherInterface
	Timings      *model.Timings
	// routine tasks
	InitialHubCheckScheduler         *Scheduler
	HubScanCompletionScheduler       *Scheduler
	HubScanRefreshScheduler          *Scheduler
	EnqueueImagesForRefreshScheduler *Scheduler
	ModelMetricsScheduler            *Scheduler
	StalledScanClientScheduler       *Scheduler
	ReloginToHubScheduler            *Scheduler
}

// NewRoutineTaskManager ...
func NewRoutineTaskManager(stop <-chan struct{}, hubClient hub.FetcherInterface, timings *model.Timings) *RoutineTaskManager {
	rtm := &RoutineTaskManager{
		actions:      make(chan actions.Action),
		stop:         stop,
		readTimings:  make(chan func(model.Timings)),
		writeTimings: make(chan model.Timings),
		hubClient:    hubClient,
		Timings:      timings,
	}
	rtm.InitialHubCheckScheduler = rtm.startHubInitialScanChecking()
	rtm.HubScanCompletionScheduler = rtm.startPollingHubForScanCompletion()
	rtm.StalledScanClientScheduler = rtm.startCheckingForStalledScanClientScans()
	rtm.ModelMetricsScheduler = rtm.startGeneratingModelMetrics()
	rtm.HubScanRefreshScheduler = rtm.startCheckingForUpdatesForCompletedScans()
	rtm.ReloginToHubScheduler = rtm.startReloggingInToHub()
	rtm.EnqueueImagesForRefreshScheduler = rtm.startEnqueueingImagesNeedingRefreshing()
	go func() {
		for {
			select {
			case <-stop:
				return
			case continuation := <-rtm.readTimings:
				timings := *rtm.Timings
				go continuation(timings)
			case newTimings := <-rtm.writeTimings:
				rtm.Timings = &newTimings
				rtm.InitialHubCheckScheduler.SetDelay(newTimings.CheckHubThrottle)
				rtm.HubScanCompletionScheduler.SetDelay(newTimings.CheckHubForCompletedScansPause)
				rtm.StalledScanClientScheduler.SetDelay(newTimings.StalledScanClientTimeout)
				rtm.ModelMetricsScheduler.SetDelay(newTimings.ModelMetricsPause)
				rtm.HubScanRefreshScheduler.SetDelay(newTimings.RefreshImagePause)
				rtm.ReloginToHubScheduler.SetDelay(newTimings.HubReloginPause)
				rtm.EnqueueImagesForRefreshScheduler.SetDelay(newTimings.EnqueueImagesForRefreshPause)
			}
		}
	}()
	return rtm
}

// SetTimings sets the timings in a threadsafe way
func (rtm *RoutineTaskManager) SetTimings(newTimings model.Timings) {
	if newTimings.HubClientTimeout != rtm.Timings.HubClientTimeout {
		rtm.hubClient.SetTimeout(newTimings.HubClientTimeout)
	}
	rtm.writeTimings <- newTimings
}

// GetTimings gets the timings in a threadsafe way
func (rtm *RoutineTaskManager) GetTimings() model.Timings {
	var wg sync.WaitGroup
	wg.Add(1)
	var timings model.Timings
	rtm.readTimings <- func(currentTimings model.Timings) {
		timings = currentTimings
		wg.Done()
	}
	wg.Wait()
	return timings
}

func (rtm *RoutineTaskManager) startHubInitialScanChecking() *Scheduler {
	action := func() {
		var wg sync.WaitGroup
		wg.Add(1)
		var image *model.Image
		rtm.actions <- &actions.CheckScanInitial{Continuation: func(i *model.Image) {
			image = i
			wg.Done()
		}}
		wg.Wait()

		if image != nil {
			scan, err := rtm.hubClient.FetchScanFromImage(*image)
			rtm.actions <- &actions.FetchScanInitial{Scan: &model.HubImageScan{Sha: (*image).Sha, Scan: scan, Err: err}}
		}
	}
	return NewScheduler(rtm.Timings.CheckHubThrottle, rtm.stop, action)
}

func (rtm *RoutineTaskManager) startPollingHubForScanCompletion() *Scheduler {
	log.Info("starting to poll hub for scan completion")
	return NewScheduler(rtm.Timings.CheckHubForCompletedScansPause, rtm.stop, func() {
		log.Debug("checking hub for completion of running hub scans")
		rtm.actions <- &actions.CheckScansCompletion{Continuation: func(images *[]model.Image) {
			if images == nil {
				return
			}
			for _, image := range *images {
				scan, err := rtm.hubClient.FetchScanFromImage(image)
				rtm.actions <- &actions.FetchScanCompletion{Scan: &model.HubImageScan{Sha: image.Sha, Scan: scan, Err: err}}
				time.Sleep(rtm.GetTimings().CheckHubThrottle)
			}
		}}
	})
}

func (rtm *RoutineTaskManager) startCheckingForStalledScanClientScans() *Scheduler {
	log.Info("starting checking for stalled scans")
	return NewScheduler(rtm.Timings.CheckForStalledScansPause, rtm.stop, func() {
		log.Debug("checking for stalled scans")
		rtm.actions <- &actions.RequeueStalledScans{StalledScanClientTimeout: rtm.GetTimings().StalledScanClientTimeout}
	})
}

func (rtm *RoutineTaskManager) startGeneratingModelMetrics() *Scheduler {
	return NewScheduler(rtm.Timings.ModelMetricsPause, rtm.stop, func() {
		rtm.actions <- &actions.GetMetrics{Continuation: recordModelMetrics}
	})
}

func (rtm *RoutineTaskManager) startCheckingForUpdatesForCompletedScans() *Scheduler {
	return NewScheduler(rtm.Timings.RefreshImagePause, rtm.stop, func() {
		log.Debug("requesting completed scans for rechecking hub")

		var wg sync.WaitGroup
		wg.Add(1)
		rtm.actions <- &actions.CheckScanRefresh{Continuation: func(image *model.Image) {
			if image != nil {
				log.Debugf("refreshing image %s", image.PullSpec())
				scan, err := rtm.hubClient.FetchScanFromImage(*image)
				rtm.actions <- &actions.FetchScanRefresh{Scan: &model.HubImageScan{Sha: (*image).Sha, Scan: scan, Err: err}}
			}
			wg.Done()
		}}
		wg.Wait()
	})
}

func (rtm *RoutineTaskManager) startEnqueueingImagesNeedingRefreshing() *Scheduler {
	return NewScheduler(rtm.Timings.EnqueueImagesForRefreshPause, rtm.stop, func() {
		log.Debug("enqueueing images in need of refreshing")
		rtm.actions <- &actions.EnqueueImagesNeedingRefreshing{RefreshThresholdDuration: rtm.GetTimings().RefreshThresholdDuration}
	})
}

func (rtm *RoutineTaskManager) startReloggingInToHub() *Scheduler {
	return NewScheduler(rtm.Timings.HubReloginPause, rtm.stop, func() {
		err := rtm.hubClient.Login()
		if err != nil {
			log.Errorf("unable to re-login to hub: %s", err.Error())
		}
		log.Infof("successfully re-logged in to hub")
	})
}

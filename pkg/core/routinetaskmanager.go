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
	actions          chan actions.Action
	stop             <-chan struct{}
	hubClient        hub.FetcherInterface
	TaskTimingConfig *model.TaskTimingConfig
	// routine tasks
	InitialHubCheckScheduler         *Scheduler
	HubScanCompletionScheduler       *Scheduler
	HubScanRefreshScheduler          *Scheduler
	ModelMetricsScheduler            *Scheduler
	StalledScanClientScheduler       *Scheduler
	EnqueueImagesForRefreshScheduler *Scheduler
	ReloginToHubScheduler            *Scheduler
}

// NewRoutineTaskManager ...
func NewRoutineTaskManager(stop <-chan struct{}, hubClient hub.FetcherInterface, config *model.TaskTimingConfig) *RoutineTaskManager {
	rtm := &RoutineTaskManager{
		actions:          make(chan actions.Action),
		stop:             stop,
		hubClient:        hubClient,
		TaskTimingConfig: config,
	}
	rtm.InitialHubCheckScheduler = rtm.startHubInitialScanChecking()
	rtm.HubScanCompletionScheduler = rtm.startPollingHubForScanCompletion()
	rtm.StalledScanClientScheduler = rtm.startCheckingForStalledScanClientScans()
	rtm.ModelMetricsScheduler = rtm.startGeneratingModelMetrics()
	rtm.HubScanRefreshScheduler = rtm.startCheckingForUpdatesForCompletedScans()
	rtm.ReloginToHubScheduler = rtm.startReloggingInToHub()
	rtm.EnqueueImagesForRefreshScheduler = rtm.startEnqueueingImagesNeedingRefreshing()
	return rtm
}

// UpdateConfig ...
func (rtm *RoutineTaskManager) UpdateConfig(config *model.TaskTimingConfig) {
	// TODO
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
	return NewScheduler(rtm.TaskTimingConfig.CheckHubThrottle, rtm.stop, action)
}

func (rtm *RoutineTaskManager) startPollingHubForScanCompletion() *Scheduler {
	log.Info("starting to poll hub for scan completion")
	return NewScheduler(rtm.TaskTimingConfig.CheckHubForCompletedScansPause, rtm.stop, func() {
		time.Sleep(rtm.TaskTimingConfig.CheckHubForCompletedScansPause)
		log.Debug("checking hub for completion of running hub scans")
		rtm.actions <- &actions.CheckScansCompletion{Continuation: func(images *[]model.Image) {
			if images == nil {
				return
			}
			for _, image := range *images {
				scan, err := rtm.hubClient.FetchScanFromImage(image)
				rtm.actions <- &actions.FetchScanCompletion{Scan: &model.HubImageScan{Sha: image.Sha, Scan: scan, Err: err}}
				time.Sleep(rtm.TaskTimingConfig.CheckHubThrottle)
			}
		}}
	})
}

func (rtm *RoutineTaskManager) startCheckingForStalledScanClientScans() *Scheduler {
	log.Info("starting checking for stalled scans")
	return NewScheduler(rtm.TaskTimingConfig.CheckForStalledScansPause, rtm.stop, func() {
		log.Debug("checking for stalled scans")
		rtm.actions <- &actions.RequeueStalledScans{StalledScanClientTimeout: rtm.TaskTimingConfig.StalledScanClientTimeout}
	})
}

func (rtm *RoutineTaskManager) startGeneratingModelMetrics() *Scheduler {
	return NewScheduler(rtm.TaskTimingConfig.ModelMetricsPause, rtm.stop, func() {
		rtm.actions <- &actions.GetMetrics{Continuation: recordModelMetrics}
	})
}

func (rtm *RoutineTaskManager) startCheckingForUpdatesForCompletedScans() *Scheduler {
	return NewScheduler(rtm.TaskTimingConfig.RefreshImagePause, rtm.stop, func() {
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
	return NewScheduler(rtm.TaskTimingConfig.EnqueueImagesForRefreshPause, rtm.stop, func() {
		log.Debug("enqueueing images in need of refreshing")
		rtm.actions <- &actions.EnqueueImagesNeedingRefreshing{RefreshThresholdDuration: rtm.TaskTimingConfig.RefreshThresholdDuration}
	})
}

func (rtm *RoutineTaskManager) startReloggingInToHub() *Scheduler {
	return NewScheduler(rtm.TaskTimingConfig.HubReloginPause, rtm.stop, func() {
		err := rtm.hubClient.Login()
		if err != nil {
			log.Errorf("unable to re-login to hub: %s", err.Error())
		}
		log.Infof("successfully re-logged in to hub")
	})
}

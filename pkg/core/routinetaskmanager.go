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

	"github.com/blackducksoftware/perceptor/pkg/core/actions"
	"github.com/blackducksoftware/perceptor/pkg/core/model"
	"github.com/blackducksoftware/perceptor/pkg/util"
)

// RoutineTaskManager manages routine tasks
type RoutineTaskManager struct {
	actions      chan actions.Action
	stop         <-chan struct{}
	readTimings  chan func(model.Timings)
	writeTimings chan model.Timings
	//	hubClient    hub.FetcherInterface
	Timings *model.Timings
	// routine tasks
	//	EnqueueImagesForRefreshScheduler *util.Scheduler
	ModelMetricsScheduler *util.Scheduler
	//	StalledScanClientScheduler       *util.Scheduler
}

// NewRoutineTaskManager ...
func NewRoutineTaskManager(stop <-chan struct{}, timings *model.Timings) *RoutineTaskManager {
	rtm := &RoutineTaskManager{
		actions:      make(chan actions.Action),
		stop:         stop,
		readTimings:  make(chan func(model.Timings)),
		writeTimings: make(chan model.Timings),
		//hubClient:    hubClient,
		Timings: timings,
	}
	// rtm.StalledScanClientScheduler = rtm.startCheckingForStalledScanClientScans()
	rtm.ModelMetricsScheduler = rtm.startGeneratingModelMetrics()
	// rtm.EnqueueImagesForRefreshScheduler = rtm.startEnqueueingImagesNeedingRefreshing()
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
				// rtm.StalledScanClientScheduler.SetDelay(newTimings.StalledScanClientTimeout)
				rtm.ModelMetricsScheduler.SetDelay(newTimings.ModelMetricsPause)
				// rtm.EnqueueImagesForRefreshScheduler.SetDelay(newTimings.EnqueueImagesForRefreshPause)
			}
		}
	}()
	return rtm
}

// SetTimings sets the timings in a threadsafe way
func (rtm *RoutineTaskManager) SetTimings(newTimings model.Timings) {
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

// func (rtm *RoutineTaskManager) startCheckingForStalledScanClientScans() *util.Scheduler {
// 	log.Info("starting checking for stalled scans")
// 	return util.NewScheduler(rtm.Timings.CheckForStalledScansPause, rtm.stop, func() {
// 		log.Debug("checking for stalled scans")
// 		rtm.actions <- &actions.RequeueStalledScans{StalledScanClientTimeout: rtm.GetTimings().StalledScanClientTimeout}
// 	})
// }

func (rtm *RoutineTaskManager) startGeneratingModelMetrics() *util.Scheduler {
	return util.NewScheduler(rtm.Timings.ModelMetricsPause, rtm.stop, func() {
		rtm.actions <- &actions.GetMetrics{Continuation: recordModelMetrics}
	})
}

// func (rtm *RoutineTaskManager) startEnqueueingImagesNeedingRefreshing() *util.Scheduler {
// 	return util.NewScheduler(rtm.Timings.EnqueueImagesForRefreshPause, rtm.stop, func() {
// 		log.Debug("enqueueing images in need of refreshing")
// 		rtm.actions <- &actions.EnqueueImagesNeedingRefreshing{RefreshThresholdDuration: rtm.GetTimings().RefreshThresholdDuration}
// 	})
// }

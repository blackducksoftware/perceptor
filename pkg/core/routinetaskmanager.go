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
	"time"

	"github.com/blackducksoftware/perceptor/pkg/core/model"
	"github.com/blackducksoftware/perceptor/pkg/util"
	log "github.com/sirupsen/logrus"
)

// RoutineTaskManager manages routine tasks
type RoutineTaskManager struct {
	//	actions chan UmWhatType
	stop <-chan struct{}
	// readTimings    chan func(model.Timings)
	// writeTimings   chan model.Timings
	// Timings        *model.Timings
	OrphanedImages chan []string
	// routine tasks
	ModelMetricsScheduler        *util.Scheduler
	StalledScanClientScheduler   *util.Scheduler
	PruneOrphanedImagesScheduler *util.Scheduler
}

// Timings ??? TODO
type Timings struct{}

// NewRoutineTaskManager ...
func NewRoutineTaskManager(stop <-chan struct{}, pruneOrphanedImagesPause time.Duration, timings *Timings) *RoutineTaskManager {
	rtm := &RoutineTaskManager{
		actions: make(chan UmWhatType),
		stop:    stop,
		// readTimings:  make(chan func(Timings)),
		// writeTimings: make(chan Timings),
		hubClient: hubClient,
		Timings:   timings,
	}
	rtm.StalledScanClientScheduler = rtm.startCheckingForStalledScanClientScans()
	rtm.ModelMetricsScheduler = rtm.startGeneratingModelMetrics()
	if pruneOrphanedImagesPause > 0 {
		rtm.PruneOrphanedImagesScheduler = rtm.startPruningOrphanedImages(pruneOrphanedImagesPause)
		rtm.OrphanedImages = make(chan []string)
	}
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
				rtm.StalledScanClientScheduler.SetDelay(newTimings.StalledScanClientTimeout)
				rtm.ModelMetricsScheduler.SetDelay(newTimings.ModelMetricsPause)
			}
		}
	}()
	return rtm
}

// SetTimings sets the timings in a threadsafe way
func (rtm *RoutineTaskManager) SetTimings(newTimings Timings) {
	if newTimings.HubClientTimeout != rtm.Timings.HubClientTimeout {
		rtm.hubClient.SetTimeout(newTimings.HubClientTimeout)
	}
	rtm.writeTimings <- newTimings
}

// // GetTimings gets the timings in a threadsafe way
// func (rtm *RoutineTaskManager) GetTimings() Timings {
// 	var wg sync.WaitGroup
// 	wg.Add(1)
// 	var timings model.Timings
// 	rtm.readTimings <- func(currentTimings model.Timings) {
// 		timings = currentTimings
// 		wg.Done()
// 	}
// 	wg.Wait()
// 	return timings
// }

func (rtm *RoutineTaskManager) startCheckingForStalledScanClientScans() *util.Scheduler {
	log.Info("starting checking for stalled scans")
	return util.NewRunningScheduler("stalledScanClient", rtm.Timings.CheckForStalledScansPause, rtm.stop, false, func() {
		log.Debug("checking for stalled scans")
		rtm.actions <- &model.RequeueStalledScans{StalledScanClientTimeout: rtm.GetTimings().StalledScanClientTimeout}
	})
}

func (rtm *RoutineTaskManager) startGeneratingModelMetrics() *util.Scheduler {
	return util.NewRunningScheduler("modelMetrics", rtm.Timings.ModelMetricsPause, rtm.stop, false, func() {
		rtm.actions <- &model.GetMetrics{Continuation: recordModelMetrics}
	})
}

func (rtm *RoutineTaskManager) startPruningOrphanedImages(pause time.Duration) *util.Scheduler {
	return util.NewRunningScheduler("orphanedImagePruning", pause, rtm.stop, false, func() {
		log.Debug("cleaning up orphaned images")
		action := &model.PruneOrphanedImages{CompletedImageShas: make(chan []string)}
		rtm.actions <- action
		images := <-action.CompletedImageShas
		rtm.OrphanedImages <- images
	})
}

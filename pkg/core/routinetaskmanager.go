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
	"time"

	"github.com/blackducksoftware/perceptor/pkg/util"
	log "github.com/sirupsen/logrus"
)

// RoutineTaskManager manages routine tasks
type RoutineTaskManager struct {
	stop           <-chan struct{}
	readTimings    chan chan *Timings
	writeTimings   chan *Timings
	timings        *Timings
	OrphanedImages chan []string
	// routine tasks
	ModelMetricsScheduler        *util.Scheduler
	StalledScanClientScheduler   *util.Scheduler
	PruneOrphanedImagesScheduler *util.Scheduler
}

// Timings ??? TODO
type Timings struct {
	CheckForStalledScansPause time.Duration
	StalledScanClientTimeout  time.Duration
	ModelMetricsPause         time.Duration
}

// NewRoutineTaskManager ...
func NewRoutineTaskManager(stop <-chan struct{}, pruneOrphanedImagesPause time.Duration, timings *Timings) *RoutineTaskManager {
	rtm := &RoutineTaskManager{
		stop:         stop,
		readTimings:  make(chan chan *Timings),
		writeTimings: make(chan *Timings),
		timings:      timings,
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
			case ch := <-rtm.readTimings:
				timings := rtm.timings
				go func() {
					select {
					case ch <- timings:
					case <-stop:
					}
				}()
			case newTimings := <-rtm.writeTimings:
				rtm.timings = newTimings
				rtm.StalledScanClientScheduler.SetDelay(newTimings.StalledScanClientTimeout)
				rtm.ModelMetricsScheduler.SetDelay(newTimings.ModelMetricsPause)
			}
		}
	}()
	return rtm
}

// SetTimings sets the timings in a threadsafe way
func (rtm *RoutineTaskManager) SetTimings(newTimings *Timings) {
	rtm.writeTimings <- newTimings
}

// GetTimings gets the timings in a threadsafe way
func (rtm *RoutineTaskManager) GetTimings() (*Timings, error) {
	ch := make(chan *Timings)
	rtm.readTimings <- ch
	select {
	case timings := <-ch:
		return timings, nil
	case <-rtm.stop:
		return nil, fmt.Errorf("cannot get timings: rtm is stopped")
	}
}

func (rtm *RoutineTaskManager) startCheckingForStalledScanClientScans() *util.Scheduler {
	log.Info("starting checking for stalled scans")
	return util.NewRunningScheduler("stalledScanClient", rtm.timings.CheckForStalledScansPause, rtm.stop, false, func() {
		log.Debug("checking for stalled scans")
		// TODO write to a channel or something?
	})
}

func (rtm *RoutineTaskManager) startGeneratingModelMetrics() *util.Scheduler {
	return util.NewRunningScheduler("modelMetrics", rtm.timings.ModelMetricsPause, rtm.stop, false, func() {
		// TODO write to a channel or something?
	})
}

func (rtm *RoutineTaskManager) startPruningOrphanedImages(pause time.Duration) *util.Scheduler {
	return util.NewRunningScheduler("orphanedImagePruning", pause, rtm.stop, false, func() {
		log.Debug("cleaning up orphaned images")
		// TODO write to a channel or something?
	})
}

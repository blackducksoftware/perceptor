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
	stop         <-chan struct{}
	readTimings  chan chan *Timings
	writeTimings chan *Timings
	timings      *Timings
	// timers
	modelMetricsTimer        *util.Timer
	stalledScanClientTimer   *util.Timer
	pruneOrphanedImagesTimer *util.Timer
	unknownImagesTimer       *util.Timer
	// channels
	metricsCh       chan bool
	orphanedImages  chan []string
	unknownImagesCh chan bool
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
		stop:            stop,
		readTimings:     make(chan chan *Timings),
		writeTimings:    make(chan *Timings),
		timings:         timings,
		metricsCh:       make(chan bool),
		unknownImagesCh: make(chan bool),
	}
	rtm.stalledScanClientTimer = rtm.startCheckingForStalledScanClientScans()
	rtm.modelMetricsTimer = rtm.startGeneratingModelMetrics()
	if pruneOrphanedImagesPause > 0 {
		rtm.pruneOrphanedImagesTimer = rtm.startPruningOrphanedImages(pruneOrphanedImagesPause)
		rtm.orphanedImages = make(chan []string)
	}
	rtm.unknownImagesTimer = rtm.startCheckingForUnknownImages(5 * time.Minute)
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
				rtm.stalledScanClientTimer.SetDelay(newTimings.StalledScanClientTimeout)
				rtm.modelMetricsTimer.SetDelay(newTimings.ModelMetricsPause)
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

func (rtm *RoutineTaskManager) startCheckingForStalledScanClientScans() *util.Timer {
	log.Info("starting checking for stalled scans")
	return util.NewRunningTimer("stalledScanClient", rtm.timings.CheckForStalledScansPause, rtm.stop, false, func() {
		log.Debug("checking for stalled scans")
		// TODO write to a channel or something?
	})
}

func (rtm *RoutineTaskManager) startGeneratingModelMetrics() *util.Timer {
	return util.NewRunningTimer("modelMetrics", rtm.timings.ModelMetricsPause, rtm.stop, false, func() {
		select {
		case <-rtm.stop:
			return
		case rtm.metricsCh <- true:
		}
	})
}

func (rtm *RoutineTaskManager) startPruningOrphanedImages(pause time.Duration) *util.Timer {
	return util.NewRunningTimer("orphanedImagePruning", pause, rtm.stop, false, func() {
		log.Debug("cleaning up orphaned images")
		// TODO write to a channel or something?
	})
}

func (rtm *RoutineTaskManager) startCheckingForUnknownImages(pause time.Duration) *util.Timer {
	return util.NewRunningTimer("unknownImageHandler", pause, rtm.stop, false, func() {
		log.Debug("handling images in Unknown status")
		select {
		case <-rtm.stop:
			return
		case rtm.unknownImagesCh <- true:
		}
	})
}

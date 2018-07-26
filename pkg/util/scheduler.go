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

package util

import (
	"time"
)

// Scheduler periodically executes `action`, with a pause of `delay` between
// invocations, and stops when receiving an event on `stop`.
type Scheduler struct {
	delay    time.Duration
	stop     <-chan struct{}
	setDelay chan time.Duration
	action   func()
}

// NewScheduler ...
func NewScheduler(delay time.Duration, stop <-chan struct{}, action func()) *Scheduler {
	scheduler := &Scheduler{delay: delay, stop: stop, setDelay: make(chan time.Duration), action: action}
	go scheduler.start()
	return scheduler
}

func (scheduler *Scheduler) start() {
	timer := time.NewTimer(scheduler.delay)
	for {
		select {
		case <-scheduler.stop:
			timer.Stop()
			return
		case <-timer.C:
			scheduler.action()
			timer = time.NewTimer(scheduler.delay)
		case delay := <-scheduler.setDelay:
			scheduler.delay = delay
		}
	}
}

// SetDelay sets the delay
func (scheduler *Scheduler) SetDelay(delay time.Duration) {
	scheduler.setDelay <- delay
}

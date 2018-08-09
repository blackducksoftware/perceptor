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
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// SchedulerState describes the state of a scheduler
type SchedulerState int

// .....
const (
	SchedulerStateReady         SchedulerState = iota
	SchedulerStateRunningAction SchedulerState = iota
	SchedulerStatePaused        SchedulerState = iota
	SchedulerStateStopped       SchedulerState = iota
)

// String .....
func (state SchedulerState) String() string {
	switch state {
	case SchedulerStateReady:
		return "SchedulerStateReady"
	case SchedulerStateRunningAction:
		return "SchedulerStateRunningAction"
	case SchedulerStatePaused:
		return "SchedulerStatePaused"
	case SchedulerStateStopped:
		return "SchedulerStateStopped"
	}
	panic(fmt.Errorf("invalid SchedulerState value: %d", state))
}

type resume struct {
	runImmediately bool
	err            chan error
}

// Scheduler periodically executes `action`, waiting `delay` between invocation starts.
// If `action` takes longer than `delay`, invocations will be dropped.
// It stops when receiving an event on `stop`.
// It's basically a time.Ticker with additional functionality for pausing and resuming.
type Scheduler struct {
	name   string
	state  SchedulerState
	delay  time.Duration
	action func()
	// channels
	pause    chan chan error
	resume   chan *resume
	stop     <-chan struct{}
	setDelay chan time.Duration
}

// NewRunningScheduler creates a new scheduler which is running
func NewRunningScheduler(name string, delay time.Duration, stop <-chan struct{}, runImmediately bool, action func()) *Scheduler {
	s := NewScheduler(name, delay, stop, action)
	err := s.Resume(runImmediately)
	if err != nil {
		// TODO somehow handle error?
		log.Errorf("scheduler %s: %s", name, err.Error())
	} else {
		log.Debugf("scheduler %s started scheduler successfully", name)
	}
	return s
}

// NewScheduler creates a new scheduler which is paused
func NewScheduler(name string, delay time.Duration, stop <-chan struct{}, action func()) *Scheduler {
	scheduler := &Scheduler{
		name:     name,
		state:    SchedulerStatePaused,
		delay:    delay,
		action:   action,
		pause:    make(chan chan error),
		resume:   make(chan *resume),
		stop:     stop,
		setDelay: make(chan time.Duration)}
	go scheduler.start()
	return scheduler
}

func (scheduler *Scheduler) start() {
	var timer *time.Ticker
	var c <-chan time.Time
	startTimer := func() {
		timer = time.NewTicker(scheduler.delay)
		c = timer.C
	}
	stopTimer := func() {
		timer.Stop()
		c = nil
	}
	didFinishAction := make(chan bool)
	var shouldPauseAfterRunningAction bool
	executeAction := func() {
		scheduler.state = SchedulerStateRunningAction
		shouldPauseAfterRunningAction = false
		go func() {
			scheduler.action()
			select {
			case didFinishAction <- true:
			case <-scheduler.stop:
			}
		}()
	}
	for {
		select {
		case <-didFinishAction:
			log.Debugf("scheduler %s: didFinishAction, state %s, shouldPause %t", scheduler.name, scheduler.state, shouldPauseAfterRunningAction)
			if shouldPauseAfterRunningAction {
				scheduler.state = SchedulerStatePaused
				stopTimer()
			} else {
				scheduler.state = SchedulerStateReady
			}
		case <-c:
			log.Debugf("scheduler %s: timer.C", scheduler.name)
			switch scheduler.state {
			case SchedulerStateReady:
				executeAction()
			case SchedulerStateRunningAction:
				log.Warnf("scheduler %s: backpressuring!  cannot run scheduler action, action already in progress", scheduler.name)
			default:
				log.Errorf("scheduler %s: cannot run action from state %s", scheduler.name, scheduler.state)
			}
		case ch := <-scheduler.pause:
			log.Debugf("scheduler %s: pause (state %s)", scheduler.name, scheduler.state)
			switch scheduler.state {
			case SchedulerStateReady:
				scheduler.state = SchedulerStatePaused
				stopTimer()
				ch <- nil
			case SchedulerStateRunningAction:
				if shouldPauseAfterRunningAction {
					ch <- fmt.Errorf("cannot pause scheduler %s: pause already queued up", scheduler.name)
					break
				}
				shouldPauseAfterRunningAction = true
				ch <- nil
			default:
				ch <- fmt.Errorf("cannot pause scheduler %s while in state %s", scheduler.name, scheduler.state.String())
			}
		case action := <-scheduler.resume:
			log.Debugf("scheduler %s: resume", scheduler.name)
			switch scheduler.state {
			case SchedulerStatePaused:
				action.err <- nil
				startTimer()
				if action.runImmediately {
					executeAction()
				} else {
					scheduler.state = SchedulerStateReady
				}
			default:
				action.err <- fmt.Errorf("cannot resume scheduler %s while in state %s", scheduler.name, scheduler.state.String())
			}
		case <-scheduler.stop:
			log.Debugf("scheduler %s: stop, state %s", scheduler.name, scheduler.state)
			switch scheduler.state {
			case SchedulerStateReady:
				stopTimer()
			case SchedulerStateStopped:
				// ??? not sure how this would happen
				log.Warnf("ignoring stop signal: scheduler %s already stopped", scheduler.name)
			}
			scheduler.state = SchedulerStateStopped
			return
		case delay := <-scheduler.setDelay:
			log.Debugf("scheduler %s: setDelay", scheduler.name)
			scheduler.delay = delay
		}
	}
}

// Pause temporarily stops the scheduler.
// It returns an error if the scheduler could not be paused.
func (scheduler *Scheduler) Pause() error {
	ch := make(chan error)
	scheduler.pause <- ch
	return <-ch
}

// Resume resumes the scheduler with the option of immediately running the action.
func (scheduler *Scheduler) Resume(runImmediately bool) error {
	action := &resume{runImmediately: runImmediately, err: make(chan error)}
	scheduler.resume <- action
	return <-action.err
}

// SetDelay sets the delay
func (scheduler *Scheduler) SetDelay(delay time.Duration) {
	scheduler.setDelay <- delay
}

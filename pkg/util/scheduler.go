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

// Scheduler periodically executes `action`, with a pause of `delay` between
// invocations, and stops when receiving an event on `stop`.
type Scheduler struct {
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
func NewRunningScheduler(delay time.Duration, stop <-chan struct{}, runImmediately bool, action func()) *Scheduler {
	s := NewScheduler(delay, stop, action)
	err := s.Resume(runImmediately)
	if err != nil {
		// TODO somehow handle error?
		log.Error(err.Error())
	} else {
		log.Debug("started scheduler successfully")
	}
	return s
}

// NewScheduler creates a new scheduler which is paused
func NewScheduler(delay time.Duration, stop <-chan struct{}, action func()) *Scheduler {
	scheduler := &Scheduler{
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
	runAction := make(chan bool)
	didFinishAction := make(chan bool)
	var nextState SchedulerState
	for {
		select {
		case <-c:
			go func() {
				runAction <- true
			}()
		case <-didFinishAction:
			log.Debugf("scheduler: didFinishAction")
			switch nextState {
			case SchedulerStateStopped:
				scheduler.state = SchedulerStateStopped
				stopTimer()
				return
			case SchedulerStatePaused:
				scheduler.state = SchedulerStatePaused
				stopTimer()
			default:
				scheduler.state = SchedulerStateReady
			}
		case <-runAction:
			log.Debugf("scheduler: runAction")
			switch scheduler.state {
			case SchedulerStateReady:
				// we're good
			case SchedulerStateRunningAction:
				log.Warnf("backpressuring!  cannot run scheduler action, action already in progress")
				break
			default:
				log.Errorf("cannot run action from state %s", scheduler.state)
				break
			}
			nextState = SchedulerStateReady
			go func() {
				scheduler.action()
				didFinishAction <- true
			}()
		case ch := <-scheduler.pause:
			log.Debugf("scheduler: pause (state %s)", scheduler.state)
			switch scheduler.state {
			case SchedulerStateReady:
				scheduler.state = SchedulerStatePaused
				stopTimer()
				ch <- nil
			case SchedulerStateRunningAction:
				if nextState == SchedulerStatePaused || nextState == SchedulerStateStopped {
					ch <- fmt.Errorf("cannot pause scheduler: %s already queued up", nextState)
					break
				}
				nextState = SchedulerStatePaused
				ch <- nil
			default:
				ch <- fmt.Errorf("cannot pause scheduler while in state %s", scheduler.state.String())
			}
		case action := <-scheduler.resume:
			log.Debugf("scheduler: resume")
			switch scheduler.state {
			case SchedulerStatePaused:
				scheduler.state = SchedulerStateReady
				action.err <- nil
				if action.runImmediately {
					go func() {
						runAction <- true
					}()
				}
				startTimer()
			default:
				action.err <- fmt.Errorf("cannot resume scheduler while in state %s", scheduler.state.String())
			}
		case <-scheduler.stop:
			log.Debugf("scheduler: stop")
			switch scheduler.state {
			case SchedulerStateReady:
				stopTimer()
			case SchedulerStateRunningAction:
				nextState = SchedulerStateStopped
			case SchedulerStatePaused:
				scheduler.state = SchedulerStateStopped
			case SchedulerStateStopped:
				// ??? not sure how this would happen
				log.Warnf("ignoring stop signal: scheduler already stopped")
			}
			return
		case delay := <-scheduler.setDelay:
			log.Debugf("scheduler: setDelay")
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

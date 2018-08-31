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

// TimerState describes the state of a timer
type TimerState int

// .....
const (
	TimerStateReady         TimerState = iota
	TimerStateRunningAction TimerState = iota
	TimerStatePaused        TimerState = iota
	TimerStateStopped       TimerState = iota
)

// String .....
func (state TimerState) String() string {
	switch state {
	case TimerStateReady:
		return "TimerStateReady"
	case TimerStateRunningAction:
		return "TimerStateRunningAction"
	case TimerStatePaused:
		return "TimerStatePaused"
	case TimerStateStopped:
		return "TimerStateStopped"
	}
	panic(fmt.Errorf("invalid TimerState value: %d", state))
}

type resume struct {
	runImmediately bool
	err            chan error
}

// Timer periodically executes `action`, waiting `delay` between invocation starts.
// If `action` takes longer than `delay`, invocations will be dropped.
// It stops when receiving an event on `stop`.
// It's basically a time.Ticker with additional functionality for pausing and resuming.
type Timer struct {
	name   string
	state  TimerState
	delay  time.Duration
	action func()
	// channels
	pause    chan chan error
	resume   chan *resume
	stop     <-chan struct{}
	setDelay chan time.Duration
}

// NewRunningTimer creates a new timer which is running
func NewRunningTimer(name string, delay time.Duration, stop <-chan struct{}, runImmediately bool, action func()) *Timer {
	s := NewTimer(name, delay, stop, action)
	err := s.Resume(runImmediately)
	if err != nil {
		// TODO somehow handle error?
		log.Errorf("timer %s: %s", name, err.Error())
	} else {
		log.Debugf("timer %s started timer successfully", name)
	}
	return s
}

// NewTimer creates a new timer which is paused
func NewTimer(name string, delay time.Duration, stop <-chan struct{}, action func()) *Timer {
	if delay <= 0 {
		panic(fmt.Errorf("invalid delay for timer %s: must be positive, was %s", name, delay))
	}
	timer := &Timer{
		name:     name,
		state:    TimerStatePaused,
		delay:    delay,
		action:   action,
		pause:    make(chan chan error),
		resume:   make(chan *resume),
		stop:     stop,
		setDelay: make(chan time.Duration)}
	go timer.start()
	return timer
}

func (timer *Timer) start() {
	var baseTimer *time.Ticker
	var c <-chan time.Time
	startTimer := func() {
		baseTimer = time.NewTicker(timer.delay)
		c = baseTimer.C
	}
	stopTimer := func() {
		baseTimer.Stop()
		c = nil
	}
	didFinishAction := make(chan bool)
	var shouldPauseAfterRunningAction bool
	executeAction := func() {
		timer.state = TimerStateRunningAction
		shouldPauseAfterRunningAction = false
		go func() {
			timer.action()
			select {
			case didFinishAction <- true:
			case <-timer.stop:
			}
		}()
	}
	for {
		select {
		case <-didFinishAction:
			//			log.Debugf("timer %s: didFinishAction, state %s, shouldPause %t", timer.name, timer.state, shouldPauseAfterRunningAction)
			if shouldPauseAfterRunningAction {
				timer.state = TimerStatePaused
				stopTimer()
			} else {
				timer.state = TimerStateReady
			}
		case <-c:
			//			log.Debugf("timer %s: timer.C", timer.name)
			switch timer.state {
			case TimerStateReady:
				executeAction()
			case TimerStateRunningAction:
				log.Warnf("timer %s: backpressuring!  cannot run timer action, action already in progress", timer.name)
			default:
				log.Errorf("timer %s: cannot run action from state %s", timer.name, timer.state)
			}
		case ch := <-timer.pause:
			//			log.Debugf("timer %s: pause (state %s)", timer.name, timer.state)
			switch timer.state {
			case TimerStateReady:
				timer.state = TimerStatePaused
				stopTimer()
				ch <- nil
			case TimerStateRunningAction:
				if shouldPauseAfterRunningAction {
					ch <- fmt.Errorf("cannot pause timer %s: pause already queued up", timer.name)
					break
				}
				shouldPauseAfterRunningAction = true
				ch <- nil
			default:
				ch <- fmt.Errorf("cannot pause timer %s while in state %s", timer.name, timer.state.String())
			}
		case action := <-timer.resume:
			//			log.Debugf("timer %s: resume", timer.name)
			switch timer.state {
			case TimerStatePaused:
				action.err <- nil
				startTimer()
				if action.runImmediately {
					executeAction()
				} else {
					timer.state = TimerStateReady
				}
			default:
				action.err <- fmt.Errorf("cannot resume timer %s while in state %s", timer.name, timer.state.String())
			}
		case <-timer.stop:
			//			log.Debugf("timer %s: stop, state %s", timer.name, timer.state)
			switch timer.state {
			case TimerStateReady:
				stopTimer()
			case TimerStateStopped:
				// ??? not sure how this would happen
				log.Warnf("ignoring stop signal: timer %s already stopped", timer.name)
			}
			timer.state = TimerStateStopped
			return
		case delay := <-timer.setDelay:
			//			log.Debugf("timer %s: setDelay", timer.name)
			timer.delay = delay
		}
	}
}

// Pause temporarily stops the timer.
// It returns an error if the timer could not be paused.
func (timer *Timer) Pause() error {
	ch := make(chan error)
	timer.pause <- ch
	return <-ch
}

// Resume resumes the timer with the option of immediately running the action.
func (timer *Timer) Resume(runImmediately bool) error {
	action := &resume{runImmediately: runImmediately, err: make(chan error)}
	timer.resume <- action
	return <-action.err
}

// SetDelay sets the delay
func (timer *Timer) SetDelay(delay time.Duration) {
	timer.setDelay <- delay
}

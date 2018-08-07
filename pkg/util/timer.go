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

	log "github.com/sirupsen/logrus"
)

// Timer is a thin wrapper around time.Timer, that provides:
// a concurrent-safe interface to have a cancellable timer,
// providing a single channel which sends a single value --
// namely, whether it was canceled or completed successfully.
type Timer struct {
	baseTimer *time.Timer
	done      chan bool
	stop      chan bool
}

// NewTimer instantiates a timer
func NewTimer(duration time.Duration) *Timer {
	t := &Timer{
		baseTimer: time.NewTimer(duration),
		done:      make(chan bool),
		stop:      make(chan bool)}
	go func() {
		select {
		case <-t.baseTimer.C:
			t.done <- true
		case <-t.stop:
			if !t.baseTimer.Stop() {
				<-t.baseTimer.C
			}
			t.done <- false
		}
		t.baseTimer = nil
		close(t.done)
		t.stop = nil
	}()
	return t
}

// Stop cancels the timer and returns true if the timer was not already canceled,
// and false otherwise.
func (t *Timer) Stop() bool {
	log.Debugf("timer: %t, %+v", t == nil, t)
	log.Debugf("timer.stop: %t, %+v", t.stop == nil, t.stop)
	select {
	case t.stop <- true:
		return true
	default:
		return false
	}
}

// Done returns a channel which sends a single value: true for normal completion,
// false for cancelation.
func (t *Timer) Done() <-chan bool {
	return t.done
}

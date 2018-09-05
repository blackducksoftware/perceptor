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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("Timer", func() {
	It("Pause before completion", func() {
		stop := make(chan struct{})
		defer close(stop)
		x := 0
		timer := NewRunningTimer("test1", 1*time.Second, stop, false, func() { x++ })
		time.Sleep(500 * time.Millisecond)
		err := timer.Pause()
		Expect(err).To(BeNil())
		Expect(x).To(Equal(0))
	})

	It("Pause after completion", func() {
		stop := make(chan struct{})
		defer close(stop)
		x := 0
		timer := NewRunningTimer("test2", 1*time.Second, stop, false, func() { x++ })
		time.Sleep(1500 * time.Millisecond)
		err := timer.Pause()
		Expect(err).To(BeNil())
		Expect(x).To(Equal(1))
	})

	It("Pause after stop", func() {
		stop := make(chan struct{})
		defer close(stop)
		x := 0
		timer := NewRunningTimer("test3", 1*time.Second, stop, false, func() { x++ })
		log.Debugf("started timer: %+v", timer)
		time.Sleep(1500 * time.Millisecond)
		Skip("Pausing after stopping is currently not supported")
	})

	It("Stop", func() {
		// TODO
	})

	It("Don't run immediately", func() {
		stop := make(chan struct{})
		defer close(stop)
		x := 0
		timer := NewRunningTimer("test4", 250*time.Millisecond, stop, false, func() { x++ })
		log.Debug("middle")
		time.Sleep(650 * time.Millisecond)
		err := timer.Pause()
		Expect(err).To(BeNil())
		Expect(x).To(Equal(2))
		log.Debug("end")
	})

	It("Run immediately", func() {
		stop := make(chan struct{})
		defer close(stop)
		x := 0
		timer := NewRunningTimer("test5", 250*time.Millisecond, stop, true, func() { x++ })
		log.Debug("middle")
		time.Sleep(650 * time.Millisecond)
		err := timer.Pause()
		Expect(err).To(BeNil())
		Expect(x).To(Equal(3))
		log.Debug("end")
	})

	It("Resume immediately", func() {
		stop := make(chan struct{})
		defer close(stop)
		x := 0
		y := 0
		useX := true
		timer := NewRunningTimer("test6", 500*time.Millisecond, stop, true, func() {
			if useX {
				x++
			} else {
				y++
			}
		})
		time.Sleep(250 * time.Millisecond)
		Expect(timer.Pause()).To(BeNil())
		Expect(x).To(Equal(1))
		Expect(y).To(Equal(0))
		useX = false
		Expect(timer.Resume(true)).To(BeNil())
		time.Sleep(750 * time.Millisecond)
		Expect(x).To(Equal(1))
		Expect(y).To(Equal(2))
	})

	It("state during action execution is 'running action'", func() {
		stop := make(chan struct{})
		defer close(stop)
		x := 0
		timer := NewRunningTimer("test7", 2*time.Second, stop, true, func() {
			time.Sleep(1 * time.Second)
			x++
		})
		time.Sleep(500 * time.Millisecond)
		Expect(timer.state).To(Equal(TimerStateRunningAction))
		Expect(x).To(Equal(0))
		time.Sleep(1 * time.Second)
		Expect(timer.state).To(Equal(TimerStateReady))
		Expect(x).To(Equal(1))
	})

	It("stops executing action after being stopped", func() {
		stop := make(chan struct{})
		x := 0
		timer := NewRunningTimer("test8", 500*time.Millisecond, stop, false, func() {
			log.Errorf("this shouldn't get executed")
			x++
		})
		time.Sleep(250 * time.Millisecond)
		close(stop)
		time.Sleep(10 * time.Millisecond)
		Expect(timer.state).To(Equal(TimerStateStopped))
		time.Sleep(2 * time.Second)
		Expect(x).To(Equal(0))
		Expect(timer.state).To(Equal(TimerStateStopped))
		// wait a bit longer to make sure it's not running
		time.Sleep(2 * time.Second)
		Expect(x).To(Equal(0))
		Expect(timer.state).To(Equal(TimerStateStopped))
	})

	It("can be stopped while running action -- action will complete, then it will stop", func() {
		stop := make(chan struct{})
		beforeSleep := 0
		afterSleep := 0
		timer := NewRunningTimer("test9", 2*time.Second, stop, true, func() {
			beforeSleep++
			time.Sleep(1 * time.Second)
			afterSleep++
		})
		time.Sleep(1 * time.Millisecond)
		Expect(timer.state).To(Equal(TimerStateRunningAction))
		Expect(beforeSleep).To(Equal(1))
		Expect(afterSleep).To(Equal(0))
		// now, cancel it while the action is running
		time.Sleep(500 * time.Millisecond)
		Expect(timer.state).To(Equal(TimerStateRunningAction))
		close(stop)
		Expect(timer.state).To(Equal(TimerStateRunningAction))
		// wait for the action to complete ... even though that shouldn't make a difference
		time.Sleep(750 * time.Millisecond)
		Expect(timer.state).To(Equal(TimerStateStopped))
		Expect(beforeSleep).To(Equal(1))
		Expect(afterSleep).To(Equal(1))
		// wait a bit longer to make sure it's not running
		time.Sleep(2 * time.Second)
		Expect(timer.state).To(Equal(TimerStateStopped))
		Expect(beforeSleep).To(Equal(1))
		Expect(afterSleep).To(Equal(1))
	})

	It("should still deallocate despite a long-running action", func() {
		stop := make(chan struct{})
		beforeSleep := 0
		afterSleep := 0
		timer := NewRunningTimer("test10", 1*time.Second, stop, true, func() {
			beforeSleep++
			time.Sleep(1 * time.Second)
			afterSleep++
		})
		time.Sleep(1 * time.Millisecond)
		Expect(timer.state).To(Equal(TimerStateRunningAction))
		Expect(beforeSleep).To(Equal(1))
		Expect(afterSleep).To(Equal(0))
		// now, cancel it while the action is running
		time.Sleep(250 * time.Millisecond)
		Expect(timer.state).To(Equal(TimerStateRunningAction))
		close(stop)
		time.Sleep(3 * time.Millisecond)
		Expect(timer.state).To(Equal(TimerStateStopped))
		// wait a bit longer to make sure it's not running
		time.Sleep(3 * time.Second)
		Expect(timer.state).To(Equal(TimerStateStopped))
		Expect(beforeSleep).To(Equal(1))
		Expect(afterSleep).To(Equal(1))
	})
})

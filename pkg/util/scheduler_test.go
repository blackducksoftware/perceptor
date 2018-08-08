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

var _ = Describe("Scheduler", func() {
	It("Pause before completion", func() {
		stop := make(chan struct{})
		defer close(stop)
		x := 0
		s := NewRunningScheduler(1*time.Second, stop, false, func() { x++ })
		time.Sleep(500 * time.Millisecond)
		err := s.Pause()
		Expect(err).To(BeNil())
		Expect(x).To(Equal(0))
	})

	It("Pause after completion", func() {
		stop := make(chan struct{})
		defer close(stop)
		x := 0
		s := NewRunningScheduler(1*time.Second, stop, false, func() { x++ })
		time.Sleep(1500 * time.Millisecond)
		err := s.Pause()
		Expect(err).To(BeNil())
		Expect(x).To(Equal(1))
	})

	It("Pause after stop", func() {
		stop := make(chan struct{})
		defer close(stop)
		x := 0
		s := NewRunningScheduler(1*time.Second, stop, false, func() { x++ })
		log.Debugf("started s: %+v", s)
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
		s := NewRunningScheduler(250*time.Millisecond, stop, false, func() { x++ })
		log.Debug("middle")
		time.Sleep(650 * time.Millisecond)
		err := s.Pause()
		Expect(err).To(BeNil())
		Expect(x).To(Equal(2))
		log.Debug("end")
	})

	It("Run immediately", func() {
		stop := make(chan struct{})
		defer close(stop)
		x := 0
		s := NewRunningScheduler(250*time.Millisecond, stop, true, func() { x++ })
		log.Debug("middle")
		time.Sleep(650 * time.Millisecond)
		err := s.Pause()
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
		s := NewRunningScheduler(500*time.Millisecond, stop, true, func() {
			if useX {
				x++
			} else {
				y++
			}
		})
		time.Sleep(250 * time.Millisecond)
		Expect(s.Pause()).To(BeNil())
		Expect(x).To(Equal(1))
		Expect(y).To(Equal(0))
		useX = false
		Expect(s.Resume(true)).To(BeNil())
		time.Sleep(750 * time.Millisecond)
		Expect(x).To(Equal(1))
		Expect(y).To(Equal(2))
	})
})

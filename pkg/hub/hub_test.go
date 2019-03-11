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

package hub

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func newClient(ignoreEvents bool) (*MockRawClient, *Hub) {
	rawClient := NewMockRawClient(false, []string{"a", "b", "c"})
	timings := &Timings{
		ScanCompletionPause:    125 * time.Millisecond,
		FetchUnknownScansPause: 250 * time.Millisecond,
		FetchAllScansPause:     500 * time.Millisecond,
		GetMetricsPause:        DefaultTimings.GetMetricsPause,
		LoginPause:             DefaultTimings.LoginPause,
		RefreshScanThreshold:   DefaultTimings.RefreshScanThreshold,
	}
	hub := NewHub("sysadmin", "password", "host1", 2, rawClient, timings)
	if ignoreEvents {
		go func() {
			updates := hub.Updates()
			for {
				<-updates
			}
		}()
	}
	return rawClient, hub
}

func getScanResults(hub *Hub) map[string]ScanStage {
	cls := map[string]ScanStage{}
	for key, val := range <-hub.ScanResults() {
		cls[key] = val.Stage
	}
	return cls
}

func RunClientTests() {
	Describe("Client", func() {
		It("should fetch initial code locations", func() {
			_, client := newClient(true)
			time.Sleep(1 * time.Second)
			Expect(getScanResults(client)).To(Equal(map[string]ScanStage{"a": ScanStageComplete, "b": ScanStageComplete, "c": ScanStageComplete}))
		})

		It("should add code locations as they're scanned", func() {
			_, client := newClient(true)
			time.Sleep(250 * time.Millisecond)
			Expect(<-client.InProgressScans()).To(Equal([]string{}))

			client.StartScanClient("abc")
			time.Sleep(250 * time.Millisecond)
			Expect(getScanResults(client)).To(Equal(map[string]ScanStage{"c": ScanStageComplete, "abc": ScanStageScanClient, "a": ScanStageComplete, "b": ScanStageComplete}))
			Expect(<-client.InProgressScans()).To(Equal([]string{"abc"}))

			client.FinishScanClient("abc", fmt.Errorf("planned failure"))
			time.Sleep(250 * time.Millisecond)
			Expect(getScanResults(client)).To(Equal(map[string]ScanStage{"c": ScanStageComplete, "abc": ScanStageFailure, "a": ScanStageComplete, "b": ScanStageComplete}))
			Expect(<-client.InProgressScans()).To(Equal([]string{}))

			// TODO is there a reasonable test case here?
			// rawClient.addCodeLocation("abc", ScanStageComplete)
			// time.Sleep(250 * time.Millisecond)
			// Expect(<-client.CodeLocations()).To(Equal(map[string]ScanStage{"c": ScanStageComplete, "abc": ScanStageComplete, "a": ScanStageComplete, "b": ScanStageComplete}))
			// Expect(<-client.InProgressScans()).To(Equal([]string{}))
		})
	})
}

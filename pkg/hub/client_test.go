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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func newClient() ClientInterface {
	rawClient := NewMockRawClient(false, []string{"a", "b", "c"})
	return NewClient("sysadmin", "password", "host1", rawClient, 250*time.Millisecond, 500*time.Millisecond)
}

func RunClientTests() {
	Describe("Client", func() {
		It("should fetch initial code locations", func() {
			client := newClient()
			time.Sleep(1 * time.Second)
			cls := <-client.CodeLocations()
			Expect(cls).To(Equal(map[string]ScanStage{"a": ScanStageComplete, "b": ScanStageComplete}))
		})

		It("should add code locations as they're scanned", func() {
			client := newClient()
			client.StartScanClient("abc")
			Expect(len(<-client.CodeLocations())).To(Equal(4))
			time.Sleep(1 * time.Second)
			client.FinishScanClient("abc")
			Expect(len(<-client.CodeLocations())).To(Equal(4))
		})
	})
}

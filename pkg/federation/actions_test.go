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

package federation

import (
	"time"

	"github.com/blackducksoftware/perceptor/pkg/hub"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func RunActionsTests() {
	Describe("API model", func() {
		It("should convert from model, including errors", func() {
			hub := hub.NewClient("username", "password", "host", 443, time.Second, time.Minute)
			time.Sleep(2 * time.Second)
			apiHub := <-hub.Model()
			Expect(len(apiHub.Errors)).NotTo(Equal(0))
			Expect(apiHub).NotTo(BeNil())
		})
	})
}

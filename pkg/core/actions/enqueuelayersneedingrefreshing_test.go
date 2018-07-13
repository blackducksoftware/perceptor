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

package actions

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func RunTestEnqueueLayersNeedingRefreshing() {
	Describe("enqueue layers needing refreshing", func() {
		It("should not enqueue layers that have recently been scanned", func() {
			actual := createNewModel1()
			(&EnqueueLayersNeedingRefreshing{30 * time.Second}).Apply(actual)
			Expect(actual.LayerRefreshQueue).To(Equal([]string{}))
		})

		It("should enqueue layers that have *not* recently been updated", func() {
			actual := createNewModel1()
			actual.Layers[layer1].TimeOfLastRefresh = time.Now().Add(-5 * time.Hour)
			(&EnqueueLayersNeedingRefreshing{30 * time.Second}).Apply(actual)
			Expect(actual.LayerRefreshQueue).To(Equal([]string{layer1}))
		})
	})
}
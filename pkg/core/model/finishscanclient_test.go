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

package model

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func RunTestFinishScanClient() {
	Describe("FinishScanClient", func() {
		It("handles failures", func() {
			model := NewModel()
			image := *NewImage("abc", "4.0", DockerImageSha("23bcf2dae3"))
			model.addImage(image, 0)
			model.setImageScanStatus(image.Sha, ScanStatusInQueue)
			model.setImageScanStatus(image.Sha, ScanStatusRunningScanClient)
			model.finishRunningScanClient(&image, fmt.Errorf("oops, unable to run scan client"))
			Expect(model.Images[image.Sha].ScanStatus).To(Equal(ScanStatusInQueue))
			Expect(*model.getNextImageFromScanQueue()).To(Equal(image))
		})
	})
}

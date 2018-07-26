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

package api

import (
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"
)

func RunMockResponderTests() {
	Describe("mock responder", func() {
		It("implements responder interface", func() {
			consumeResponder(NewMockResponder())
		})
		It("data", func() {
			mr := NewMockResponder()
			repo1 := "repo1"
			tag1 := "tag1"
			sha1 := "sha1"
			repo2 := "repo2"
			tag2 := "tag2"
			sha2 := "sha2"
			err := mr.AddImage(Image{Repository: repo1, Tag: tag1, Sha: sha1})
			Expect(err).To(BeNil())
			err = mr.AddPod(Pod{
				Containers: []Container{
					{
						Image: Image{Repository: repo1, Sha: sha1, Tag: tag1},
						Name:  "cont1",
					},
					{
						Image: Image{repo2, tag2, sha2},
						Name:  "cont2",
					},
				},
				Name:      "pod1",
				Namespace: "ns1",
				UID:       "uid1",
			})
			Expect(err).To(BeNil())
			scanResults := mr.GetScanResults()
			sort.Slice(scanResults.Images, func(i int, j int) bool {
				return scanResults.Images[i].Sha < scanResults.Images[j].Sha
			})
			Expect(scanResults.Images[0].Repository).To(Equal(repo1))
			Expect(scanResults.Images[0].Tag).To(Equal(tag1))
			Expect(scanResults.Images[0].Sha).To(Equal(sha1))
			Expect(scanResults.Images[1].Repository).To(Equal(repo2))
			Expect(scanResults.Images[1].Tag).To(Equal(tag2))
			Expect(scanResults.Images[1].Sha).To(Equal(sha2))
		})
	})
}

func consumeResponder(r Responder) {
	log.Infof("responder: %+v", r)
}

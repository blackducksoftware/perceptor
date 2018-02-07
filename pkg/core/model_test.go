/*
Copyright (C) 2018 Black Duck Software, Inc.

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

package core

import (
	"encoding/json"
	"testing"

	"github.com/blackducksoftware/perceptor/pkg/common"
	"github.com/prometheus/common/log"
)

func TestMarshalModel(t *testing.T) {
	model := Model{ConcurrentScanLimit: 1,
		ImageHubCheckQueue: []common.Image{common.Image{}},
		ImageScanQueue:     []common.Image{},
		Images:             map[common.Image]*ImageScanResults{},
		Pods:               map[string]common.Pod{}}
	jsonBytes, err := json.Marshal(model)
	jb, e := model.MarshalJSON()
	log.Infof("JSON: %s\n%v", string(jb), e)
	if err != nil {
		t.Errorf("unable to marshal %v as JSON: %v", model, err)
		t.Fail()
		return
	}
	expectedString := `{"Pods":{},"Images":{},"ImageScanQueue":[],"ImageHubCheckQueue":[{"Name":"","Sha":"","DockerImage":""}],"ConcurrentScanLimit":1}`
	if string(jsonBytes) != expectedString {
		t.Errorf("expected %s, got %s", expectedString, string(jsonBytes))
	}
}

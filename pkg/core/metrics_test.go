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
	"testing"

	"github.com/blackducksoftware/perceptor/pkg/common"
	log "github.com/sirupsen/logrus"
)

func TestMetrics(t *testing.T) {
	m := newMetrics()
	if m == nil {
		t.Error("expected m to be non-nil")
	}

	m.addImage(common.Image{})
	m.addPod(common.Pod{})
	m.allPods([]common.Pod{})
	m.deletePod("abcd")
	m.getNextImage()
	m.getScanResults()
	// TODO not good for testing
	// m.httpError(request, err)
	// m.httpNotFound(request)
	m.postFinishedScan()
	m.updateModel(Model{Images: map[common.Image]*ImageScanResults{
		common.Image{}: &ImageScanResults{ScanStatus: ScanStatusInQueue, ScanResults: nil},
	}})
	m.updatePod(common.Pod{})

	message := "finished test case"
	t.Log(message)
	log.Info(message)
}

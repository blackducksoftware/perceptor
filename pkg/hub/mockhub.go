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
	"math/rand"
	"strings"
	"time"
)

// need: mock hub, ?mock apiserver?

// MockHub is a mock implementation of ScanClientInterface .
type MockHub struct {
	inProgressImages []string
	finishedImages   map[string]int
}

func NewMockHub() *MockHub {
	hub := new(MockHub)
	hub.inProgressImages = []string{}
	hub.finishedImages = make(map[string]int)
	return hub
}

func (hub *MockHub) startRandomScanFinishing() {
	fmt.Println("starting!")
	for {
		time.Sleep(3 * time.Second)
		// TODO should lock the hub
		length := len(hub.inProgressImages)
		fmt.Println("in progress -- [", strings.Join(hub.inProgressImages, ", "), "]")
		if length <= 0 {
			continue
		}
		index := rand.Intn(length)
		image := hub.inProgressImages[index]
		fmt.Println("something finished --", image)
		hub.inProgressImages = append(hub.inProgressImages[:index], hub.inProgressImages[index+1:]...)
		hub.finishedImages[image] = 1
	}
}

func (hub *MockHub) FetchProjectByName(string) (*Project, error) {
	return nil, nil
}

func (hub *MockHub) FetchScanFromImage(image ImageInterface) (*ImageScan, error) {
	return nil, nil
}

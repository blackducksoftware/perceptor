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

	"github.com/blackducksoftware/perceptor/pkg/api"
)

// need: mock hub, ?mock apiserver?

// MockClient is a mock implementation of ClientInterface.
type MockClient struct {
	inProgressImages []string
	finishedImages   map[string]int
	hubVersion       string
}

// NewMockClient .....
func NewMockClient(hubVersion string) *MockClient {
	return &MockClient{
		inProgressImages: []string{},
		finishedImages:   map[string]int{},
		hubVersion:       hubVersion,
	}
}

func (hub *MockClient) startRandomScanFinishing() {
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

// DeleteScan ...
func (hub *MockClient) DeleteScan(scanName string) {
	//
}

// Version .....
func (hub *MockClient) Version() (string, error) {
	return hub.hubVersion, nil
}

// SetTimeout ...
func (hub *MockClient) SetTimeout(timeout time.Duration) {
	//
}

// Model ...
func (hub *MockClient) Model() <-chan *api.ModelHub {
	return nil
}

// ResetCircuitBreaker ...
func (hub *MockClient) ResetCircuitBreaker() {
	//
}

// IsEnabled ...
func (hub *MockClient) IsEnabled() <-chan bool {
	return make(<-chan bool)
}

// Host ...
func (hub *MockClient) Host() string {
	return "unimplemented"
}

// CodeLocationsCount ...
func (hub *MockClient) CodeLocationsCount() <-chan int {
	return nil
}

// StartScanClient ...
func (hub *MockClient) StartScanClient(scanName string) {
	//
}

// FinishScanClient ...
func (hub *MockClient) FinishScanClient(scanName string) {
	//
}

// ScanDidFinish ...
func (hub *MockClient) ScanDidFinish() <-chan *ScanResults {
	return nil
}

// DidFetchScanResults ...
func (hub *MockClient) DidFetchScanResults() <-chan *ScanResults {
	return nil
}

// InProgressScans ...
func (hub *MockClient) InProgressScans() <-chan []string {
	return nil
}

// Stop ...
func (hub *MockClient) Stop() {
	//
}

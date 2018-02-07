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

package clustermanager

import (
	"fmt"
	"time"

	"github.com/blackducksoftware/perceptor/pkg/common"
)

// MockClient implements the client interface without actually requiring
// a running cluster, in order to facilitate testing
type MockClient struct {
	pods        map[string]common.Pod
	annotations map[string]BlackDuckAnnotations
	addPod      chan AddPod
	updatePod   chan UpdatePod
	deletePod   chan DeletePod
}

func NewMockClient() *MockClient {
	client := MockClient{
		pods:        make(map[string]common.Pod),
		annotations: make(map[string]BlackDuckAnnotations),
		addPod:      make(chan AddPod),
		updatePod:   make(chan UpdatePod),
		deletePod:   make(chan DeletePod),
	}
	client.startPodUpdates()
	return &client
}

func (client *MockClient) startPodUpdates() {
	// newPodCounter := 1
	// updateCounter := 1
	go func() {
		for {
			time.Sleep(time.Second * 10)
			// newPod := common.Pod{
			// 	Name:      fmt.Sprintf("new-pod-%d", newPodCounter),
			// 	Namespace: "whatevs-namespace",
			// }
			// client.pods[newPod.GetKey()] = newPod
			// client.addPod <- AddPod{New: newPod}
		}
	}()
	go func() {
		for {
			time.Sleep(time.Second * 10)
			// index := rand.Intn(newPodCounter) + 1
			// namespace := "whatevs-namespace"
			// name := fmt.Sprintf("%s:new-pod-%d", namespace, index)
			// old, _ := client.pods[name]
			// // TODO this makes a completely new copy, right?
			// new := old
			// annotations := old.Annotations
			// newAnnotations := make(map[string]string)
			// for key, val := range annotations {
			// 	newAnnotations[key] = val
			// }
			// newAnnotations[fmt.Sprintf("some-key-%d:", updateCounter)] = "some-value"
			// new.Annotations = newAnnotations
			// update := UpdatePod{Old: old, New: new}
			// client.pods[name] = new
			// client.updatePod <- update
		}
	}()
	go func() {
		for {
			time.Sleep(time.Second * 50)
			// TODO delete a pod
		}
	}()
}

func (client *MockClient) ClearBlackDuckPodAnnotations(namespace string, name string) error {
	podKey := fmt.Sprintf("%s:%s", namespace, name)
	delete(client.annotations, podKey)
	return nil
}

func (client *MockClient) GetBlackDuckPodAnnotations(namespace string, name string) (*BlackDuckAnnotations, error) {
	podKey := fmt.Sprintf("%s:%s", namespace, name)
	annotations, _ := client.annotations[podKey]
	return &annotations, nil
}

func (client *MockClient) SetBlackDuckPodAnnotations(namespace string, name string, bdAnnotations BlackDuckAnnotations) error {
	podKey := fmt.Sprintf("%s:%s", namespace, name)
	client.annotations[podKey] = bdAnnotations
	return nil
}

func (client *MockClient) ClearBlackDuckPodAnnotationsWithPod(pod common.Pod) error {
	// podKey := pod.GetKey()
	// delete(client.annotations, podKey)
	return nil
}

func (client *MockClient) GetBlackDuckPodAnnotationsWithPod(pod common.Pod) (*BlackDuckAnnotations, error) {
	// annotations, _ := client.annotations[pod.GetKey()]
	// return &annotations, nil
	return nil, nil
}

func (client *MockClient) SetBlackDuckPodAnnotationsWithPod(pod common.Pod, bdAnnotations BlackDuckAnnotations) error {
	// podKey := pod.GetKey()
	// client.annotations[podKey] = bdAnnotations
	return nil
}

func (client *MockClient) PodAdd() <-chan AddPod {
	return client.addPod
}

func (client *MockClient) PodUpdate() <-chan UpdatePod {
	return client.updatePod
}

func (client *MockClient) PodDelete() <-chan DeletePod {
	return client.deletePod
}

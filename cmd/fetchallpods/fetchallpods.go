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

package main

import (
	"encoding/json"
	"fmt"

	"github.com/blackducksoftware/perceptor/pkg/clustermanager"
)

func main() {
	hubHost := "34.227.56.110.xip.io"
	masterURL := "https://" + hubHost + ":8443"
	kubeconfigPath := "/Users/mfenwick/.kube/config"
	client, err := clustermanager.NewKubeClient(masterURL, kubeconfigPath)
	if err != nil {
		panic(err)
	}
	pods, err := client.GetAllPods()
	if err != nil {
		panic(err)
	}
	jsonBytes, err := json.Marshal(pods)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", string(jsonBytes))
}

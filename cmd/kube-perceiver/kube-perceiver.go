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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	api "github.com/blackducksoftware/perceptor/pkg/api"
	clustermanager "github.com/blackducksoftware/perceptor/pkg/clustermanager"
	log "github.com/sirupsen/logrus"
)

// TODO metrics
// number of namespaces found
// number of pods per namespace
// number of images per pod
// number of occurrences of each pod
// number of successes, failures, of each perceptor endpoint
// ??? number of scan results fetched from perceptor

func main() {
	log.Info("started")

	podURL := fmt.Sprintf("%s:%s/%s", api.PerceptorBaseURL, api.PerceptorPort, api.PodPath)
	allPodsURL := fmt.Sprintf("%s:%s/%s", api.PerceptorBaseURL, api.PerceptorPort, api.AllPodsPath)
	scanResultsURL := fmt.Sprintf("%s:%s/%s", api.PerceptorBaseURL, api.PerceptorPort, api.ScanResultsPath)

	// 1. get kube client
	clusterClient, err := clustermanager.NewKubeClientFromCluster()
	if err != nil {
		log.Errorf("unable to instantiate kube client: %s", err.Error())
		panic(err)
	}

	// 2. send events from kube client into perceptor
	go func() {
		for {
			select {
			case addPod := <-clusterClient.PodAdd():
				log.Infof("cluster manager event -- add pod: UID %s, name %s", addPod.New.UID, addPod.New.QualifiedName())
				jsonBytes, err := json.Marshal(addPod.New)
				if err != nil {
					log.Errorf("unable to serialize pod: %s", err.Error())
					continue
				}
				resp, err := http.Post(podURL, "application/json", bytes.NewBuffer(jsonBytes))
				if err != nil {
					log.Errorf("unable to POST to %s: %s", podURL, err.Error())
					continue
				}
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					log.Infof("http POST request to %s succeeded", podURL)
				} else {
					log.Errorf("http POST request to %s failed with status code %d", podURL, resp.StatusCode)
				}
			case updatePod := <-clusterClient.PodUpdate():
				log.Infof("cluster manager event -- update pod: UID %s, name %s", updatePod.New.UID, updatePod.New.QualifiedName())
				jsonBytes, err := json.Marshal(updatePod.New)
				if err != nil {
					log.Errorf("unable to serialize pod: %s", err.Error())
					continue
				}
				req, err := http.NewRequest("PUT", podURL, bytes.NewBuffer(jsonBytes))
				if err != nil {
					log.Errorf("unable to create PUT request for %s: %s", podURL, err.Error())
					continue
				}
				req.Header.Set("Content-Type", "application/json")
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Errorf("unable to PUT to %s: %s", podURL, err.Error())
					continue
				}
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					log.Infof("http PUT request to %s succeeded", podURL)
				} else {
					log.Errorf("http PUT request to %s failed with status code %d", podURL, resp.StatusCode)
				}
			case deletePod := <-clusterClient.PodDelete():
				log.Infof("cluster manager event -- delete pod: qualified name %s", deletePod.QualifiedName)
				jsonBytes, err := json.Marshal(deletePod)
				if err != nil {
					log.Errorf("unable to serialize pod: %s", err.Error())
					continue
				}
				req, err := http.NewRequest("DELETE", podURL, bytes.NewBuffer(jsonBytes))
				if err != nil {
					log.Errorf("unable to create DELETE request for %s: %s", podURL, err.Error())
					continue
				}
				req.Header.Set("Content-Type", "application/json")
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Errorf("unable to DELETE to %s: %s", podURL, err.Error())
					continue
				}
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					log.Infof("http DELETE request to %s succeeded", podURL)
				} else {
					log.Errorf("http DELETE request to %s failed with status code %d", podURL, resp.StatusCode)
				}
			}
		}
	}()

	// 3. poll perceptor for vulns, translating those into annotations which
	//    get sent off to the kube apiserver
	go func() {
		for {
			time.Sleep(20 * time.Second)
			log.Infof("attempting to GET %s", scanResultsURL)
			resp, err := http.Get(scanResultsURL)
			if err != nil {
				log.Errorf("unable to GET %s: %s", scanResultsURL, err.Error())
				continue
			}
			defer resp.Body.Close()

			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Errorf("unable to read resp body from %s: %s", scanResultsURL, err.Error())
				continue
			}

			var scanResults api.ScanResults
			err = json.Unmarshal(bodyBytes, &scanResults)
			if err == nil && resp.StatusCode == 200 {
				log.Infof("GET to %s succeeded, about to update annotations", scanResultsURL)
				for _, pod := range scanResults.Pods {
					bdAnnotations := clustermanager.NewBlackDuckAnnotations(pod.PolicyViolations, pod.Vulnerabilities, pod.OverallStatus)
					clusterClient.SetBlackDuckPodAnnotations(pod.Namespace, pod.Name, *bdAnnotations)
				}
			} else {
				log.Errorf("unable to Unmarshal ScanResults from url %s: %s", scanResultsURL, err.Error())
			}
		}
	}()

	// 4. send over all pod information every <insert-time-period>.  This is a hack
	//    for when perceptor misses events -- either because it started after perceiver,
	//    or because it went down.
	go func() {
		duration := 20 * time.Second
		for {
			time.Sleep(duration)
			pods, err := clusterClient.GetAllPods()
			if err != nil {
				log.Errorf("unable to get all pods: %s", err.Error())
				continue
			}
			log.Infof("about to PUT all pods -- found %d pods", len(pods))

			jsonBytes, err := json.Marshal(api.NewAllPods(pods))
			if err != nil {
				log.Errorf("unable to serialize all pods: %s", err.Error())
				continue
			}

			req, err := http.NewRequest("PUT", allPodsURL, bytes.NewBuffer(jsonBytes))
			if err != nil {
				log.Errorf("unable to create PUT request for %s: %s", allPodsURL, err.Error())
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Errorf("unable to PUT to %s: %s", allPodsURL, err.Error())
				continue
			}
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				log.Infof("http POST request to %s succeeded", allPodsURL)
			} else {
				log.Errorf("http POST request to %s failed with status code %d", allPodsURL, resp.StatusCode)
			}
		}
	}()

	addr := fmt.Sprintf(":%s", api.PerceiverPort)
	http.ListenAndServe(addr, nil)
	log.Info("Http server started!")
}

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
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/blackducksoftware/perceptor/pkg/api"
	"github.com/blackducksoftware/perceptor/pkg/common"
	"github.com/blackducksoftware/perceptor/pkg/scanner"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// TODO metrics
// number of images scanned
// file size
// pull duration
// get duration
// scan client duration
// number of successes
// number of failures
// amount of time (or cycles?) idled
// number of times asked for a job and didn't get one

func main() {
	log.Info("started")

	// config, err := GetScannerConfig()
	// if err != nil {
	// 	log.Error("Failed to load configuration")
	// 	panic(err)
	// }
	config := ScannerConfig{
		HubHost:         "34.227.56.110.xip.io",
		HubUser:         "sysadmin",
		HubUserPassword: "blackduck"}

	scanClient, err := scanner.NewHubScanClient(config.HubHost, config.HubUser, config.HubUserPassword)
	if err != nil {
		log.Errorf("unable to instantiate hub scan client: %s", err.Error())
		panic(err)
	}

	imageScanStats := make(chan scanner.ScanClientJobResults)
	httpStats := make(chan scanner.HttpResult)

	go func() {
		for {
			time.Sleep(20 * time.Second)
			err := requestAndRunScanJob(scanClient, imageScanStats, httpStats)
			if err != nil {
				log.Errorf("error requesting or running scan job: %v", err)
			}
		}
	}()

	hostName, err := os.Hostname()
	if err != nil {
		log.Errorf("unable to get hostname: %s", err.Error())
		hostName = fmt.Sprintf("%d", rand.Int())
	}
	log.Infof("using hostName %s", hostName)
	http.Handle("/metrics", scanner.ScannerMetricsHandler(hostName, imageScanStats, httpStats))

	addr := fmt.Sprintf(":%s", api.PerceptorScannerPort)
	http.ListenAndServe(addr, nil)
	log.Info("Http server started!")
}

func requestAndRunScanJob(scanClient *scanner.HubScanClient, imageScanStats chan<- scanner.ScanClientJobResults, httpStats chan<- scanner.HttpResult) error {
	image := requestScanJob(httpStats)
	if image == nil {
		return nil
	}
	job := scanner.NewScanJob(*image)
	scanResults := scanClient.Scan(*job)
	imageScanStats <- scanResults
	errorString := ""
	if scanResults.Err != nil {
		errorString = scanResults.Err.Error()
	}
	finishedJob := api.FinishedScanClientJob{Err: errorString, Image: job.Image}
	log.Infof("about to finish job, going to send over %v", finishedJob)
	return finishScan(finishedJob, httpStats)
}

func requestScanJob(httpStats chan<- scanner.HttpResult) *common.Image {
	nextImageURL := fmt.Sprintf("%s:%s/%s", api.PerceptorBaseURL, api.PerceptorPort, api.NextImagePath)
	resp, err := http.Post(nextImageURL, "", bytes.NewBuffer([]byte{}))
	if resp != nil {
		httpStats <- scanner.HttpResult{Path: scanner.PathGetNextImage, StatusCode: resp.StatusCode}
	} else {
		// let's just assume this is due to something we did wrong
		httpStats <- scanner.HttpResult{Path: scanner.PathGetNextImage, StatusCode: 400}
	}
	if err != nil {
		log.Errorf("unable to POST to %s: %s", nextImageURL, err.Error())
		return nil
	} else if resp.StatusCode != 200 {
		log.Errorf("http POST request to %s failed with status code %d", nextImageURL, resp.StatusCode)
		return nil
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("unable to read response body from %s: %s", nextImageURL, err.Error())
		return nil
	}

	var nextImage api.NextImage
	err = json.Unmarshal(bodyBytes, &nextImage)
	if err != nil {
		log.Errorf("unmarshaling JSON body bytes %s failed for URL %s: %s", string(bodyBytes), nextImageURL, err.Error())
		return nil
	}

	imageName := "null"
	if nextImage.Image != nil {
		imageName = nextImage.Image.ShaName()
	}
	log.Infof("http POST request to %s succeeded, got image %s", nextImageURL, imageName)
	return nextImage.Image
}

func finishScan(results api.FinishedScanClientJob, httpStats chan<- scanner.HttpResult) error {
	finishedScanURL := fmt.Sprintf("%s:%s/%s", api.PerceptorBaseURL, api.PerceptorPort, api.FinishedScanPath)
	jsonBytes, err := json.Marshal(results)
	if err != nil {
		log.Errorf("unable to marshal json for finished job: %s", err.Error())
		return err
	}
	log.Infof("about to send over json text for finishing a job: %s", string(jsonBytes))
	// TODO change to exponential backoff or something ... but don't loop indefinitely in production
	for {
		resp, err := http.Post(finishedScanURL, "application/json", bytes.NewBuffer(jsonBytes))
		if resp != nil {
			httpStats <- scanner.HttpResult{Path: scanner.PathPostScanResults, StatusCode: resp.StatusCode}
		} else {
			// TODO so this 400 is actually a lie ... need to change it
			httpStats <- scanner.HttpResult{Path: scanner.PathPostScanResults, StatusCode: 400}
		}
		if err != nil {
			log.Errorf("unable to POST to %s: %s", finishedScanURL, err.Error())
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Errorf("POST to %s failed with status code %d", finishedScanURL, resp.StatusCode)
			continue
		}

		log.Infof("POST to %s succeeded", finishedScanURL)
		return nil
	}
}

// ScannerConfig contains all configuration for Perceptor
type ScannerConfig struct {
	HubHost         string
	HubUser         string
	HubUserPassword string
}

// GetScannerConfig returns a configuration object to configure Perceptor
func GetScannerConfig() (*ScannerConfig, error) {
	var cfg *ScannerConfig

	viper.SetConfigName("scanner_conf")
	viper.AddConfigPath("/etc/scanner_conf")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	err = viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}
	return cfg, nil
}

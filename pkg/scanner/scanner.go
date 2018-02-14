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

package scanner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/blackducksoftware/perceptor/pkg/api"
	log "github.com/sirupsen/logrus"
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

type Scanner struct {
	scanClient     ScanClientInterface
	httpClient     *http.Client
	imageScanStats chan ScanClientJobResults
	httpStats      chan HttpResult
}

func NewScanner(hubHost string, hubUser string, hubPassword string) (*Scanner, error) {
	os.Setenv("BD_HUB_PASSWORD", hubPassword)

	log.Infof("instantiating scanner with hub %s, user %s", hubHost, hubUser)

	scanClient, err := NewHubScanClient(hubHost, hubUser, hubPassword)
	if err != nil {
		log.Errorf("unable to instantiate hub scan client: %s", err.Error())
		return nil, err
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}

	scanner := Scanner{
		scanClient:     scanClient,
		httpClient:     httpClient,
		imageScanStats: make(chan ScanClientJobResults),
		httpStats:      make(chan HttpResult)}

	scanner.startRequestingScanJobs()

	return &scanner, nil
}

func (scanner *Scanner) ImageScanStats() <-chan ScanClientJobResults {
	return scanner.imageScanStats
}

func (scanner *Scanner) HttpStats() <-chan HttpResult {
	return scanner.httpStats
}

func (scanner *Scanner) startRequestingScanJobs() {
	log.Infof("starting to request scan jobs")
	go func() {
		for {
			time.Sleep(20 * time.Second)
			scanner.requestAndRunScanJob()
		}
	}()
}

func (scanner *Scanner) requestAndRunScanJob() {
	log.Info("requesting scan job")
	image, err := scanner.requestScanJob()
	if err != nil {
		log.Errorf("unable to request scan job: %s", err.Error())
		return
	}
	if image == nil {
		log.Info("requested scan job, got nil")
		return
	}
	job := NewScanJob(image.PullSpec, image.Sha, image.HubProjectName, image.HubProjectVersionName, image.HubScanName)
	scanResults := scanner.scanClient.Scan(*job)
	scanner.imageScanStats <- scanResults
	errorString := ""
	if scanResults.Err != nil {
		errorString = scanResults.Err.Error()
	}
	finishedJob := api.FinishedScanClientJob{Err: errorString, Sha: job.Sha}
	log.Infof("about to finish job, going to send over %v", finishedJob)
	err = scanner.finishScan(finishedJob)
	if err != nil {
		log.Errorf("unable to finish scan job: %s", err.Error())
	}
}

func (scanner *Scanner) requestScanJob() (*api.ImageSpec, error) {
	nextImageURL := fmt.Sprintf("%s:%s/%s", api.PerceptorBaseURL, api.PerceptorPort, api.NextImagePath)
	resp, err := scanner.httpClient.Post(nextImageURL, "", bytes.NewBuffer([]byte{}))
	if resp != nil {
		scanner.httpStats <- HttpResult{Path: PathGetNextImage, StatusCode: resp.StatusCode}
	} else {
		// let's just assume this is due to something we did wrong
		scanner.httpStats <- HttpResult{Path: PathGetNextImage, StatusCode: 500}
	}
	if err != nil {
		log.Errorf("unable to POST to %s: %s", nextImageURL, err.Error())
		return nil, err
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("http POST request to %s failed with status code %d", nextImageURL, resp.StatusCode)
		log.Error(err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("unable to read response body from %s: %s", nextImageURL, err.Error())
		return nil, err
	}

	var nextImage api.NextImage
	err = json.Unmarshal(bodyBytes, &nextImage)
	if err != nil {
		log.Errorf("unmarshaling JSON body bytes %s failed for URL %s: %s", string(bodyBytes), nextImageURL, err.Error())
		return nil, err
	}

	imageSha := "null"
	if nextImage.ImageSpec != nil {
		imageSha = nextImage.ImageSpec.Sha
	}
	log.Infof("http POST request to %s succeeded, got image %s", nextImageURL, imageSha)
	return nextImage.ImageSpec, nil
}

func (scanner *Scanner) finishScan(results api.FinishedScanClientJob) error {
	finishedScanURL := fmt.Sprintf("%s:%s/%s", api.PerceptorBaseURL, api.PerceptorPort, api.FinishedScanPath)
	jsonBytes, err := json.Marshal(results)
	if err != nil {
		log.Errorf("unable to marshal json for finished job: %s", err.Error())
		return err
	}
	log.Infof("about to send over json text for finishing a job: %s", string(jsonBytes))
	// TODO change to exponential backoff or something ... but don't loop indefinitely in production
	for {
		resp, err := scanner.httpClient.Post(finishedScanURL, "application/json", bytes.NewBuffer(jsonBytes))
		if resp != nil {
			scanner.httpStats <- HttpResult{Path: PathPostScanResults, StatusCode: resp.StatusCode}
		} else {
			// TODO this error may mean that we weren't even able to issue the request
			// ... so what should we use for the status code?
			scanner.httpStats <- HttpResult{Path: PathPostScanResults, StatusCode: 0}
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

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

package model

import (
	"fmt"

	"github.com/blackducksoftware/perceptor/pkg/api" // TODO I hate how this package depends on the api package
	"github.com/blackducksoftware/perceptor/pkg/hub"
	log "github.com/sirupsen/logrus"
)

// GetScanResults .....
type GetScanResults struct {
	Done chan api.ScanResults
}

// NewGetScanResults ...
func NewGetScanResults() *GetScanResults {
	return &GetScanResults{Done: make(chan api.ScanResults)}
}

// Apply .....
func (g *GetScanResults) Apply(model *Model) {
	scanResults := ScanResults(model)
	go func() {
		g.Done <- scanResults
	}()
}

func scanResultsForPod(model *Model, podName string) (*Scan, error) {
	pod, ok := model.Pods[podName]
	if !ok {
		return nil, fmt.Errorf("could not find pod of name %s in cache", podName)
	}

	overallStatus := hub.PolicyStatusTypeNotInViolation
	policyViolationCount := 0
	vulnerabilityCount := 0
	for _, container := range pod.Containers {
		imageScan, err := scanResultsForImage(model, container.Image.Sha)
		if err != nil {
			log.Errorf("unable to get scan results for image %s: %s", container.Image.Sha, err.Error())
			return nil, err
		}
		if imageScan == nil {
			return nil, nil
		}
		policyViolationCount += imageScan.PolicyViolations
		vulnerabilityCount += imageScan.Vulnerabilities
		imageScanOverallStatus := imageScan.OverallStatus
		if imageScanOverallStatus != hub.PolicyStatusTypeNotInViolation {
			overallStatus = imageScanOverallStatus
		}
	}
	podScan := &Scan{
		OverallStatus:    overallStatus,
		PolicyViolations: policyViolationCount,
		Vulnerabilities:  vulnerabilityCount}
	return podScan, nil
}

func scanResultsForImage(model *Model, sha DockerImageSha) (*Scan, error) {
	imageInfo, ok := model.Images[sha]
	if !ok {
		return nil, fmt.Errorf("could not find image of sha %s in cache", sha)
	}

	if imageInfo.ScanStatus != ScanStatusComplete {
		return nil, nil
	}
	if imageInfo.ScanResults == nil {
		return nil, fmt.Errorf("model inconsistency: could not find scan results for completed image %s", sha)
	}

	imageScan := &Scan{
		OverallStatus:    imageInfo.ScanResults.OverallStatus(),
		PolicyViolations: imageInfo.ScanResults.PolicyViolationCount(),
		Vulnerabilities:  imageInfo.ScanResults.VulnerabilityCount()}
	return imageScan, nil
}

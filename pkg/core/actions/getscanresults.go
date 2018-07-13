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

package actions

import (
	"github.com/blackducksoftware/perceptor/pkg/api"
	m "github.com/blackducksoftware/perceptor/pkg/core/model"
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
func (g *GetScanResults) Apply(model *m.Model) {
	scanResults := scanResults(model)
	go func() {
		g.Done <- scanResults
	}()
}

func scanResults(model *m.Model) api.ScanResults {
	// pods
	pods := []api.ScannedPod{}
	for podName, pod := range model.Pods {
		podScan, err := model.ScanResultsForPod(podName)
		if err != nil {
			log.Errorf("unable to retrieve scan results for Pod %s: %s", podName, err.Error())
			continue
		}
		if podScan == nil {
			log.Debugf("image scans not complete for pod %s, skipping (pod info: %+v)", podName, pod)
			continue
		}
		pods = append(pods, api.ScannedPod{
			Namespace:        pod.Namespace,
			Name:             pod.Name,
			PolicyViolations: podScan.PolicyViolations,
			Vulnerabilities:  podScan.Vulnerabilities,
			OverallStatus:    podScan.OverallStatus.String()})
	}

	// images
	images := []api.ScannedImage{}
	for sha, imageInfo := range model.Images {
		imageScan, err := model.ScanResultsForImage(sha)
		if err != nil {
			log.Errorf("unable to retrieve scan results for Pod %s: %s", sha, err.Error())
			continue
		}
		if imageScan == nil {
			log.Debugf("layer scans not complete for image %s, skipping", sha)
			continue
		}
		image := imageInfo.Image()
		apiImage := api.ScannedImage{
			Name:             image.HumanReadableName(),
			Sha:              string(image.Sha),
			PolicyViolations: imageScan.PolicyViolations,
			Vulnerabilities:  imageScan.Vulnerabilities,
			OverallStatus:    imageScan.OverallStatus.String(),
			ComponentsURL:    "TODO -- this is no longer possible to implement.  was imageInfo.ScanResults.ComponentsHref"}
		images = append(images, apiImage)
	}

	return *api.NewScanResults(model.HubVersion, model.HubVersion, pods, images)
}

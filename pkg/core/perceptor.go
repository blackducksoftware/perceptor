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

package core

import (
	"fmt"
	"net/http"
	"time"

	api "github.com/blackducksoftware/perceptor/pkg/api"
	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	log "github.com/sirupsen/logrus"
)

const (
	actionChannelSize = 100
)

// Perceptor ties together: a cluster, scan clients, and a hub.
// It listens to the cluster to learn about new pods.
// It keeps track of pods, containers, images, and scan results in a model.
// It has the hub scan images that have never been seen before.
// It grabs the scan results from the hub and adds them to its model.
// It publishes vulnerabilities that the cluster can find out about.
type Perceptor struct {
	model              *m.Model
	routineTaskManager *RoutineTaskManager
	scanScheduler      *ScanScheduler
	hubManager         HubManagerInterface
	// channels
	postConfig chan *api.PostConfig
	getModel   chan struct{}
}

// NewPerceptor creates a Perceptor using a real hub client.
func NewPerceptor(config *Config, hubManager HubManagerInterface) (*Perceptor, error) {
	model := m.NewModel()

	// 1. routine task manager
	stop := make(chan struct{})
	pruneOrphanedImagesPause := time.Duration(config.PruneOrphanedImagesPauseMinutes) * time.Minute
	routineTaskManager := NewRoutineTaskManager(stop, pruneOrphanedImagesPause, &Timings{})
	go func() {
		for {
			select {
			case <-stop:
				return
			case imageShas := <-routineTaskManager.OrphanedImages:
				// TODO reenable deletion with appropriate throttling
				// hubClient.DeleteScans(imageShas)
				log.Errorf("deletion temporarily disabled, ignoring shas %+v", imageShas)
			}
		}
	}()

	// 2. perceptor
	perceptor := &Perceptor{
		model:              model,
		routineTaskManager: routineTaskManager,
		scanScheduler:      &ScanScheduler{ConcurrentScanLimit: 2, CodeLocationLimit: 1000, HubManager: hubManager},
		hubManager:         hubManager,
		postConfig:         make(chan *api.PostConfig),
		getModel:           make(chan struct{}),
	}

	// 3. gather up model actions
	go func() {
		for {
			select {
			case <-stop:
				return
			case config := <-perceptor.postConfig:
				// TODO
				log.Warnf("unimplemented: %+v", config)
			case get := <-perceptor.getModel:
				// TODO
				log.Warnf("unimplemented: %+v", get)
				//case a := <-routineTaskManager.actions:
				// TODO
				// case isEnabled := <-hubClient.IsEnabled():
				// 	actions <- &model.SetIsHubEnabled{IsEnabled: isEnabled}
			}
		}
	}()

	// 4. http responder
	api.SetupHTTPServer(perceptor)

	// 5. done
	return perceptor, nil
}

// Section: api.Responder implementation

// GetModel .....
func (pcp *Perceptor) GetModel() api.Model {
	coreModel := pcp.model.GetModel()
	hubModels := map[string]*api.ModelHub{}
	for hubURL, hub := range pcp.hubManager.HubClients() {
		hubModels[hubURL] = hub.Model()
	}
	return api.Model{CoreModel: coreModel, Hubs: hubModels}
}

// PutHubs ...
func (pcp *Perceptor) PutHubs(hubs *api.PutHubs) {
	pcp.hubManager.SetHubs(hubs.HubURLs)
}

// AddPod .....
func (pcp *Perceptor) AddPod(apiPod api.Pod) error {
	recordAddPod()
	pod, err := APIPodToCorePod(apiPod)
	if err != nil {
		return err
	}
	pcp.model.AddPod(*pod)
	log.Debugf("handled add pod %s -- %s", pod.UID, pod.QualifiedName())
	return nil
}

// DeletePod .....
func (pcp *Perceptor) DeletePod(qualifiedName string) {
	recordDeletePod()
	pcp.model.DeletePod(qualifiedName)
	log.Debugf("handled delete pod %s", qualifiedName)
}

// UpdatePod .....
func (pcp *Perceptor) UpdatePod(apiPod api.Pod) error {
	recordUpdatePod()
	pod, err := APIPodToCorePod(apiPod)
	if err != nil {
		return err
	}
	pcp.model.UpdatePod(*pod)
	log.Debugf("handled update pod %s -- %s", pod.UID, pod.QualifiedName())
	return nil
}

// AddImage .....
func (pcp *Perceptor) AddImage(apiImage api.Image) error {
	recordAddImage()
	image, err := APIImageToCoreImage(apiImage)
	if err != nil {
		return err
	}
	pcp.model.AddImage(*image)
	log.Debugf("handled add image %s", image.PullSpec())
	return nil
}

// UpdateAllPods .....
func (pcp *Perceptor) UpdateAllPods(allPods api.AllPods) error {
	recordAllPods()
	pods := []m.Pod{}
	for _, apiPod := range allPods.Pods {
		pod, err := APIPodToCorePod(apiPod)
		if err != nil {
			return err
		}
		pods = append(pods, *pod)
	}
	pcp.model.SetPods(pods)
	log.Debugf("handled update all pods -- %d pods", len(allPods.Pods))
	return nil
}

// UpdateAllImages .....
func (pcp *Perceptor) UpdateAllImages(allImages api.AllImages) error {
	recordAllImages()
	images := []m.Image{}
	for _, apiImage := range allImages.Images {
		image, err := APIImageToCoreImage(apiImage)
		if err != nil {
			return err
		}
		images = append(images, *image)
	}
	pcp.model.SetImages(images)
	log.Debugf("handled update all images -- %d images", len(allImages.Images))
	return nil
}

// GetScanResults returns results for:
//  - all images that have a scan status of complete
//  - all pods for which all their images have a scan status of complete
func (pcp *Perceptor) GetScanResults() api.ScanResults {
	recordGetScanResults()
	return pcp.model.GetScanResults()
}

// GetNextImage .....
func (pcp *Perceptor) GetNextImage() api.NextImage {
	recordGetNextImage()
	log.Debugf("handled GET next image")
	image := pcp.model.GetNextImage()
	if image == nil {
		return *api.NewNextImage(nil)
	}
	hub := pcp.scanScheduler.AssignImage(image)
	if hub == nil {
		return *api.NewNextImage(nil)
	}
	imageSpec := &api.ImageSpec{
		Repository:            image.Repository,
		Tag:                   image.Tag,
		Sha:                   string(image.Sha),
		HubURL:                hub.Host(),
		HubProjectName:        image.HubProjectName(),
		HubProjectVersionName: image.HubProjectVersionName(),
		HubScanName:           image.HubScanName()}
	nextImage := *api.NewNextImage(imageSpec)
	log.Debugf("handled GET next image -- %s", image.PullSpec())
	return nextImage
}

// PostFinishScan .....
func (pcp *Perceptor) PostFinishScan(job api.FinishedScanClientJob) error {
	recordPostFinishedScan()
	var err error
	if job.Err == "" {
		err = nil
	} else {
		err = fmt.Errorf(job.Err)
	}
	// TODO also send to hub manager so that it can start polling the hub for this scan to complete
	image := m.NewImage(job.ImageSpec.Repository, job.ImageSpec.Tag, m.DockerImageSha(job.ImageSpec.Sha))
	pcp.model.FinishScanJob(image, err)
	log.Debugf("handled finished scan job -- %v", job)
	return nil
}

// internal use

// PostConfig .....
func (pcp *Perceptor) PostConfig(config *api.PostConfig) {
	pcp.postConfig <- config
	log.Debugf("handled post config -- %+v", config)
}

// PostCommand .....
func (pcp *Perceptor) PostCommand(command *api.PostCommand) {
	if command.ResetCircuitBreaker != nil {
		for _, hub := range pcp.hubManager.HubClients() {
			hub.ResetCircuitBreaker()
		}
	}
	log.Debugf("handled post command -- %+v", command)
}

// errors

// NotFound .....
func (pcp *Perceptor) NotFound(w http.ResponseWriter, r *http.Request) {
	log.Errorf("HTTPResponder not found from request %+v", r)
	recordHTTPNotFound(r)
	http.NotFound(w, r)
}

// Error .....
func (pcp *Perceptor) Error(w http.ResponseWriter, r *http.Request, err error, statusCode int) {
	log.Errorf("HTTPResponder error %s with code %d from request %+v", err.Error(), statusCode, r)
	recordHTTPError(r, err, statusCode)
	http.Error(w, err.Error(), statusCode)
}

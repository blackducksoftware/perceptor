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

	api "github.com/blackducksoftware/perceptor/pkg/api"
	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	"github.com/blackducksoftware/perceptor/pkg/hub"
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
	config             *Config
	// channels
	stop           <-chan struct{}
	getNextImageCh chan chan *api.ImageSpec
}

// NewPerceptor creates a Perceptor using a real hub client.
func NewPerceptor(config *Config, timings *Timings, scanScheduler *ScanScheduler, hubManager HubManagerInterface) (*Perceptor, error) {
	model := m.NewModel()

	// 1. routine task manager
	stop := make(chan struct{})
	routineTaskManager := NewRoutineTaskManager(stop, timings)
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-routineTaskManager.metricsCh:
				recordModelMetrics(model.GetMetrics())
			case <-routineTaskManager.unknownImagesCh:
				log.Debugf("handling RTM unknown images")
				/*
					if:
					 - any unknown scans
					 - all hubs up
					 - any scans missing from all hubs:
					move into scan queue
				*/
				unknownShas := model.GetImages(m.ScanStatusUnknown)
				log.Debugf("found %d unknown shas", len(unknownShas))
				if len(unknownShas) == 0 {
					break
				}
				isHubNotReady := false
				scans := map[string]*hub.Scan{}
				for _, hub := range hubManager.HubClients() {
					if !<-hub.HasFetchedScans() {
						isHubNotReady = true
						log.Debugf("found hub %s which is not ready", hub.Host())
						break
					}
					for scanName, results := range <-hub.ScanResults() {
						scans[scanName] = results
					}
				}
				// jbs, _ := json.MarshalIndent(scans, "", "  ")
				// fmt.Printf("%t\n%s\n", isHubNotReady, string(jbs))
				if isHubNotReady {
					log.Debugf("one or more hubs not ready")
					break
				}
				log.Debugf("about to change status of %d shas", len(unknownShas))
				for _, sha := range unknownShas {
					results, ok := scans[string(sha)]
					if ok {
						switch results.Stage {
						case hub.ScanStageComplete:
							model.ScanDidFinish(sha, results.ScanResults)
						case hub.ScanStageFailure:
							model.ScanDidFinish(sha, nil)
						default:
							log.Warnf("TODO: implement for other cases.  currently ignoring scan results for sha %s, %+v", sha, results)
							// case hub.ScanStageScanClient:
							// 	model.SetImageStatus(sha, m.ScanStatusRunningScanClient)
							// case hub.ScanStageHubScan:
							// 	model.SetImageStatus(sha, m.ScanStatusRunningHubScan)
						}
					} else {
						// didn't find the scan -> move it into the queue
						model.ScanDidFinish(sha, nil)
					}
				}
			}
		}
	}()
	go func() {
		updates := hubManager.Updates()
		for {
			select {
			case <-stop:
				return
			case update := <-updates:
				switch u := update.Update.(type) {
				case *hub.DidFindScan:
					model.ScanDidFinish(m.DockerImageSha(u.Name), u.Results)
				case *hub.DidFinishScan:
					model.ScanDidFinish(m.DockerImageSha(u.Name), u.Results)
				case *hub.DidRefreshScan:
					model.ScanDidFinish(m.DockerImageSha(u.Name), u.Results)
				}
			}
		}
	}()

	// 2. perceptor
	perceptor := &Perceptor{
		model:              model,
		routineTaskManager: routineTaskManager,
		scanScheduler:      scanScheduler,
		hubManager:         hubManager,
		config:             config,
		stop:               stop,
		getNextImageCh:     make(chan chan *api.ImageSpec),
	}

	go func() {
		for {
			select {
			case <-stop:
				return
			case ch := <-perceptor.getNextImageCh:
				perceptor.getNextImage(ch)
			}
		}
	}()

	// 3. done
	return perceptor, nil
}

// UpdateConfig ...
func (pcp *Perceptor) UpdateConfig(config *Config) {
	log.Infof("set config")
	pcp.hubManager.SetHubs(config.Hub.Hosts)
	logLevel, err := config.GetLogLevel()
	if err != nil {
		log.Errorf("unable to get log level: %s", err.Error())
	} else {
		log.SetLevel(logLevel)
	}
	// config.Timings
}

// Section: api.Responder implementation

// GetModel .....
func (pcp *Perceptor) GetModel() api.Model {
	coreModel := pcp.model.GetModel()
	hubModels := map[string]*api.ModelHub{}
	for hubURL, hub := range pcp.hubManager.HubClients() {
		hubModels[hubURL] = <-hub.Model()
	}
	return api.Model{
		CoreModel: coreModel,
		Hubs:      hubModels,
		Config:    pcp.config.model(),
		Scheduler: pcp.scanScheduler.model(),
	}
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
	go func() {
		pcp.model.SetImages(images)
	}()
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

func (pcp *Perceptor) getNextImage(ch chan<- *api.ImageSpec) {
	finish := func(spec *api.ImageSpec) {
		select {
		case <-pcp.stop:
		case ch <- spec:
		}
	}
	image := pcp.model.GetNextImage()
	if image == nil {
		log.Debug("get next image: no image found")
		finish(nil)
		return
	}
	hub := pcp.scanScheduler.AssignImage(image)
	if hub == nil {
		log.Debug("get next image: no available hub found")
		finish(nil)
		return
	}

	finish(&api.ImageSpec{
		Repository:            image.Repository,
		Tag:                   image.Tag,
		Sha:                   string(image.Sha),
		HubURL:                hub.Host(),
		HubProjectName:        image.HubProjectName(),
		HubProjectVersionName: image.HubProjectVersionName(),
		HubScanName:           image.HubScanName(),
		Priority:              image.Priority})
	log.Debugf("handle didStartScan")
	pcp.model.StartScanClient(image.Sha)
	pcp.hubManager.StartScanClient(hub.Host(), string(image.Sha))
}

// GetNextImage .....
func (pcp *Perceptor) GetNextImage() api.NextImage {
	recordGetNextImage()
	log.Debugf("handling GET next image")
	ch := make(chan *api.ImageSpec)
	pcp.getNextImageCh <- ch
	nextImage := *api.NewNextImage(<-ch)
	log.Debugf("handled GET next image -- %+v", nextImage)
	return nextImage
}

// PostFinishScan .....
func (pcp *Perceptor) PostFinishScan(job api.FinishedScanClientJob) error {
	recordPostFinishedScan()
	go func() {
		log.Debugf("handle didFinishScanClient")
		var scanErr error
		if job.Err != "" {
			scanErr = fmt.Errorf(job.Err)
		}
		err := pcp.hubManager.FinishScanClient(job.ImageSpec.HubURL, job.ImageSpec.HubScanName, scanErr)
		if err != nil {
			log.Errorf("unable to record FinishScanClient for hub %s, image %s:", job.ImageSpec.HubURL, job.ImageSpec.HubScanName)
		}
		image := m.NewImage(job.ImageSpec.Repository, job.ImageSpec.Tag, m.DockerImageSha(job.ImageSpec.Sha), job.ImageSpec.Priority)
		pcp.model.FinishScanJob(image, scanErr)
	}()
	log.Debugf("handled finished scan job -- %v", job)
	return nil
}

// internal use

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

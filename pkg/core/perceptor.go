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
	httpResponder      *HTTPResponder
	routineTaskManager *RoutineTaskManager
	//	hubs               map[string]*HubManager
	hubClientCreater HubClientCreaterInterface // func(hubURL string, user string, password string, port int) <-chan *hubCreationResult
}

// NewPerceptor creates a Perceptor using a real hub client.
func NewPerceptor(config *Config, hubClientCreater HubClientCreaterInterface) (*Perceptor, error) {
	model := m.NewModel()

	// 1. http responder
	httpResponder := NewHTTPResponder(model)
	api.SetupHTTPServer(httpResponder)

	// 2. routine task manager
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

	// 3. gather up model actions
	go func() {
		for {
			select {
			case a := <-httpResponder.PostNextImageChannel:
				// TODO
				log.Warnf("unimplemented: %+v", a)
			case config := <-httpResponder.PostConfigChannel:
				// TODO
				log.Warnf("unimplemented: %+v", config)
			case get := <-httpResponder.GetModelChannel:
				// TODO
				log.Warnf("unimplemented: %+v", get)
			//case a := <-routineTaskManager.actions:
			// TODO
			// case isEnabled := <-hubClient.IsEnabled():
			// 	actions <- &model.SetIsHubEnabled{IsEnabled: isEnabled}
			case <-httpResponder.ResetCircuitBreakerChannel:
				// TODO
			}
		}
	}()

	// 6. perceptor
	perceptor := Perceptor{
		httpResponder:      httpResponder,
		routineTaskManager: routineTaskManager,
		hubClientCreater:   hubClientCreater,
	}

	// 7. done
	return &perceptor, nil
}

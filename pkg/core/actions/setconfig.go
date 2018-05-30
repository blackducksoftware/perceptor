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
	"time"

	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	log "github.com/sirupsen/logrus"
)

// SetConfig .....
type SetConfig struct {
	ConcurrentScanLimit                 *int
	HubClientTimeoutMilliseconds        *int
	LogLevel                            *string
	ImageRefreshThresholdSeconds        *int
	EnqueueImagesForRefreshPauseSeconds *int
}

// Apply .....
func (s *SetConfig) Apply(model *m.Model) {
	if s.HubClientTimeoutMilliseconds != nil {
		model.Timings.HubClientTimeout = time.Duration(*s.HubClientTimeoutMilliseconds) * time.Millisecond
	}
	if s.ConcurrentScanLimit != nil {
		limit := *s.ConcurrentScanLimit
		if limit < 0 {
			log.Errorf("cannot set concurrent scan limit to less than 0 (got %d)", limit)
		} else {
			model.Config.ConcurrentScanLimit = limit
		}
	}
	if s.LogLevel != nil {
		logLevel, err := log.ParseLevel(*s.LogLevel)
		if err == nil {
			log.SetLevel(logLevel)
		} else {
			log.Errorf("invalid log level: %s", err.Error())
		}
	}
	if s.ImageRefreshThresholdSeconds != nil {
		model.Timings.RefreshThresholdDuration = time.Duration(*s.ImageRefreshThresholdSeconds) * time.Second
	}
	if s.EnqueueImagesForRefreshPauseSeconds != nil {
		seconds := time.Duration(*s.EnqueueImagesForRefreshPauseSeconds)
		model.Timings.EnqueueImagesForRefreshPause = seconds * time.Second
	}
}

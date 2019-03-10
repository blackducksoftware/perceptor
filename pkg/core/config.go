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
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/blackducksoftware/perceptor/pkg/api"
	log "github.com/sirupsen/logrus"
)

// Host configures the Black Duck hosts
type Host struct {
	Scheme              string
	Domain              string // it can be domain name or ip address
	Port                int
	User                string
	Password            string
	ConcurrentScanLimit int
}

// BlackDuckConfig handles BlackDuck-specific configuration
type BlackDuckConfig struct {
	ConnectionsEnvironmentVariableName string
	TLSVerification                    bool
}

// Timings stores all timings configuration that is used for various operations
type Timings struct {
	CheckForStalledScansPauseHours int
	StalledScanClientTimeoutHours  int
	ModelMetricsPauseSeconds       int
	UnknownImagePauseMilliseconds  int
	ClientTimeoutMilliseconds      int
}

// ClientTimeout returns the Black Duck client timeout
func (t *Timings) ClientTimeout() time.Duration {
	return time.Duration(t.ClientTimeoutMilliseconds) * time.Millisecond
}

// CheckForStalledScansPause returns an interval in hours to check the stalled scans
func (t *Timings) CheckForStalledScansPause() time.Duration {
	return time.Duration(t.CheckForStalledScansPauseHours) * time.Hour
}

// StalledScanClientTimeout returns client timeout in hours for the stalled scans
func (t *Timings) StalledScanClientTimeout() time.Duration {
	return time.Duration(t.StalledScanClientTimeoutHours) * time.Hour
}

// ModelMetricsPause returns an interval to pause the model metrics
func (t *Timings) ModelMetricsPause() time.Duration {
	return time.Duration(t.ModelMetricsPauseSeconds) * time.Second
}

// UnknownImagePause returns an interval in milliseconds to check for unknown images
func (t *Timings) UnknownImagePause() time.Duration {
	return time.Duration(t.UnknownImagePauseMilliseconds) * time.Millisecond
}

// PerceptorConfig stores the perceptor configuration
type PerceptorConfig struct {
	Timings     *Timings
	UseMockMode bool
	Port        int
}

// Config stores the input perceptor configuration
type Config struct {
	BlackDuck *BlackDuckConfig
	Perceptor *PerceptorConfig
	LogLevel  string
}

// getModelBlackDuckHosts will get the list of Black Duck hosts
func (config *Config) getModelBlackDuckHosts() ([]*api.ModelHost, error) {
	connectionStrings, ok := os.LookupEnv(config.BlackDuck.ConnectionsEnvironmentVariableName)
	if !ok {
		return nil, fmt.Errorf("cannot find Black Duck hosts: environment variable %s not found", config.BlackDuck.ConnectionsEnvironmentVariableName)
	}

	blackduckHosts := map[string]*api.ModelHost{}
	err := json.Unmarshal([]byte(connectionStrings), &blackduckHosts)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshall Black Duck hosts due to %+v", err)
	}

	hosts := []*api.ModelHost{}
	for _, host := range blackduckHosts {
		hosts = append(hosts, host)
	}

	return hosts, nil
}

// model will return the model configurations
func (config *Config) model() (*api.ModelConfig, error) {
	hosts, err := config.getModelBlackDuckHosts()
	if err != nil {
		return nil, err
	}
	return &api.ModelConfig{
		BlackDuck: &api.ModelBlackDuckConfig{
			Hosts:           hosts,
			ClientTimeout:   *api.NewModelTime(config.Perceptor.Timings.ClientTimeout()),
			TLSVerification: config.BlackDuck.TLSVerification,
		},
		LogLevel: config.LogLevel,
		Port:     config.Perceptor.Port,
		Timings: &api.ModelTimings{
			CheckForStalledScansPause: *api.NewModelTime(config.Perceptor.Timings.CheckForStalledScansPause()),
			ModelMetricsPause:         *api.NewModelTime(config.Perceptor.Timings.ModelMetricsPause()),
			StalledScanClientTimeout:  *api.NewModelTime(config.Perceptor.Timings.StalledScanClientTimeout()),
			UnknownImagePause:         *api.NewModelTime(config.Perceptor.Timings.UnknownImagePause()),
		},
	}, nil
}

// GetLogLevel returns the log level
func (config *Config) GetLogLevel() (log.Level, error) {
	return log.ParseLevel(config.LogLevel)
}

// dump will dump the perceptor configuration
func (config *Config) dump() (string, error) {
	bytes, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

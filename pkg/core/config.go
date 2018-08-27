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

	"github.com/fatih/structs"
	log "github.com/sirupsen/logrus"
)

// Config contains all configuration for Perceptor
type Config struct {
	HubHost                         string
	HubUser                         string
	HubUserPasswordEnvVar           string
	HubClientTimeoutMilliseconds    int
	HubPort                         int
	PruneOrphanedImagesPauseMinutes int
	ConcurrentScanLimit             int
	UseMockMode                     bool
	Port                            int
	LogLevel                        string
}

// HubClientTimeout converts the milliseconds to a duration
func (config *Config) HubClientTimeout() time.Duration {
	return time.Duration(config.HubClientTimeoutMilliseconds) * time.Millisecond
}

// GetLogLevel .....
func (config *Config) GetLogLevel() (log.Level, error) {
	return log.ParseLevel(config.LogLevel)
}

// The following does a couple of things:
// 	* Provide a map of sensible defaults for the struct
//	* Provide a map of values that can be loaded into viper so all structure
//		config keys are loaded. This allows viper to override these defaults
//		from either the configMap or environment variables
//
// If the struct above is modified, this method MUST also be modified
// so viper is aware of the new structure
func (config *Config) GetDefaults() map[string]interface{} {
	defaults := Config{
		HubHost:													"",
		HubUser:													"",
		HubUserPasswordEnvVar:						"",
		HubClientTimeoutMilliseconds:			5000,
		HubPort:													443,
		PruneOrphanedImagesPauseMinutes:	20,
		ConcurrentScanLimit:							1,
		UseMockMode:											false,
		Port:															0,
		LogLevel:													"warn",
	}

	return structs.Map(defaults)
}

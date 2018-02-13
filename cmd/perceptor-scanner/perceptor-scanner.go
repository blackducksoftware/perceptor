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

package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"

	"github.com/blackducksoftware/perceptor/pkg/api"
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

	config, err := GetScannerConfig()
	if err != nil {
		log.Errorf("Failed to load configuration: %s", err.Error())
		panic(err)
	}

	scannerManager, err := scanner.NewScanner(config.HubHost, config.HubUser, config.HubUserPassword)
	if err != nil {
		log.Errorf("unable to instantiate scanner: %s", err.Error())
		panic(err)
	}

	hostName, err := os.Hostname()
	if err != nil {
		log.Errorf("unable to get hostname: %s", err.Error())
		hostName = fmt.Sprintf("%d", rand.Int())
	}
	log.Infof("using hostName %s", hostName)
	http.Handle("/metrics", scanner.ScannerMetricsHandler(hostName, scannerManager.ImageScanStats(), scannerManager.HttpStats()))

	addr := fmt.Sprintf(":%s", api.PerceptorScannerPort)
	http.ListenAndServe(addr, nil)
	log.Info("Http server started!")
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

	viper.SetConfigName("perceptor_scanner_conf")
	viper.AddConfigPath("/etc/perceptor_scanner")

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

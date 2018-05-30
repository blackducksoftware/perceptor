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
	"os"

	"github.com/blackducksoftware/perceptor/pkg/api"
	// import just for the side-effect of changing how logrus works
	_ "github.com/blackducksoftware/perceptor/pkg/logging"
	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

// RunPerceptor .....
func RunPerceptor(configPath string) {
	log.Info("start")

	configManager := NewConfigManager(configPath)

	config, err := configManager.GetConfig()
	if err != nil {
		log.Errorf("Failed to load configuration: %s", err.Error())
		panic(err)
	}
	if config == nil {
		err = fmt.Errorf("expected non-nil config, but got nil")
		log.Errorf(err.Error())
		panic(err)
	}

	level, err := config.GetLogLevel()
	if err != nil {
		log.Errorf(err.Error())
		panic(err)
	}

	log.SetLevel(level)

	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())

	http.Handle("/metrics", prometheus.Handler())

	if config.UseMockMode {
		responder := api.NewMockResponder()
		api.SetupHTTPServer(responder)
		log.Info("instantiated responder in mock mode")
	} else {
		perceptor, err := NewPerceptor(config)
		if err != nil {
			log.Errorf("unable to instantiate percepter: %s", err.Error())
			panic(err)
		}

		log.Infof("instantiated perceptor in real mode: %+v", perceptor)
	}

	addr := fmt.Sprintf(":%d", config.Port)
	http.ListenAndServe(addr, nil)
	log.Info("Http server started!")
}

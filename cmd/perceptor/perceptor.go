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
	"os"
	"strconv"

	core "github.com/blackducksoftware/perceptor/pkg/core"
	log "github.com/sirupsen/logrus"
)

func main() {
	ignoreConfigMap, _ := strconv.ParseBool(os.Getenv("IGNORE_CONFIG_MAP"))
	if len(os.Args) == 1 && !ignoreConfigMap {
		log.Errorf("configPath not present and IGNORE_CONFIG_MAP is false")
		panic("configPath not present and IGNORE_CONFIG_MAP is false")
	}
	configPath := ""
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	log.Infof("Config path: %s", configPath)
	log.Infof("IGNORE_CONFIG_MAP env: %v", ignoreConfigMap)
	core.RunPerceptor(configPath, ignoreConfigMap)
}

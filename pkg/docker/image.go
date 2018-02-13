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

package docker

import (
	"fmt"
	"net/url"
)

type Image interface {
	DockerPullSpec() string
	DockerTarFilePath() string
}

func urlEncodedName(image Image) string {
	return url.QueryEscape(image.DockerPullSpec())
}

// createURL returns the URL used for hitting the docker daemon's create endpoint
func createURL(image Image) string {
	// TODO v1.24 refers to the docker version.  figure out how to avoid hard-coding this
	// TODO can probably use the docker api code for this
	return fmt.Sprintf("http://localhost/v1.24/images/create?fromImage=%s", urlEncodedName(image))
}

// getURL returns the URL used for hitting the docker daemon's get endpoint
func getURL(image Image) string {
	return fmt.Sprintf("http://localhost/v1.24/images/%s/get", urlEncodedName(image))
}

/*
Copyright (C) 2016 Black Duck Software, Inc.
http://www.blackducksoftware.com/

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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"

	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
	log "github.com/sirupsen/logrus"
)

func getHTTPRequestResponse(client *httputil.ClientConn, httpMethod string, requestURL string) (resp *http.Response, err error) {
	log.Infof("Making http request: [%s] [%s]\n", httpMethod, requestURL)
	req, err := http.NewRequest(httpMethod, requestURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("HTTP ERROR: received status != 200 on resp OK: %s", resp.Status))
	}
	return resp, nil
}

const (
	dockerSocketPath = "/var/run/docker.sock"
)

// PullImage gives us access to a docker image by:
//   1. hitting a docker create endpoint (?)
//   2. pulling down the newly created image and saving as a tarball
// It does this by accessing the host's docker daemon, locally, over the docker
// socket.  This gives us a window into any images that are local.
func PullImage(image common.Image) error {
	c, err := net.Dial("unix", dockerSocketPath)
	if err != nil {
		log.Errorf("unable to dial in to docker.sock: %s", err.Error())
		return err
	}
	defer c.Close()
	client := httputil.NewClientConn(c, nil)
	defer client.Close()

	err = createImageInLocalDocker(image)
	if err != nil {
		log.Errorf("unable to create image in local docker: %s", err.Error())
		return err
	}

	log.Infof("Processing image: %s", image.Name)

	err = saveImageToTar(client, image)
	if err != nil {
		log.Errorf("Error while making tar file: %s", err)
		return err
	}

	log.Infof("Ready to scan %s %s", image.Name, image.TarFilePath())
	return nil
}

// createImageInLocalDocker could also be implemented using curl:
// this example hits ... ? the default registry?  docker hub?
//   curl --unix-socket /var/run/docker.sock -X POST http://localhost/images/create?fromImage=alpine
// this example hits the kipp registry:
//   curl --unix-socket /var/run/docker.sock -X POST http://localhost/images/create\?fromImage\=registry.kipp.blackducksoftware.com%2Fblackducksoftware%2Fhub-jobrunner%3A4.5.0
// ... or could it?  Not really sure what this does.
func createImageInLocalDocker(image common.Image) (err error) {
	fd := func(proto, addr string) (conn net.Conn, err error) {
		return net.Dial("unix", "/var/run/docker.sock")
	}
	tr := &http.Transport{
		Dial: fd,
	}
	client := &http.Client{Transport: tr}
	imageURL := image.CreateURL()
	log.Infof("Attempting to create %s ......", imageURL)
	resp, err := client.Post(imageURL, "", nil)
	defer resp.Body.Close()

	if resp.StatusCode == 200 && err == nil {
		log.Infof("Create succeeded for %s %v", imageURL, resp)
	} else if err == nil {
		// This should get hit if there's a 404
		log.Infof("Create may have failed for %s: status code %d, response", imageURL, resp.StatusCode, resp)
	} else {
		log.Errorf("Create failed for %s , ERROR = ((  %s  )) ", imageURL, err)
	}
	return err
}

// saveImageToTar: part of what it does is to issue an http request similar to the following:
//   curl --unix-socket /var/run/docker.sock -X GET http://localhost/images/openshift%2Forigin-docker-registry%3Av3.6.1/get
func saveImageToTar(client *httputil.ClientConn, image common.Image) error {
	imageURL := image.GetURL()
	resp, err := getHTTPRequestResponse(client, "GET", imageURL)
	if err != nil {
		return err
	}

	body := resp.Body
	defer func() {
		body.Close()
	}()
	log.Info("Starting to write file contents to a tar file.")
	tarFilePath := image.TarFilePath()
	log.Infof("Tar File Path: %s", tarFilePath)

	// just need to create `./tmp` if it doesn't already exist
	os.Mkdir("./tmp", 0755)

	f, err := os.OpenFile(tarFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Errorf("Error opening file: %s", err.Error())
		return err
	}
	if _, err := io.Copy(f, body); err != nil {
		log.Errorf("Error copying file: %s", err.Error())
		return err
	}
	return nil
}

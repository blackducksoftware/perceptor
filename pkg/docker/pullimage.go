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
func PullImage(image Image) error {
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

	log.Infof("Processing image: %s", image.directory())

	err = saveImageToTar(client, image)
	if err != nil {
		log.Errorf("Error while making tar file: %s", err)
		return err
	}

	log.Infof("Ready to scan %s %s", image.name, image.tarFilePath())
	return nil
}

// createImageInLocalDocker could also be implemented using curl:
//   --unix-socket /var/run/docker.sock -X POST http://localhost/images/create?fromImage=alpine
// ... or could it?  Not really sure what this does.
func createImageInLocalDocker(image Image) (err error) {
	fd := func(proto, addr string) (conn net.Conn, err error) {
		return net.Dial("unix", "/var/run/docker.sock")
	}
	tr := &http.Transport{
		Dial: fd,
	}
	client := &http.Client{Transport: tr}
	imageURL := image.createURL()
	log.Infof("Attempting to create %s ......", imageURL)
	resp, err := client.Post(imageURL, "", nil)
	defer resp.Body.Close()

	if resp.StatusCode == 200 && err == nil {
		log.Infof("Create succeeded for %s %v", imageURL, resp)
	} else {
		log.Errorf("Create failed for %s , ERROR = ((  %s  )) ", imageURL, err)
	}
	return err
}

func saveImageToTar(client *httputil.ClientConn, image Image) error {
	imageURL := image.getURL()
	resp, err := getHTTPRequestResponse(client, "GET", imageURL)
	if err != nil {
		return err
	}

	body := resp.Body
	defer func() {
		body.Close()
	}()
	os.MkdirAll(image.path(), 0755)
	log.Info("Starting to write file contents to a tar file.")
	tarFilePath := image.tarFilePath()
	log.Infof("Tar File Path: %s", tarFilePath)
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

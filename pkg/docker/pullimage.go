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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	dockerSocketPath = "/var/run/docker.sock"
)

type ImagePuller struct {
	rootTarballDir string
	client         *http.Client
}

func NewImagePuller() *ImagePuller {
	fd := func(proto, addr string) (conn net.Conn, err error) {
		return net.Dial("unix", dockerSocketPath)
	}
	tr := &http.Transport{Dial: fd}
	client := &http.Client{Transport: tr}
	return &ImagePuller{rootTarballDir: "./tmp", client: client}
}

// PullImage gives us access to a docker image by:
//   1. hitting a docker create endpoint (?)
//   2. pulling down the newly created image and saving as a tarball
// It does this by accessing the host's docker daemon, locally, over the docker
// socket.  This gives us a window into any images that are local.
func (ip *ImagePuller) PullImage(image Image) ImagePullStats {
	stats := ImagePullStats{}
	start := time.Now()

	createDuration, err := ip.createImageInLocalDocker(image)
	if createDuration != nil {
		stats.CreateDuration = createDuration
	}
	if err != nil {
		stats.Err = &ImagePullError{Code: ErrorTypeUnableToCreateImage, RootCause: err}
		return stats
	}
	log.Infof("Processing image: %s", image.DockerPullSpec())

	startSave := time.Now()
	fileSize, pullError := ip.saveImageToTar(image)
	saveDuration := time.Now().Sub(startSave)
	stats.SaveDuration = &saveDuration
	if pullError != nil {
		log.Errorf("save image %+v to tar failed: %s", image, pullError.Error())
		stats.Err = pullError
		return stats
	}

	stop := time.Now()

	log.Infof("Ready to scan image %s at path %s", image.DockerPullSpec(), image.DockerTarFilePath())
	duration := stop.Sub(start)
	stats.TotalDuration = &duration
	stats.TarFileSizeMBs = fileSize
	return stats
}

// createImageInLocalDocker could also be implemented using curl:
// this example hits ... ? the default registry?  docker hub?
//   curl --unix-socket /var/run/docker.sock -X POST http://localhost/images/create?fromImage=alpine
// this example hits the kipp registry:
//   curl --unix-socket /var/run/docker.sock -X POST http://localhost/images/create\?fromImage\=registry.kipp.blackducksoftware.com%2Fblackducksoftware%2Fhub-jobrunner%3A4.5.0
//
func (ip *ImagePuller) createImageInLocalDocker(image Image) (*time.Duration, error) {
	start := time.Now()
	imageURL := createURL(image)
	log.Infof("Attempting to create %s ......", imageURL)
	resp, err := ip.client.Post(imageURL, "", nil)
	if err != nil {
		log.Errorf("Create failed for image %s: %s", imageURL, err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		message := fmt.Sprintf("Create may have failed for %s: status code %d, response %+v", imageURL, resp.StatusCode, resp)
		log.Errorf(message)
		return nil, errors.New(message)
	}

	_, err = ioutil.ReadAll(resp.Body)
	duration := time.Now().Sub(start)
	return &duration, err
}

// saveImageToTar: part of what it does is to issue an http request similar to the following:
//   curl --unix-socket /var/run/docker.sock -X GET http://localhost/images/openshift%2Forigin-docker-registry%3Av3.6.1/get
func (ip *ImagePuller) saveImageToTar(image Image) (*int, *ImagePullError) {
	url := getURL(image)
	log.Infof("Making http request: [%s]", url)
	resp, err := ip.client.Get(url)
	if err != nil {
		return nil, &ImagePullError{Code: ErrorTypeUnableToGetImage, RootCause: err}
	} else if resp.StatusCode != http.StatusOK {
		return nil, &ImagePullError{
			Code:      ErrorTypeBadStatusCodeFromGetImage,
			RootCause: fmt.Errorf("HTTP ERROR: received status != 200 from %s: %s", url, resp.Status)}
	}

	log.Infof("GET request for %s successful", url)

	body := resp.Body
	defer func() {
		body.Close()
	}()
	tarFilePath := image.DockerTarFilePath()
	log.Infof("Starting to write file contents to tar file %s", tarFilePath)

	// just need to create `./tmp` if it doesn't already exist
	os.Mkdir(ip.rootTarballDir, 0755)

	f, err := os.OpenFile(tarFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return nil, &ImagePullError{Code: ErrorTypeUnableToCreateTarFile, RootCause: err}
	}
	if _, err = io.Copy(f, body); err != nil {
		return nil, &ImagePullError{Code: ErrorTypeUnableToCopyTarFile, RootCause: err}
	}

	// What's the right way to get the size of the file?
	//  1. resp.ContentLength
	//  2. check the size of the file after it's written
	// fileSizeInMBs := int(resp.ContentLength / (1024 * 1024))
	stats, err := os.Stat(tarFilePath)

	if err != nil {
		return nil, &ImagePullError{Code: ErrorTypeUnableToGetFileStats, RootCause: err}
	}

	fileSizeInMBs := int(stats.Size() / (1024 * 1024))

	return &fileSizeInMBs, nil
}

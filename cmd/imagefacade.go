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

// Executable that caches images in a directory as tarballs.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
)

const (
	DEFAULT_BDS_SCANNER_BASE_DIR  = "/tmp/ocp-scanner"
)

type input struct {
	fromImage       string
	tag string
}

var in input

func init() {
	// fromImage=busyBox or fromImage=85fioh87h998hojR8h98hf.  
	flag.StringVar(&in.fromImage, "fromImage", "busybox123", "imageDigest or name Will have .tar at the end.")
	flag.StringVar(&in.tag, "tag", "", "tag, empty is ok.")
}

func writeContents(body io.ReadCloser, path string) (tarFilePath string, err error) {
	defer func() {
		body.Close()
	}()
	fmt.Printf(fmt.Sprintf("Starting to write file contents to a tar file."))
	tarFilePath = fmt.Sprintf("%s.%s", path, "tar")
	log.Printf("Tar File Path: %s\n", tarFilePath)
	f, err := os.OpenFile(tarFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		fmt.Println("ERROR : opening file.")
		fmt.Println(err)
		return "", err
	}
	if _, err := io.Copy(f, body); err != nil {
		fmt.Println("ERROR : copying into file.")
		fmt.Println(err)
		return "", err
	}
	return tarFilePath, nil
}

func getHttpRequestResponse(client *httputil.ClientConn, httpMethod string, requestUrl string) (resp *http.Response, err error) {
	log.Printf("Making request: [%s] [%s]\n", httpMethod, requestUrl)
	req, err := http.NewRequest(httpMethod, requestUrl, nil)
	if err != nil {
		return nil, err
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("received status != 200 on resp OK: %s", resp.Status))
	}
	return resp, nil
}


// func pullImage(img string, tag string) (err error) {
	

// }

func saveImageToTar(client *httputil.ClientConn, image string, path string) (tarFilePath string, err error) {
	exists, err := imageExists(client, image)

	if err != nil {
		return "", err
	}
	if !exists {
		return "", nil
	}
	os.MkdirAll(path, 0755)
	imageUrl := fmt.Sprintf("/images/%s/get", image)
	resp, err := getHttpRequestResponse(client, "GET", imageUrl)
	if err != nil {
		return "", err
	}
	return writeContents(resp.Body, path)
}

func imageExists(client *httputil.ClientConn, image string) (result bool, err error) {
	imageUrl := fmt.Sprintf("/images/%s/history", image)
	resp, err := getHttpRequestResponse(client, "GET", imageUrl)
	if err != nil {
		log.Printf("Error testing for image presence\n%s\n", err.Error())
		return false, err
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		log.Printf("Error reading image history\n%s\n", err.Error())
		return false, err
	} else if len(buf.Bytes()) == 0 {
		log.Printf("No data in image history\n")
	}
	log.Printf("image found: %s",imageUrl)
	return true, nil
}

func main() {

	flag.Parse() // Scan the arguments list

	if in.fromImage == "" {
		panic("Need -image=...")
	}

	// Accessing the host's docker daemon, locally, over the docker socket.
	// This gives us a window into any images that are local.
	c, err := net.Dial("unix", "/var/run/docker.sock")
	if err != nil {
		panic(err)
	}
	defer c.Close()
	client := httputil.NewClientConn(c, nil)
	defer client.Close()

	fmt.Printf(fmt.Sprintf("Processing image: %s with engine ID %s\n", in.fromImage))


	img_dir_name := strings.Replace(in.fromImage, ":", "_", -1)
	img_dir_name = strings.Replace(img_dir_name, "/", "_", -1)
	path := fmt.Sprintf("%s/%s", "/tmp/", img_dir_name)

	if strings.Contains(path, "<none>") {
		fmt.Printf(fmt.Sprintf("WARNING: Image : %s won't be scanned.", in.fromImage))
	} else {
		newTarPath, err := saveImageToTar(client, in.fromImage, path)
		if err != nil {
			log.Printf(fmt.Sprintf("Error while making tar file: %s\n", err))
		} else {
			log.Printf("Ready to scan !!!!! %s %s", in.fromImage, newTarPath)
		}
		// Need to possibly dump images peridodically... 
	}
}

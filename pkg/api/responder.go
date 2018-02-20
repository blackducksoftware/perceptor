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

package api

import (
	"net/http"
)

type Responder interface {
	GetModel() string

	// perceiver
	AddPod(pod Pod)
	UpdatePod(pod Pod)
	DeletePod(qualifiedName string)
	GetScanResults() ScanResults
	AddImage(image Image)
	UpdateAllPods(allPods AllPods)
	UpdateAllImages(allImages AllImages)

	// scanner
	GetNextImage() NextImage
	PostFinishScan(job FinishedScanClientJob)

	// internal use
	SetConcurrentScanLimit(limit SetConcurrentScanLimit)

	// errors
	NotFound(w http.ResponseWriter, r *http.Request)
	Error(w http.ResponseWriter, r *http.Request, err error, statusCode int)
}

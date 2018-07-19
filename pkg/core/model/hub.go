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

package model

import "fmt"

// Hub ...
type Hub struct {
	URL             string
	Images          map[DockerImageSha]bool
	InProgressScans map[DockerImageSha]bool
}

// NewHub ...
func NewHub(url string) *Hub {
	return &Hub{URL: url, Images: map[DockerImageSha]bool{}, InProgressScans: map[DockerImageSha]bool{}}
}

// InProgressScanCount ...
func (h *Hub) InProgressScanCount() int {
	return len(h.InProgressScans)
}

// AddImage ...
func (h *Hub) AddImage(sha DockerImageSha) error {
	if _, ok := h.Images[sha]; ok {
		return fmt.Errorf("image %s already present", sha)
	}
	h.Images[sha] = true
	return nil
}

// RemoveImage ...
func (h *Hub) RemoveImage(sha DockerImageSha) error {
	if _, ok := h.Images[sha]; !ok {
		return fmt.Errorf("image %s not present", sha)
	}
	delete(h.Images, sha)
	// we don't care if 'sha' is in InProgressScans or not
	delete(h.InProgressScans, sha)
	return nil
}

// StartScanningImage ...
func (h *Hub) StartScanningImage(sha DockerImageSha) error {
	if _, ok := h.Images[sha]; !ok {
		return fmt.Errorf("image %s not found", sha)
	}
	if _, ok := h.InProgressScans[sha]; ok {
		return fmt.Errorf("image %s already in progress", sha)
	}
	h.InProgressScans[sha] = true
	return nil
}

// ScanDidFinish ...
func (h *Hub) ScanDidFinish(sha DockerImageSha) error {
	if _, ok := h.Images[sha]; !ok {
		return fmt.Errorf("image %s not found", sha)
	}
	if _, ok := h.InProgressScans[sha]; !ok {
		return fmt.Errorf("image %s not in progress", sha)
	}
	delete(h.InProgressScans, sha)
	return nil
}

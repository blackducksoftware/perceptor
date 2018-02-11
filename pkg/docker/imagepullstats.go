/*
Copyright (C) 2018 Black Duck Software, Inc.

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
	"time"
)

type ImagePullStats struct {
	CreateDuration *time.Duration
	SaveDuration   *time.Duration
	TotalDuration  *time.Duration
	TarFileSizeMBs *int
	Err            *ImagePullError
}

func (img *ImagePullStats) Summary() string {
	return fmt.Sprintf("[image pull stats : create %v save %v total_duration %v sizeMB %v (Error %v) ]",
		img.CreateDuration, img.SaveDuration, img.TotalDuration, img.TarFileSizeMBs, img.Err)

}

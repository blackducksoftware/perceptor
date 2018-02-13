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

import "fmt"

type ErrorType int

const (
	ErrorTypeUnableToCreateImage       ErrorType = iota
	ErrorTypeUnableToGetImage          ErrorType = iota
	ErrorTypeBadStatusCodeFromGetImage ErrorType = iota
	ErrorTypeUnableToCreateTarFile     ErrorType = iota
	ErrorTypeUnableToCopyTarFile       ErrorType = iota
	ErrorTypeUnableToGetFileStats      ErrorType = iota
)

func (et ErrorType) String() string {
	switch et {
	case ErrorTypeUnableToCreateImage:
		return "unable to create image in local docker"
	case ErrorTypeUnableToGetImage:
		return "unable to get image"
	case ErrorTypeBadStatusCodeFromGetImage:
		return "bad status code from GET image"
	case ErrorTypeUnableToCreateTarFile:
		return "Error opening file"
	case ErrorTypeUnableToCopyTarFile:
		return "Error copying file"
	case ErrorTypeUnableToGetFileStats:
		return "Error getting file stats"
	}
	panic(fmt.Errorf("invalid ErrorType value: %d", et))
}

type ImagePullError struct {
	Code      ErrorType
	RootCause error
}

func (ipe *ImagePullError) String() string {
	return fmt.Sprintf("%s: %s", ipe.Code.String(), ipe.RootCause.Error())
}

func (ipe ImagePullError) Error() string {
	return ipe.String()
}

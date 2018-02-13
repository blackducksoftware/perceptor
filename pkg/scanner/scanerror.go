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

package scanner

import (
	"fmt"
)

type ErrorType int

const (
	ErrorTypeUnableToPullDockerImage = iota
	ErrorTypeFailedToRunJavaScanner  = iota
)

func (et ErrorType) String() string {
	switch et {
	case ErrorTypeUnableToPullDockerImage:
		return "unable to pull docker image"
	case ErrorTypeFailedToRunJavaScanner:
		return "failed to run java scanner"
	}
	panic(fmt.Errorf("invalid ErrorType value: %d", et))
}

type ScanError struct {
	Code      ErrorType
	RootCause error
}

func (se *ScanError) String() string {
	return fmt.Sprintf("%s: %s", se.Code.String(), se.RootCause.Error())
}

func (se ScanError) Error() string {
	return se.String()
}

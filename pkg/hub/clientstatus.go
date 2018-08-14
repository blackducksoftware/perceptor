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

package hub

import (
	"fmt"
)

// ClientStatus describes the state of a hub client
type ClientStatus int

// .....
const (
	ClientStatusError ClientStatus = iota
	ClientStatusUp    ClientStatus = iota
	ClientStatusDown  ClientStatus = iota
)

// String .....
func (status ClientStatus) String() string {
	switch status {
	case ClientStatusError:
		return "ClientStatusError"
	case ClientStatusUp:
		return "ClientStatusUp"
	case ClientStatusDown:
		return "ClientStatusDown"
	}
	panic(fmt.Errorf("invalid ClientStatus value: %d", status))
}

// MarshalJSON .....
func (status ClientStatus) MarshalJSON() ([]byte, error) {
	jsonString := fmt.Sprintf(`"%s"`, status.String())
	return []byte(jsonString), nil
}

// MarshalText .....
func (status ClientStatus) MarshalText() (text []byte, err error) {
	return []byte(status.String()), nil
}

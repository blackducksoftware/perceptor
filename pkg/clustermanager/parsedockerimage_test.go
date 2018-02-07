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

package clustermanager

import (
	"testing"
)

func TestParseImageID(t *testing.T) {
	name, sha, err := ParseImageIDString("docker-pullable://registry.kipp.blackducksoftware.com/blackducksoftware/hub-registration@sha256:cb4983d8399a59bb5ee6e68b6177d878966a8fe41abe18a45c3b1d8809f1d043")
	//	name, sha, err := ParseImageIDString("docker-pullable://r.k/h@sha256:cb4983d8399a59bb5ee6e68b6177d878966a8fe41abe18a45c3b1d8809f1d043")
	if err != nil {
		t.Errorf("expected no error, found %s", err.Error())
		t.Fail()
	}
	if name != "registry.kipp.blackducksoftware.com/blackducksoftware/hub-registration" {
		t.Errorf("incorrect name, got %s", name)
		t.Fail()
	}
	if sha != "cb4983d8399a59bb5ee6e68b6177d878966a8fe41abe18a45c3b1d8809f1d043" {
		t.Errorf("incorrect sha, got %s", sha)
		t.Fail()
	}
}

func TestParseImageIDFail(t *testing.T) {
	name, tag, err := ParseImageIDString("abc")
	if err == nil {
		t.Errorf("expected error, found nil")
		t.Fail()
	}
	if err.Error() != "could not find prefix <docker-pullable://> in <abc>" {
		t.Errorf("incorrect error message: %s", err.Error())
		t.Fail()
	}
	if name != "" {
		t.Errorf("incorrect name: %s", name)
		t.Fail()
	}
	if tag != "" {
		t.Errorf("incorrect tag %s", tag)
		t.Fail()
	}
}

func TestParseImageIDFailMissingSha(t *testing.T) {
	name, tag, err := ParseImageIDString("docker-pullable://abc")
	if err == nil {
		t.Errorf("expected error, found nil")
		t.Fail()
	}
	if err.Error() != "unable to match digestRegexp regex <@sha256:([a-zA-Z0-9]+)$> to input <abc>" {
		t.Errorf("incorrect error message: %s", err.Error())
		t.Fail()
	}
	if name != "" {
		t.Errorf("incorrect name: %s", name)
		t.Fail()
	}
	if tag != "" {
		t.Errorf("incorrect tag %s", tag)
		t.Fail()
	}
}

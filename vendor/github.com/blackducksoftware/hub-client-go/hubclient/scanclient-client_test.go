// Copyright 2018 Synopsys, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hubclient

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func TestDownloadScanClient(t *testing.T) {
	client, err := NewWithSession("https://localhost", HubClientDebugTimings, 5*time.Second)
	if err != nil {
		t.Error(err)
		return
	}
	err = client.Login("sysadmin", "blackduck")
	if err != nil {
		t.Error(err)
		return
	}

	err = client.DownloadScanClientLinux("/tmp/scanclient-linux.tar.gz")
	if err != nil {
		t.Error(err)
		return
	}

	err = client.DownloadScanClientMac("/tmp/scanclient-mac.tar.gz")
	if err != nil {
		t.Error(err)
		return
	}

	err = client.DownloadScanClientWindows("/tmp/scanclient-windows.tar.gz")
	if err != nil {
		t.Error(err)
		return
	}

	log.Info("successfully downloaded scan clients")
	t.Logf("successfully downloaded scan clients")
}

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
	"fmt"
	"io"
	"net/http"
	"os"
)

func (c *Client) DownloadScanClientMac(path string) error {
	return c.downloadScanClientHelper(path, "download/scan.cli-macosx.zip")
}

func (c *Client) DownloadScanClientLinux(path string) error {
	return c.downloadScanClientHelper(path, "download/scan.cli.zip")
}

func (c *Client) DownloadScanClientWindows(path string) error {
	return c.downloadScanClientHelper(path, "download/scan.cli-windows.zip")
}

func (c *Client) downloadScanClientHelper(path string, urlPath string) error {

	scanClientURL := fmt.Sprintf("%s/%s", c.baseURL, urlPath)

	resp, err := c.httpClient.Get(scanClientURL)
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("GET failed: received status != 200 from %s: %s", scanClientURL, resp.Status)
		return err
	}

	body := resp.Body
	defer func() {
		body.Close()
	}()

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	if _, err = io.Copy(f, body); err != nil {
		return err
	}

	return nil
}

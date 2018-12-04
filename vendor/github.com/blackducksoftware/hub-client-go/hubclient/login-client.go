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
	"net/url"

	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
)

func (c *Client) Login(username string, password string) error {

	loginURL := fmt.Sprintf("%s/j_spring_security_check", c.baseURL)
	formValues := url.Values{
		"j_username": {username},
		"j_password": {password},
	}

	resp, err := c.httpClient.PostForm(loginURL, formValues)

	if err != nil {
		return errors.Annotate(err, "Error trying to login via form login")
	}

	if resp.StatusCode != 204 {
		return errors.Errorf("got a %d response instead of a 204", resp.StatusCode)
	}

	if csrf := resp.Header.Get(HeaderNameCsrfToken); csrf != "" {
		c.haveCsrfToken = true
		c.csrfToken = csrf
	}

	log.Debugln("Login: Successfully authenticated")

	return nil
}

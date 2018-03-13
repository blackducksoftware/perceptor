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
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"

	log "github.com/sirupsen/logrus"
)

type HubClientDebug uint16

const (
	HubClientDebugTimings HubClientDebug = 1 << iota
	HubClientDebugContent
)

// Client will need to support CSRF tokens for session-based auth for Hub 4.1.x (or was it 4.0?)
type Client struct {
	httpClient    *http.Client
	baseURL       string
	authToken     string
	useAuthToken  bool
	haveCsrfToken bool
	csrfToken     string
	debugFlags    HubClientDebug
}

func NewWithSession(baseURL string, debugFlags HubClientDebug, timeout time.Duration) (*Client, error) {

	jar, err := cookiejar.New(nil) // Look more at this function

	if err != nil {
		return nil, err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Jar:       jar,
		Transport: tr,
		Timeout:   timeout,
	}

	return &Client{
		httpClient:   client,
		baseURL:      baseURL,
		useAuthToken: false,
		debugFlags:   debugFlags,
	}, nil
}

func NewWithToken(baseURL string, authToken string, debugFlags HubClientDebug, timeout time.Duration) (*Client, error) {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}

	return &Client{
		httpClient:   client,
		baseURL:      baseURL,
		authToken:    authToken,
		useAuthToken: true,
		debugFlags:   debugFlags,
	}, nil
}

func readBytes(readCloser io.ReadCloser) ([]byte, error) {

	defer readCloser.Close()
	buf := new(bytes.Buffer)

	if _, err := buf.ReadFrom(readCloser); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func validateHTTPResponse(resp *http.Response, expectedStatusCode int) error {

	if resp.StatusCode != expectedStatusCode { // Should this be a list at some point?
		log.Errorf("Got a %d response instead of a %d.", resp.StatusCode, expectedStatusCode)
		readResponseBody(resp)
		return fmt.Errorf("got a %d response instead of a %d", resp.StatusCode, expectedStatusCode)
	}

	return nil
}

func (c *Client) processResponse(resp *http.Response, result interface{}, expectedStatusCode int) error {

	var bodyBytes []byte
	var err error

	if err := validateHTTPResponse(resp, expectedStatusCode); err != nil {
		log.Errorf("Error validating HTTP Response: %+v.", err)
		return err
	}

	if result == nil {
		// Don't have a result to deserialize to, skip it
		return nil
	}

	if bodyBytes, err = readBytes(resp.Body); err != nil {
		log.Errorf("Error reading HTTP Response: %+v.", err)
		return err
	}

	if c.debugFlags&HubClientDebugContent != 0 {
		log.Debugf("START DEBUG: --------------------------------------------------------------------------- \n\n")
		log.Debugf("TEXT OF RESPONSE: \n %s", string(bodyBytes[:]))
		log.Debugf("END DEBUG: --------------------------------------------------------------------------- \n\n\n\n")
	}

	if err := json.Unmarshal(bodyBytes, result); err != nil {
		log.Errorf("Error parsing HTTP Response: %+v.", err)
		log.Errorln("\n\n--------------------")
		log.Errorln("Response:")
		log.Errorln("--------------------\n\n")
		log.Errorln(resp)
		return err
	}

	return nil
}

func (c *Client) httpGetJSON(url string, result interface{}, expectedStatusCode int) error {

	// TODO: Content type?

	var resp *http.Response
	var err error

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP STARTING GET REQUEST: %s", url)
	}

	httpStart := time.Now()
	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		log.Errorf("Error making http get request: %+v.", err)
		return err
	}

	c.doPreRequest(req)

	if resp, err = c.httpClient.Do(req); err != nil {
		log.Errorf("Error getting HTTP Response: %+v.", err)
		return err
	}

	httpElapsed := time.Since(httpStart)

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP GET ELAPSED TIME: %d ms.   -- Request: %s", (httpElapsed / 1000 / 1000), url)
	}

	return c.processResponse(resp, result, expectedStatusCode)
}

func (c *Client) httpPutJSON(url string, data interface{}, contentType string, expectedStatusCode int) error {

	var resp *http.Response
	var err error

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP STARTING PUT REQUEST: %s", url)
	}

	// Encode json
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	if err := enc.Encode(&data); err != nil {
		log.Errorf("Error encoding json: %+v.", err)
	}

	httpStart := time.Now()
	req, err := http.NewRequest(http.MethodPut, url, &buf)
	req.Header.Set(HeaderNameContentType, contentType)

	if err != nil {
		log.Errorf("Error making http put request: %+v.", err)
		return err
	}

	c.doPreRequest(req)
	log.Debugf("PUT Request: %+v.", req)

	if resp, err = c.httpClient.Do(req); err != nil {
		log.Errorf("Error getting HTTP Response: %+v.", err)
		readResponseBody(resp)
		return err
	}

	httpElapsed := time.Since(httpStart)

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP PUT ELAPSED TIME: %d ms.   -- Request: %s", (httpElapsed / 1000 / 1000), url)
	}

	return c.processResponse(resp, nil, expectedStatusCode) // TODO: Maybe need a response too?
}

func (c *Client) httpPostJSON(url string, data interface{}, contentType string, expectedStatusCode int) (string, error) {

	var resp *http.Response
	var err error

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP STARTING POST REQUEST: %s", url)
	}

	// Encode json
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	if err := enc.Encode(&data); err != nil {
		log.Errorf("Error encoding json: %+v.", err)
	}

	httpStart := time.Now()
	req, err := http.NewRequest(http.MethodPost, url, &buf)
	req.Header.Set(HeaderNameContentType, contentType)

	if err != nil {
		log.Errorf("Error making http post request: %+v.", err)
		return "", err
	}

	c.doPreRequest(req)
	log.Debugf("POST Request: %+v.", req)

	if resp, err = c.httpClient.Do(req); err != nil {
		log.Errorf("Error getting HTTP Response: %+v.", err)
		readResponseBody(resp)
		return "", err
	}

	httpElapsed := time.Since(httpStart)

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP POST ELAPSED TIME: %d ms.   -- Request: %s", (httpElapsed / 1000 / 1000), url)
	}

	if err := c.processResponse(resp, nil, expectedStatusCode); err != nil {
		return "", err
	}

	return resp.Header.Get("Location"), nil
}

func (c *Client) httpDelete(url string, contentType string, expectedStatusCode int) error {

	var resp *http.Response
	var err error

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP STARTING DELETE REQUEST: %s", url)
	}

	httpStart := time.Now()
	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer([]byte{}))
	req.Header.Set(HeaderNameContentType, contentType)

	if err != nil {
		log.Errorf("Error making http delete request: %+v.", err)
		return err
	}

	c.doPreRequest(req)
	log.Debugf("DELETE Request: %+v.", req)

	if resp, err = c.httpClient.Do(req); err != nil {
		log.Errorf("Error getting HTTP Response: %+v.", err)
		readResponseBody(resp)
		return err
	}

	httpElapsed := time.Since(httpStart)

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP DELETE ELAPSED TIME: %d ms.   -- Request: %s", (httpElapsed / 1000 / 1000), url)
	}

	return c.processResponse(resp, nil, expectedStatusCode)
}

func (c *Client) doPreRequest(request *http.Request) {

	if c.useAuthToken {
		request.Header.Set(HeaderNameAuthorization, fmt.Sprintf("Bearer %s", c.authToken))
	}

	if c.haveCsrfToken {
		request.Header.Set(HeaderNameCsrfToken, c.csrfToken)
	}
}

func readResponseBody(resp *http.Response) {

	var bodyBytes []byte
	var err error

	if bodyBytes, err = readBytes(resp.Body); err != nil {
		log.Errorf("Error reading HTTP Response: %+v.", err)
	}

	log.Debugf("TEXT OF RESPONSE: \n %s", string(bodyBytes[:]))
}

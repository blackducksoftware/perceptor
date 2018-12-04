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

	"github.com/juju/errors"
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
		return nil, errors.Annotate(err, "unable to instantiate cookie jar")
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

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

func readBytes(readCloser io.ReadCloser) ([]byte, error) {

	defer readCloser.Close()
	buf := new(bytes.Buffer)

	if _, err := buf.ReadFrom(readCloser); err != nil {
		return nil, errors.Trace(err)
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
		return errors.Annotate(err, "Error validating HTTP Response")
	}

	if result == nil {
		// Don't have a result to deserialize to, skip it
		return nil
	}

	if bodyBytes, err = readBytes(resp.Body); err != nil {
		return errors.Annotate(err, "Error reading HTTP Response")
	}

	if c.debugFlags&HubClientDebugContent != 0 {
		log.Debugf("START DEBUG: --------------------------------------------------------------------------- \n\n")
		log.Debugf("TEXT OF RESPONSE: \n %s", string(bodyBytes[:]))
		log.Debugf("END DEBUG: --------------------------------------------------------------------------- \n\n\n\n")
	}

	if err := json.Unmarshal(bodyBytes, result); err != nil {
		return errors.Annotate(err, "Error parsing HTTP Response")
	}

	return nil
}

func (c *Client) HttpGetJSON(url string, result interface{}, expectedStatusCode int) error {

	// TODO: Content type?

	var resp *http.Response
	var err error

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP STARTING GET REQUEST: %s", url)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		return errors.Annotate(err, "Error making http get request")
	}

	c.doPreRequest(req)

	httpStart := time.Now()
	if resp, err = c.httpClient.Do(req); err != nil {
		return errors.Annotate(err, "Error getting HTTP Response")
	}

	httpElapsed := time.Since(httpStart)

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP GET ELAPSED TIME: %d ms.   -- Request: %s", (httpElapsed / 1000 / 1000), url)
	}

	return c.processResponse(resp, result, expectedStatusCode)
}

func (c *Client) HttpPutJSON(url string, data interface{}, contentType string, expectedStatusCode int) error {

	var resp *http.Response
	var err error

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP STARTING PUT REQUEST: %s", url)
	}

	// Encode json
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	if err := enc.Encode(&data); err != nil {
		return errors.Annotate(err, "Error encoding json")
	}

	req, err := http.NewRequest(http.MethodPut, url, &buf)
	if err != nil {
		return errors.Annotate(err, "Error making http put request")
	}

	req.Header.Set(HeaderNameContentType, contentType)

	c.doPreRequest(req)
	log.Debugf("PUT Request: %+v.", req)

	httpStart := time.Now()
	if resp, err = c.httpClient.Do(req); err != nil {
		readResponseBody(resp)
		return errors.Annotate(err, "Error getting HTTP Response")
	}

	httpElapsed := time.Since(httpStart)

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP PUT ELAPSED TIME: %d ms.   -- Request: %s", (httpElapsed / 1000 / 1000), url)
	}

	return c.processResponse(resp, nil, expectedStatusCode) // TODO: Maybe need a response too?
}

func (c *Client) HttpPostJSON(url string, data interface{}, contentType string, expectedStatusCode int) (string, error) {

	var resp *http.Response
	var err error

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP STARTING POST REQUEST: %s", url)
	}

	// Encode json
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	if err := enc.Encode(&data); err != nil {
		return "", errors.Annotate(err, "Error encoding json")
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return "", errors.Annotate(err, "Error making http post request")
	}

	req.Header.Set(HeaderNameContentType, contentType)

	c.doPreRequest(req)
	log.Debugf("POST Request: %+v.", req)

	httpStart := time.Now()
	if resp, err = c.httpClient.Do(req); err != nil {
		readResponseBody(resp)
		return "", errors.Annotate(err, "Error getting HTTP Response")
	}

	httpElapsed := time.Since(httpStart)

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP POST ELAPSED TIME: %d ms.   -- Request: %s", (httpElapsed / 1000 / 1000), url)
	}

	if err := c.processResponse(resp, nil, expectedStatusCode); err != nil {
		return "", errors.Trace(err)
	}

	return resp.Header.Get("Location"), nil
}

func (c *Client) HttpPostJSONExpectResult(url string, data interface{}, result interface{}, contentType string, expectedStatusCode int) (string, error) {

	var resp *http.Response
	var err error

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP STARTING POST REQUEST: %s", url)
	}

	// Encode json
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	if err := enc.Encode(&data); err != nil {
		return "", errors.Annotate(err, "Error encoding json")
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return "", errors.Annotate(err, "Error making http post request")
	}

	req.Header.Set(HeaderNameContentType, contentType)

	c.doPreRequest(req)
	log.Debugf("POST Request: %+v.", req)

	httpStart := time.Now()
	if resp, err = c.httpClient.Do(req); err != nil {
		readResponseBody(resp)
		return "", errors.Annotate(err, "Error getting HTTP Response")
	}

	httpElapsed := time.Since(httpStart)

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP POST ELAPSED TIME: %d ms.   -- Request: %s", (httpElapsed / 1000 / 1000), url)
	}

	if err := c.processResponse(resp, result, expectedStatusCode); err != nil {
		return "", errors.Trace(err)
	}

	return resp.Header.Get("Location"), nil
}

func (c *Client) HttpDelete(url string, contentType string, expectedStatusCode int) error {

	var resp *http.Response
	var err error

	if c.debugFlags&HubClientDebugTimings != 0 {
		log.Debugf("DEBUG HTTP STARTING DELETE REQUEST: %s", url)
	}

	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer([]byte{}))
	if err != nil {
		return errors.Annotate(err, "Error making http delete request")
	}

	req.Header.Set(HeaderNameContentType, contentType)

	c.doPreRequest(req)
	log.Debugf("DELETE Request: %+v.", req)

	httpStart := time.Now()
	if resp, err = c.httpClient.Do(req); err != nil {
		readResponseBody(resp)
		return errors.Annotate(err, "Error getting HTTP Response")
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

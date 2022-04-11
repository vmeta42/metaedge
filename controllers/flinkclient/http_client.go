/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2021 TenxCloud. All Rights Reserved.
 * 9/30/21, 3:24 PM  @author hu.xiaolin3
 */

package flinkclient

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-logr/logr"
)

// HTTPClient - HTTP client.
type HTTPClient struct {
	Log logr.Logger
}

type HTTPError struct {
	StatusCode int
	Status     string
}

func (e *HTTPError) Error() string {
	return e.Status
}

// Get - HTTP GET.
func (c *HTTPClient) Get(url string, outStructPtr interface{}) error {
	return c.doHTTP("GET", url, nil, outStructPtr)
}

// Post - HTTP POST.
func (c *HTTPClient) Post(
	url string, body []byte, outStructPtr interface{}) error {
	return c.doHTTP("POST", url, body, outStructPtr)
}

// Patch - HTTP PATCH.
func (c *HTTPClient) Patch(
	url string, body []byte, outStructPtr interface{}) error {
	return c.doHTTP("PATCH", url, body, outStructPtr)
}

func (c *HTTPClient) doHTTP(
	method string, url string, body []byte, outStructPtr interface{}) error {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	req, err := c.createRequest(method, url, body)
	c.Log.Info("HTTPClient", "url", url, "method", method, "error", err)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	c.Log.Info("HTTPClient", "status", resp.Status, "body", outStructPtr, "error", err)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
		return err
	}
	return c.readResponse(resp, outStructPtr)
}

func (c *HTTPClient) createRequest(
	method, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "flink-operator")
	return req, err
}

func (c *HTTPClient) readResponse(
	resp *http.Response, out interface{}) error {
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		err = json.Unmarshal(body, out)
	}
	return err
}

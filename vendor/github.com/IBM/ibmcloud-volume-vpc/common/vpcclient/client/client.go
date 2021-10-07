/**
 * Copyright 2020 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package client ...
package client

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
)

// handler ...
type handler interface {
	Before(request *Request) error
}

// SessionClient provides an interface for a REST API client
// go:generate counterfeiter -o fakes/client.go --fake-name SessionClient . SessionClient
type SessionClient interface {
	NewRequest(operation *Operation) *Request
	WithDebug(writer io.Writer) SessionClient
	WithAuthToken(authToken string) SessionClient
	WithPathParameter(name, value string) SessionClient
	WithQueryValue(name, value string) SessionClient
}

type client struct {
	baseURL       string
	httpClient    *http.Client
	pathParams    Params
	queryValues   url.Values
	authenHandler handler
	debugWriter   io.Writer
	resourceGroup string
	contextID     string
	context       context.Context
}

// New creates a new instance of a SessionClient
func New(ctx context.Context, baseURL string, queryValues url.Values, httpClient *http.Client, contextID string, resourceGroupID string) SessionClient {
	return &client{
		baseURL:       baseURL,
		httpClient:    httpClient,
		pathParams:    Params{},
		queryValues:   queryValues,
		authenHandler: &authenticationHandler{},
		contextID:     contextID,
		context:       ctx,
		resourceGroup: resourceGroupID,
	}
}

// NewRequest creates a request and configures it with the supplied operation
func (c *client) NewRequest(operation *Operation) *Request {
	headers := http.Header{}
	headers.Set("Accept", "application/json")
	headers.Set("User-Agent", models.UserAgent)
	if c.contextID != "" {
		headers.Set("X-Request-ID", c.contextID)
		headers.Set("X-Transaction-ID", c.contextID) // To avoid IKS cloudflare overriding X-Request-ID
	}

	if c.resourceGroup != "" {
		headers.Set("X-Auth-Resource-Group-ID", c.resourceGroup)
	}

	// Copy the query values to a new map
	qv := url.Values{}
	for k, v := range c.queryValues {
		qv[k] = v
	}

	return &Request{
		httpClient:    c.httpClient,
		context:       c.context,
		baseURL:       c.baseURL,
		operation:     operation,
		pathParams:    c.pathParams.Copy(),
		authenHandler: c.authenHandler,
		headers:       headers,
		debugWriter:   c.debugWriter,
		resourceGroup: c.resourceGroup,
		queryValues:   qv,
	}
}

// WithDebug enables debug for this SessionClient, outputting to the supplied writer
func (c *client) WithDebug(writer io.Writer) SessionClient {
	c.debugWriter = writer
	return c
}

// WithAuthToken supplies the authentication token to use for all requests made by this session
func (c *client) WithAuthToken(authToken string) SessionClient {
	c.authenHandler = &authenticationHandler{
		authToken: authToken,
	}
	return c
}

// WithPathParameter adds a path parameter to the request
func (c *client) WithPathParameter(name, value string) SessionClient {
	c.pathParams[name] = value
	return c
}

// WithQueryValue adds a query parameter to the request
func (c *client) WithQueryValue(name, value string) SessionClient {
	c.queryValues.Set(name, value)
	return c
}

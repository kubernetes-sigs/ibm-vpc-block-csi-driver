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

// Package riaas ...
package riaas

import (
	"context"
	"io"
	"net/http"
)

// Config for the Session
type Config struct {
	BaseURL       string
	AccountID     string
	Username      string
	APIKey        string
	ResourceGroup string
	Password      string
	ContextID     string

	DebugWriter   io.Writer
	HTTPClient    *http.Client
	Context       context.Context
	APIVersion    string
	APIGeneration int
}

func (c Config) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}

	return http.DefaultClient
}

func (c Config) baseURL() string {
	return c.BaseURL
}

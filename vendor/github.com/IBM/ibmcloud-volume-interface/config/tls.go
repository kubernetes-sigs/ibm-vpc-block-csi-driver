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

// Package config ...
package config

import (
	"crypto/tls"
	"net/http"
	"time"
)

// GeneralCAHttpClient returns an http.Client configured for general use
func GeneralCAHttpClient() (*http.Client, error) {
	httpClient := &http.Client{

		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12, // Require TLS 1.2 or higher
			},
		},

		// softlayer.go has been overriding http.DefaultClient and forcing 120s
		// timeout on us, so we'll continue to force it on ourselves in case
		// we've accidentally become acustomed to it.
		Timeout: time.Second * 120,
	}

	return httpClient, nil
}

// GeneralCAHttpClientWithTimeout returns an http.Client configured for general use
func GeneralCAHttpClientWithTimeout(timeout time.Duration) (*http.Client, error) {
	httpClient := &http.Client{

		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12, // Require TLS 1.2 or higher
			},
		},

		Timeout: timeout,
	}

	return httpClient, nil
}

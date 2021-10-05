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
	"errors"
)

// ErrAuthenticationRequired is returned if a request is made before an authentication
// token has been provided to the client
var ErrAuthenticationRequired = errors.New("authentication token required")

type authenticationHandler struct {
	authToken     string
	resourceGroup string
}

// Before is called before each request
func (a *authenticationHandler) Before(request *Request) error {
	request.resourceGroup = a.resourceGroup

	if a.authToken == "" {
		return ErrAuthenticationRequired
	}
	request.headers.Set("Authorization", "Bearer "+a.authToken)
	return nil
}

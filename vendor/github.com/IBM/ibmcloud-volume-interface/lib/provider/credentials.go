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

// Package provider ...
package provider

// AuthType ...
type AuthType string

const (
	// IaaSAPIKey is an IaaS-native user ID and API key
	IaaSAPIKey = AuthType("IAAS_API_KEY")

	// IAMAPIKey is an IAM account ID and API key
	IAMAPIKey = AuthType("IAM_API_KEY")

	// IAMAccessToken indicates the credential is an IAM access token
	IAMAccessToken = AuthType("IAM_ACCESS_TOKEN")
)

// ContextCredentials represents user credentials (e.g. API key) for volume operations from IaaS provider
type ContextCredentials struct {
	AuthType       AuthType
	DefaultAccount bool
	Region         string
	IAMAccountID   string
	UserID         string `json:"-"` // Do not trace
	Credential     string `json:"-"` // Do not trace

	// ContextID is an optional request/context/correlation identifier for diagnostics (need not be unique)
	ContextID string
}

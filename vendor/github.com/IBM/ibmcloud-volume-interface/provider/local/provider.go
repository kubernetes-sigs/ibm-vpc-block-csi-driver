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

// Package local ...
package local

import (
	"context"

	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	"go.uber.org/zap"
)

// Provider describes the contract that is implemented by an internal provider implementation
//go:generate counterfeiter -o fakes/provider.go  --fake-name Provider . Provider
type Provider interface {
	// OpenSession begins and initialises a new provider session.
	// The implementation can choose to verify the credentials and return an error if they are invalid.
	// Alternatively, the implementation can choose to defer credential verification until individual
	// methods of the context are called.
	OpenSession(context.Context, provider.ContextCredentials, *zap.Logger) (provider.Session, error)

	// Returns a configured ContextCredentialsFactory for this provider
	ContextCredentialsFactory(datacenter *string) (ContextCredentialsFactory, error)
}

// ContextCredentialsFactory is a factory which can generate ContextCredentials instances
//go:generate counterfeiter -o fakes/context_credentials_factory.go --fake-name ContextCredentialsFactory . ContextCredentialsFactory
type ContextCredentialsFactory interface {
	// ForIaaSAPIKey returns a config using an explicit API key for an IaaS user account
	ForIaaSAPIKey(iamAccountID, iaasUserID, iaasAPIKey string, logger *zap.Logger) (provider.ContextCredentials, error)

	// ForIAMAPIKey returns a config derived from an IAM API key (if applicable)
	ForIAMAPIKey(iamAccountID, iamAPIKey string, logger *zap.Logger) (provider.ContextCredentials, error)

	// ForIAMAccessToken returns a config derived from an IAM API key (if applicable)
	ForIAMAccessToken(apiKey string, logger *zap.Logger) (provider.ContextCredentials, error)
}

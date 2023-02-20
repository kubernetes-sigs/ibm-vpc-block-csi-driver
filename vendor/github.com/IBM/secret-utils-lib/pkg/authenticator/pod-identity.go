/**
 * Copyright 2022 IBM Corp.
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

package authenticator

import (
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/secret-utils-lib/pkg/token"
	"github.com/IBM/secret-utils-lib/pkg/utils"
	"go.uber.org/zap"
)

// ComputeIdentityAuthenticator ...
type ComputeIdentityAuthenticator struct {
	authenticator *core.ContainerAuthenticator
	logger        *zap.Logger
	token         string
}

// NewComputeIdentityAuthenticator ...
func NewComputeIdentityAuthenticator(profileID string, logger *zap.Logger) *ComputeIdentityAuthenticator {
	ca := new(ComputeIdentityAuthenticator)
	ca.authenticator = new(core.ContainerAuthenticator)
	ca.authenticator.IAMProfileID = profileID
	ca.logger = logger
	return ca
}

// GetToken ...
func (ca *ComputeIdentityAuthenticator) GetToken(freshTokenRequired bool) (string, uint64, error) {
	var err error
	var tokenlifetime uint64

	if !freshTokenRequired {
		// Fetching token life time of the token in cache
		tokenlifetime, err = token.CheckTokenLifeTime(ca.token)
		if err == nil {
			ca.logger.Info("Fetched iam token from cache", zap.Uint64("token-life-time-in-seconds", tokenlifetime))
			return ca.token, tokenlifetime, nil
		}
	}

	tokenResponse, err := ca.authenticator.RequestToken()
	if err != nil {
		ca.logger.Error("Error fetching fresh token", zap.Error(err))
		// If the cluster cannot access private iam endpoint, hence returns timeout error, switch to public IAM endpoint.
		if !isTimeout(err) {
			return "", tokenlifetime, utils.Error{Description: "Error fetching iam token using compute identity", BackendError: err.Error()}
		}

		ca.logger.Info("Updating iam URL to public, if it is private and retrying to fetch token")
		if !resetIAMURL(ca) {
			return "", tokenlifetime, utils.Error{Description: "Error fetching iam token using compute identity", BackendError: err.Error()}
		}
		return ca.GetToken(freshTokenRequired)
	}

	if tokenResponse == nil {
		ca.logger.Error("Token response received is empty")
		return "", tokenlifetime, utils.Error{Description: utils.ErrEmptyTokenResponse}
	}

	tokenlifetime, err = token.CheckTokenLifeTime(tokenResponse.AccessToken)
	if err != nil {
		ca.logger.Error("Error fetching token lifetime for new token", zap.Error(err))
		return "", tokenlifetime, utils.Error{Description: "Error fetching token lifetime", BackendError: err.Error()}
	}
	ca.token = tokenResponse.AccessToken

	ca.logger.Info("Fetched fresh iam token", zap.Uint64("token-life-time-in-seconds", tokenlifetime))
	return ca.token, tokenlifetime, nil
}

// GetSecret ...
func (ca *ComputeIdentityAuthenticator) GetSecret() string {
	return ca.authenticator.IAMProfileID
}

// SetSecret ...
func (ca *ComputeIdentityAuthenticator) SetSecret(secret string) {
	ca.authenticator.IAMProfileID = secret
}

// SetURL ...
func (ca *ComputeIdentityAuthenticator) SetURL(url string) {
	ca.authenticator.URL = url
}

// IsSecretEncrypted ...
func (ca *ComputeIdentityAuthenticator) IsSecretEncrypted() bool {
	return false
}

// SetEncryption ...
func (ca *ComputeIdentityAuthenticator) SetEncryption(encrypted bool) {
	ca.logger.Info("Unimplemented")
}

// getURL ...
func (ca *ComputeIdentityAuthenticator) getURL() string {
	return ca.authenticator.URL
}

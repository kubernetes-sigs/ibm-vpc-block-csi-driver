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

// APIKeyAuthenticator ...
type APIKeyAuthenticator struct {
	authenticator     *core.IamAuthenticator
	logger            *zap.Logger
	isSecretEncrypted bool
	token             string
}

// NewIamAuthenticator ...
func NewIamAuthenticator(apikey string, logger *zap.Logger) *APIKeyAuthenticator {
	aa := new(APIKeyAuthenticator)
	aa.authenticator = new(core.IamAuthenticator)
	aa.authenticator.ApiKey = apikey
	aa.logger = logger
	return aa
}

// GetToken ...
func (aa *APIKeyAuthenticator) GetToken(freshTokenRequired bool) (string, uint64, error) {
	var err error
	var tokenlifetime uint64

	if !freshTokenRequired {
		// Fetching token life time of the token in cache
		tokenlifetime, err = token.CheckTokenLifeTime(aa.token)
		if err == nil {
			aa.logger.Info("Fetched iam token from cache", zap.Uint64("token-life-time-in-seconds", tokenlifetime))
			return aa.token, tokenlifetime, nil
		}
	}

	aa.logger.Info("Fetching fresh token")
	tokenResponse, err := aa.authenticator.RequestToken()
	if err != nil {
		aa.logger.Error("Error fetching fresh token", zap.Error(err))
		// If the cluster cannot access private iam endpoint, hence returns timeout error, switch to public IAM endpoint.
		if !isTimeout(err) {
			return "", tokenlifetime, utils.Error{Description: "Error fetching iam token using api key", BackendError: err.Error()}
		}

		aa.logger.Info("Updating iam URL to public, if it is private and retrying to fetch token")
		if !resetIAMURL(aa) {
			return "", tokenlifetime, utils.Error{Description: "Error fetching iam token using api key", BackendError: err.Error()}
		}
		return aa.GetToken(freshTokenRequired)
	}

	if tokenResponse == nil {
		aa.logger.Error("Token response received is empty")
		return "", tokenlifetime, utils.Error{Description: utils.ErrEmptyTokenResponse}
	}

	tokenlifetime, err = token.CheckTokenLifeTime(tokenResponse.AccessToken)
	if err != nil {
		aa.logger.Error("Error fetching token lifetime for new token", zap.Error(err))
		return "", tokenlifetime, utils.Error{Description: "Error fetching token lifetime", BackendError: err.Error()}
	}
	aa.token = tokenResponse.AccessToken

	aa.logger.Info("Fetched fresh iam token", zap.Uint64("token-life-time-in-seconds", tokenlifetime))
	return aa.token, tokenlifetime, nil
}

// GetSecret ...
func (aa *APIKeyAuthenticator) GetSecret() string {
	return aa.authenticator.ApiKey
}

// SetSecret ...
func (aa *APIKeyAuthenticator) SetSecret(secret string) {
	aa.authenticator.ApiKey = secret
}

// SetURL ...
func (aa *APIKeyAuthenticator) SetURL(url string) {
	aa.authenticator.URL = url
}

// IsSecretEncrypted ...
func (aa *APIKeyAuthenticator) IsSecretEncrypted() bool {
	return aa.isSecretEncrypted
}

// SetEncryption ...
func (aa *APIKeyAuthenticator) SetEncryption(encrypted bool) {
	aa.isSecretEncrypted = encrypted
}

// getURL ...
func (aa *APIKeyAuthenticator) getURL() string {
	return aa.authenticator.URL
}

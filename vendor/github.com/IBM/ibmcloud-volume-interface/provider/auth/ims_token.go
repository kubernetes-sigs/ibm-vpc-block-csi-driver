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

// Package auth ...
package auth

import (
	"strconv"

	"go.uber.org/zap"

	"github.com/IBM/ibmcloud-volume-interface/provider/iam"
	"github.com/IBM/ibmcloud-volume-interface/provider/local"

	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
)

const (
	// IMSToken is an IMS user ID and token
	IMSToken = provider.AuthType("IMS_TOKEN")
	// IAMAccessToken ...
	IAMAccessToken = provider.AuthType("IAM_ACCESS_TOKEN")
)

// ForRefreshToken ...
func (ccf *ContextCredentialsFactory) ForRefreshToken(refreshToken string, logger *zap.Logger) (provider.ContextCredentials, error) {
	accessToken, err := ccf.TokenExchangeService.ExchangeRefreshTokenForAccessToken(refreshToken, logger)
	if err != nil {
		// Must preserve provider error code in the ErrorProviderAccountTemporarilyLocked case
		logger.Error("Unable to retrieve access token from refresh token", local.ZapError(err))
		return provider.ContextCredentials{}, err
	}

	imsToken, err := ccf.TokenExchangeService.ExchangeAccessTokenForIMSToken(*accessToken, logger)
	if err != nil {
		// Must preserve provider error code in the ErrorProviderAccountTemporarilyLocked case
		logger.Error("Unable to retrieve IAM token from access token", local.ZapError(err))
		return provider.ContextCredentials{}, err
	}

	return forIMSToken("", imsToken), nil
}

// ForIAMAPIKey ...
func (ccf *ContextCredentialsFactory) ForIAMAPIKey(iamAccountID, apiKey string, logger *zap.Logger) (provider.ContextCredentials, error) {
	imsToken, err := ccf.TokenExchangeService.ExchangeIAMAPIKeyForIMSToken(apiKey, logger)
	if err != nil {
		// Must preserve provider error code in the ErrorProviderAccountTemporarilyLocked case
		logger.Error("Unable to retrieve IMS credentials from IAM API key", local.ZapError(err))
		return provider.ContextCredentials{}, err
	}

	return forIMSToken(iamAccountID, imsToken), nil
}

// ForIAMAccessToken ...
func (ccf *ContextCredentialsFactory) ForIAMAccessToken(apiKey string, logger *zap.Logger) (provider.ContextCredentials, error) {
	iamAccessToken, err := ccf.TokenExchangeService.ExchangeIAMAPIKeyForAccessToken(apiKey, logger)
	if err != nil {
		logger.Error("Unable to retrieve IAM access token from IAM API key", local.ZapError(err))
		return provider.ContextCredentials{}, err
	}
	iamAccountID, err := ccf.TokenExchangeService.GetIAMAccountIDFromAccessToken(iam.AccessToken{Token: iamAccessToken.Token}, logger)
	if err != nil {
		logger.Error("Unable to retrieve IAM access token from IAM API key", local.ZapError(err))
		return provider.ContextCredentials{}, err
	}

	return forIAMAccessToken(iamAccountID, iamAccessToken), nil
}

// forIMSToken ...
func forIMSToken(iamAccountID string, imsToken *iam.IMSToken) provider.ContextCredentials {
	return provider.ContextCredentials{
		AuthType:     IMSToken,
		IAMAccountID: iamAccountID,
		UserID:       strconv.Itoa(imsToken.UserID),
		Credential:   imsToken.Token,
	}
}

// forIAMAccessToken ...
func forIAMAccessToken(iamAccountID string, iamAccessToken *iam.AccessToken) provider.ContextCredentials {
	return provider.ContextCredentials{
		AuthType:     IAMAccessToken,
		IAMAccountID: iamAccountID,
		Credential:   iamAccessToken.Token,
	}
}

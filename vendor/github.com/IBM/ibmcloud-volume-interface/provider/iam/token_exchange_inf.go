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

// Package iam ...
package iam

import (
	"go.uber.org/zap"
)

// IMSToken ...
type IMSToken struct {
	UserID int    // Numerical ID is safe to trace
	Token  string `json:"-"` // Do not trace
}

// AccessToken ...
type AccessToken struct {
	Token string `json:"-"` // Do not trace
}

// TokenExchangeService ...
type TokenExchangeService interface {

	// ExchangeRefreshTokenForAccessToken ...
	// TODO Deprecate when no longer reliant on refresh token authentication
	ExchangeRefreshTokenForAccessToken(refreshToken string, logger *zap.Logger) (*AccessToken, error)

	// ExchangeAccessTokenForIMSToken ...
	ExchangeAccessTokenForIMSToken(accessToken AccessToken, logger *zap.Logger) (*IMSToken, error)

	// ExchangeIAMAPIKeyForIMSToken ...
	ExchangeIAMAPIKeyForIMSToken(iamAPIKey string, logger *zap.Logger) (*IMSToken, error)

	// ExchangeIAMAPIKeyForAccessToken ...
	ExchangeIAMAPIKeyForAccessToken(iamAPIKey string, logger *zap.Logger) (*AccessToken, error)

	// GetIAMAccountIDFromAccessToken ...
	GetIAMAccountIDFromAccessToken(accessToken AccessToken, logger *zap.Logger) (string, error)
}

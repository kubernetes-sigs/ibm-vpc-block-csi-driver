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
	"errors"

	"github.com/golang-jwt/jwt/v4"

	"go.uber.org/zap"
)

type accessTokenClaims struct {
	jwt.StandardClaims

	Account struct {
		Bss string `json:"bss"`
	} `json:"account"`
}

func (r *tokenExchangeService) GetIAMAccountIDFromAccessToken(accessToken AccessToken, logger *zap.Logger) (accountID string, err error) {
	// TODO - TEMPORARY CODE - VERIFY SIGNATURE HERE
	token, _, err := new(jwt.Parser).ParseUnverified(accessToken.Token, &accessTokenClaims{})
	if err != nil {
		return
	}
	token.Valid = true
	// TODO - TEMPORARY CODE - DONT OVERRIDE VERIFICATION

	claims, haveClaims := token.Claims.(*accessTokenClaims)

	logger.Debug("Access token parsed", zap.Bool("haveClaims", haveClaims), zap.Bool("valid", token.Valid))

	if !token.Valid || !haveClaims {
		err = errors.New("access token invalid")
		return
	}

	accountID = claims.Account.Bss
	logger.Debug("GetIAMAccountIDFromAccessToken", zap.Reflect("claims.Account.Bss", claims.Account.Bss))

	return
}

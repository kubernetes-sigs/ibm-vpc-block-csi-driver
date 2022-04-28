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

package token

import (
	"errors"
	"time"

	"github.com/IBM/secret-utils-lib/pkg/utils"
	"github.com/golang-jwt/jwt"
)

// CheckTokenLifeTime checks whether the lifetime of token is valid or not
// and returns life time of the token
func CheckTokenLifeTime(tokenString string) (uint64, error) {
	var tokenLifeTime uint64

	token, err := parseToken(tokenString)
	if err != nil {
		return tokenLifeTime, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if err := claims.Valid(); err != nil {
			return tokenLifeTime, err
		}
		currentTime := time.Now().Unix()
		var expiryTime interface{}
		if expiryTime, ok = claims["exp"]; !ok {
			return tokenLifeTime, errors.New("unable to find expiry time of token")
		}
		tokenLifeTime = uint64(expiryTime.(float64)) - uint64(currentTime)
		if tokenLifeTime < utils.TokenExpirydiff {
			return tokenLifeTime, errors.New("token life time is less than expected value")
		}
		return tokenLifeTime, nil
	}
	return tokenLifeTime, errors.New("unable to fetch token claims")
}

// parseToken parses token string to jwt token
func parseToken(tokenString string) (*jwt.Token, error) {
	if tokenString == "" {
		return nil, errors.New("empty token string")
	}
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	return token, err
}

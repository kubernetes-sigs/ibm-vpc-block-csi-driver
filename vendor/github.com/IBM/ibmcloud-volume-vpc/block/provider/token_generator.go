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

import (
	"crypto/rsa"
	"errors"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"

	"github.com/IBM/ibmcloud-volume-interface/config"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	"github.com/IBM/ibmcloud-volume-interface/provider/auth"
	"github.com/IBM/ibmcloud-volume-interface/provider/local"
)

// tokenGenerator ...
type tokenGenerator struct {
	config *config.VPCProviderConfig

	tokenKID        string
	tokenTTL        time.Duration
	tokenBeforeTime time.Duration

	privateKey *rsa.PrivateKey // Secret. Do not export
}

// readConfig ...
func (tg *tokenGenerator) readConfig(logger zap.Logger) (err error) {
	logger.Info("Entering readConfig")
	defer func() {
		logger.Info("Exiting readConfig", zap.Duration("tokenTTL", tg.tokenTTL), zap.Duration("tokenBeforeTime", tg.tokenBeforeTime), zap.String("tokenKID", tg.tokenKID), local.ZapError(err))
	}()

	if tg.privateKey != nil {
		return
	}

	path := filepath.Join(GetEtcPath(), tg.tokenKID)

	pem, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		logger.Error("Error reading PEM", local.ZapError(err))
		return
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(pem)
	if err != nil {
		logger.Error("Error parsing PEM", local.ZapError(err))
		return
	}

	tg.privateKey = privateKey

	return
}

// buildToken ...
func (tg *tokenGenerator) buildToken(contextCredentials provider.ContextCredentials, ts time.Time, logger zap.Logger) (token *jwt.Token, err error) {
	logger.Info("Entering getJWTToken", zap.Reflect("contextCredentials", contextCredentials))
	defer func() {
		logger.Info("Exiting getJWTToken", zap.Reflect("token", token), local.ZapError(err))
	}()

	err = tg.readConfig(logger)
	if err != nil {
		return
	}

	claims := jwt.MapClaims{
		"iss": "armada",
		"exp": ts.Add(tg.tokenTTL).Unix(),
		"nbf": ts.Add(tg.tokenBeforeTime).Unix(),
		"iat": ts.Unix(),
	}

	switch {
	case contextCredentials.UserID == "":
		errStr := "User ID is not configured"
		logger.Error(errStr)
		err = errors.New(errStr)
		return

	case contextCredentials.AuthType == auth.IMSToken:
		claims["ims_user_id"] = contextCredentials.UserID

	default:
		claims["ims_username"] = contextCredentials.UserID
	}

	token = jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = tg.tokenKID

	return
}

// getServiceToken ...
func (tg *tokenGenerator) getServiceToken(contextCredentials provider.ContextCredentials, logger zap.Logger) (signedToken *string, err error) {
	token, err := tg.buildToken(contextCredentials, time.Now(), logger)
	if err != nil {
		return
	}

	signedString, err := token.SignedString(tg.privateKey)
	if err != nil {
		return
	}

	signedToken = &signedString

	return
}

// GetEtcPath returns the path to the etc directory
func GetEtcPath() string {
	goPath := config.GetGoPath()
	srcPath := filepath.Join("src", "github.com", "IBM",
		"ibmcloud-volume-vpc")
	return filepath.Join(goPath, srcPath, "etc")
}

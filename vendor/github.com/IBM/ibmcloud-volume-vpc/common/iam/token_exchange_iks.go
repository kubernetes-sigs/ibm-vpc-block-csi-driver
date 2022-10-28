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
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/IBM-Cloud/ibm-cloud-cli-sdk/common/rest"
	"github.com/IBM/ibmcloud-volume-interface/config"
	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/IBM/ibmcloud-volume-interface/provider/iam"
	"github.com/IBM/secret-common-lib/pkg/secret_provider"
	sp "github.com/IBM/secret-utils-lib/pkg/secret_provider"
	"go.uber.org/zap"
)

// tokenExchangeIKSService ...
type tokenExchangeIKSService struct {
	iksAuthConfig  *IksAuthConfiguration
	httpClient     *http.Client
	secretprovider sp.SecretProviderInterface
}

// IksAuthConfiguration ...
type IksAuthConfiguration struct {
	PrivateAPIRoute string
	IamAPIKey       string
	CSRFToken       string
}

// TokenExchangeService ...
var _ iam.TokenExchangeService = &tokenExchangeIKSService{}

// NewTokenExchangeIKSService ...
func NewTokenExchangeIKSService(iksAuthConfig *IksAuthConfiguration) (iam.TokenExchangeService, error) {
	httpClient, err := config.GeneralCAHttpClient()
	if err != nil {
		return nil, err
	}
	providerType := map[string]string{
		secret_provider.ProviderType: secret_provider.VPC,
	}
	spObject, err := secret_provider.NewSecretProvider(providerType)
	if err != nil {
		return nil, err
	}
	return &tokenExchangeIKSService{
		iksAuthConfig:  iksAuthConfig,
		httpClient:     httpClient,
		secretprovider: spObject,
	}, nil
}

// tokenExchangeIKSRequest ...
type tokenExchangeIKSRequest struct {
	tes          *tokenExchangeIKSService
	request      *rest.Request
	client       *rest.Client
	logger       *zap.Logger
	errorRetrier *util.ErrorRetrier
}

// tokenExchangeIKSResponse ...
type tokenExchangeIKSResponse struct {
	AccessToken string `json:"token"`
	//ImsToken    string `json:"ims_token"`
}

// ExchangeRefreshTokenForAccessToken ...
func (tes *tokenExchangeIKSService) ExchangeRefreshTokenForAccessToken(refreshToken string, logger *zap.Logger) (*iam.AccessToken, error) {
	r := tes.newTokenExchangeRequest(logger)
	return r.exchangeForAccessToken()
}

// ExchangeIAMAPIKeyForAccessToken ...
func (tes *tokenExchangeIKSService) ExchangeIAMAPIKeyForAccessToken(iamAPIKey string, logger *zap.Logger) (*iam.AccessToken, error) {
	logger.Info("Fetching using secret provider")
	token, _, err := tes.secretprovider.GetDefaultIAMToken(false)
	if err != nil {
		logger.Error("Error fetching iam token", zap.Error(err))
		return nil, err
	}
	logger.Info("Successfully fetched iam token")
	return &iam.AccessToken{Token: token}, nil
}

// newTokenExchangeRequest ...
func (tes *tokenExchangeIKSService) newTokenExchangeRequest(logger *zap.Logger) *tokenExchangeIKSRequest {
	client := rest.NewClient()
	client.HTTPClient = tes.httpClient
	retyrInterval, _ := time.ParseDuration("3s")
	return &tokenExchangeIKSRequest{
		tes:          tes,
		request:      rest.PostRequest(fmt.Sprintf("%s/v1/iam/apikey", tes.iksAuthConfig.PrivateAPIRoute)),
		client:       client,
		logger:       logger,
		errorRetrier: util.NewErrorRetrier(40, retyrInterval, logger),
	}
}

// ExchangeAccessTokenForIMSToken ...
func (tes *tokenExchangeIKSService) ExchangeAccessTokenForIMSToken(accessToken iam.AccessToken, logger *zap.Logger) (*iam.IMSToken, error) {
	return nil, nil
}

// ExchangeIAMAPIKeyForIMSToken ...
func (tes *tokenExchangeIKSService) ExchangeIAMAPIKeyForIMSToken(iamAPIKey string, logger *zap.Logger) (*iam.IMSToken, error) {
	return nil, nil
}

func (tes *tokenExchangeIKSService) GetIAMAccountIDFromAccessToken(accessToken iam.AccessToken, logger *zap.Logger) (accountID string, err error) {
	return "Not required to implement", nil
}

// exchangeForAccessToken ...
func (r *tokenExchangeIKSRequest) exchangeForAccessToken() (*iam.AccessToken, error) {
	var iamResp *tokenExchangeIKSResponse
	var err error
	err = r.errorRetrier.ErrorRetry(func() (error, bool) {
		iamResp, err = r.sendTokenExchangeRequest()
		return err, !iam.IsConnectionError(err) // Skip retry if its not connection error
	})
	if err != nil {
		return nil, err
	}
	return &iam.AccessToken{Token: iamResp.AccessToken}, nil
}

// sendTokenExchangeRequest ...
func (r *tokenExchangeIKSRequest) sendTokenExchangeRequest() (*tokenExchangeIKSResponse, error) {
	r.logger.Info("In tokenExchangeIKSRequest's sendTokenExchangeRequest()")
	// Set headers
	r.request = r.request.Add("X-CSRF-TOKEN", r.tes.iksAuthConfig.CSRFToken)
	// Setting body
	var apikey = struct {
		APIKey string `json:"apikey"`
	}{
		APIKey: r.tes.iksAuthConfig.IamAPIKey,
	}
	r.request = r.request.Body(&apikey)

	var successV tokenExchangeIKSResponse
	var errorV = struct {
		ErrorCode        string `json:"code"`
		ErrorDescription string `json:"description"`
		ErrorType        string `json:"type"`
		IncidentID       string `json:"incidentID"`
	}{}

	r.logger.Info("Sending IAM token exchange request to container api server")
	resp, err := r.client.Do(r.request, &successV, &errorV)
	if err != nil {
		errString := err.Error()
		r.logger.Error("IAM token exchange request failed", zap.Reflect("Response", resp), zap.Error(err))
		if strings.Contains(errString, "no such host") {
			return nil, util.NewError("EndpointNotReachable", errString)
		} else if strings.Contains(errString, "Timeout") {
			return nil, util.NewError("Timeout", errString)
		} else {
			return nil, util.NewError("ErrorUnclassified",
				"IAM token exchange request failed", err)
		}
	}

	if resp != nil && resp.StatusCode == 200 {
		r.logger.Debug("IAM token exchange request successful")
		return &successV, nil
	}
	// closing resp body only when some issues, in case of success its not required
	// to close here
	defer resp.Body.Close()

	if errorV.ErrorDescription != "" {
		r.logger.Error("IAM token exchange request failed with message",
			zap.Int("StatusCode", resp.StatusCode), zap.Reflect("API IncidentID", errorV.IncidentID),
			zap.Reflect("Error", errorV))

		err := util.NewError("ErrorFailedTokenExchange",
			"IAM token exchange request failed: "+errorV.ErrorDescription,
			errors.New(errorV.ErrorCode+" "+errorV.ErrorType+", Description: "+errorV.ErrorDescription+", API IncidentID:"+errorV.IncidentID))
		return nil, err
	}

	r.logger.Error("Unexpected IAM token exchange response",
		zap.Int("StatusCode", resp.StatusCode), zap.Reflect("Response", resp))

	return nil,
		util.NewError("ErrorUnclassified",
			"Unexpected IAM token exchange response")
}

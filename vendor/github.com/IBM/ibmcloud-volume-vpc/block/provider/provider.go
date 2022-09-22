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
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/IBM/ibmcloud-volume-interface/config"
	"github.com/IBM/ibmcloud-volume-interface/lib/metrics"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/IBM/ibmcloud-volume-interface/provider/iam"
	"github.com/IBM/ibmcloud-volume-interface/provider/local"
	vpcconfig "github.com/IBM/ibmcloud-volume-vpc/block/vpcconfig"
	vpcauth "github.com/IBM/ibmcloud-volume-vpc/common/auth"
	"github.com/IBM/ibmcloud-volume-vpc/common/messages"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/riaas"
	"go.uber.org/zap"
)

const (
	// VPCClassic ...
	VPCClassic = "gc"
	// VPCNextGen ...
	VPCNextGen = "g2"
	// PrivatePrefix ...
	PrivatePrefix = "private-"
	// BasePrivateURL ...
	BasePrivateURL = "https://" + PrivatePrefix
	// HTTPSLength ...
	HTTPSLength = 8
	// NEXTGenProvider ...
	NEXTGenProvider = 2
)

// VPCBlockProvider implements provider.Provider
type VPCBlockProvider struct {
	timeout        time.Duration
	Config         *vpcconfig.VPCBlockConfig
	tokenGenerator *tokenGenerator
	ContextCF      local.ContextCredentialsFactory

	ClientProvider riaas.RegionalAPIClientProvider
	httpClient     *http.Client
	APIConfig      riaas.Config
}

var _ local.Provider = &VPCBlockProvider{}

// NewProvider initialises an instance of an IaaS provider.
func NewProvider(conf *vpcconfig.VPCBlockConfig, logger *zap.Logger) (local.Provider, error) {
	logger.Info("Entering NewProvider")

	if conf.VPCConfig == nil {
		return nil, errors.New("incomplete config for VPCBlockProvider")
	}

	//Do config validation and enable only one generationType (i.e VPC-Classic | VPC-NG)
	gcConfigFound := (conf.VPCConfig.EndpointURL != "" || conf.VPCConfig.PrivateEndpointURL != "") && (conf.VPCConfig.TokenExchangeURL != "" || conf.VPCConfig.IKSTokenExchangePrivateURL != "") && (conf.VPCConfig.APIKey != "") && (conf.VPCConfig.ResourceGroupID != "")
	g2ConfigFound := (conf.VPCConfig.G2EndpointPrivateURL != "" || conf.VPCConfig.G2EndpointURL != "") && (conf.VPCConfig.IKSTokenExchangePrivateURL != "" || conf.VPCConfig.G2TokenExchangeURL != "") && (conf.VPCConfig.G2APIKey != "") && (conf.VPCConfig.G2ResourceGroupID != "")
	//if both config found, look for VPCTypeEnabled, otherwise default to GC
	//Incase of NG configurations, override the base properties.
	if (gcConfigFound && g2ConfigFound && conf.VPCConfig.VPCTypeEnabled == VPCNextGen) || (!gcConfigFound && g2ConfigFound) {
		// overwrite the common variable in case of g2 i.e gen2, first preferences would be private endpoint
		if conf.VPCConfig.G2EndpointPrivateURL != "" {
			conf.VPCConfig.EndpointURL = conf.VPCConfig.G2EndpointPrivateURL
		} else {
			conf.VPCConfig.EndpointURL = conf.VPCConfig.G2EndpointURL
		}

		// update iam based public toke exchange endpoint
		conf.VPCConfig.TokenExchangeURL = conf.VPCConfig.G2TokenExchangeURL

		conf.VPCConfig.APIKey = conf.VPCConfig.G2APIKey
		conf.VPCConfig.ResourceGroupID = conf.VPCConfig.G2ResourceGroupID

		//Set API Generation As 2 (if unspecified in config/ENV-VAR)
		if conf.VPCConfig.G2VPCAPIGeneration <= 0 {
			conf.VPCConfig.G2VPCAPIGeneration = NEXTGenProvider
		}
		conf.VPCConfig.VPCAPIGeneration = conf.VPCConfig.G2VPCAPIGeneration

		//Set the APIVersion Date, it can be different in GC and NG
		if conf.VPCConfig.G2APIVersion != "" {
			conf.VPCConfig.APIVersion = conf.VPCConfig.G2APIVersion
		}

		//set provider-type (this usually comes from the secret)
		if conf.VPCConfig.VPCBlockProviderType != VPCNextGen {
			conf.VPCConfig.VPCBlockProviderType = VPCNextGen
		}

		//Mark this as enabled/active
		if conf.VPCConfig.VPCTypeEnabled != VPCNextGen {
			conf.VPCConfig.VPCTypeEnabled = VPCNextGen
		}
	} else { //This is GC, no-override required
		conf.VPCConfig.VPCBlockProviderType = VPCClassic //incase of gc, i dont see its being set in slclient.toml, but NG cluster has this
		// For backward compatibility as some of the cluster storage secret may not have private gc endpoint url
		if conf.VPCConfig.PrivateEndpointURL != "" {
			conf.VPCConfig.EndpointURL = conf.VPCConfig.PrivateEndpointURL
		}
	}

	contextCF, err := vpcauth.NewVPCContextCredentialsFactory(conf)
	if err != nil {
		return nil, err
	}
	timeoutString := conf.VPCConfig.VPCTimeout
	if timeoutString == "" || timeoutString == "0s" {
		logger.Info("Using VPC default timeout")
		timeoutString = "120s"
	}
	timeout, err := time.ParseDuration(timeoutString)
	if err != nil {
		return nil, err
	}

	httpClient, err := config.GeneralCAHttpClientWithTimeout(timeout)
	if err != nil {
		logger.Error("Failed to prepare HTTP client", util.ZapError(err))
		return nil, err
	}

	// SetRetryParameters sets the retry logic parameters
	SetRetryParameters(conf.VPCConfig.MaxRetryAttempt, conf.VPCConfig.MaxRetryGap)
	provider := &VPCBlockProvider{
		timeout:        timeout,
		Config:         conf,
		tokenGenerator: &tokenGenerator{config: conf.VPCConfig},
		ContextCF:      contextCF,
		httpClient:     httpClient,
		APIConfig: riaas.Config{
			BaseURL:       conf.VPCConfig.EndpointURL,
			HTTPClient:    httpClient,
			APIVersion:    conf.VPCConfig.APIVersion,
			APIGeneration: conf.VPCConfig.VPCAPIGeneration,
			ResourceGroup: conf.VPCConfig.ResourceGroupID,
		},
	}
	// Update VPC config for IKS deployment
	provider.Config.VPCConfig.IsIKS = conf.IKSConfig != nil && conf.IKSConfig.Enabled
	userError.MessagesEn = messages.InitMessages()
	return provider, nil
}

// ContextCredentialsFactory ...
func (vpcp *VPCBlockProvider) ContextCredentialsFactory(zone *string) (local.ContextCredentialsFactory, error) {
	//  Datacenter name not required by VPC provider implementation
	return vpcp.ContextCF, nil
}

// OpenSession opens a session on the provider
func (vpcp *VPCBlockProvider) OpenSession(ctx context.Context, contextCredentials provider.ContextCredentials, ctxLogger *zap.Logger) (provider.Session, error) {
	ctxLogger.Info("Entering OpenSession")
	defer metrics.UpdateDurationFromStart(ctxLogger, "OpenSession", time.Now())
	defer func() {
		ctxLogger.Debug("Exiting OpenSession")
	}()

	// validate that we have what we need - i.e. valid credentials
	if contextCredentials.Credential == "" {
		return nil, util.NewError("Error Insufficient Authentication", "No authentication credential provided")
	}

	if vpcp.Config.ServerConfig.DebugTrace {
		vpcp.APIConfig.DebugWriter = os.Stdout
	}

	if vpcp.ClientProvider == nil {
		vpcp.ClientProvider = riaas.DefaultRegionalAPIClientProvider{}
	}
	ctxLogger.Debug("", zap.Reflect("apiConfig.BaseURL", vpcp.APIConfig.BaseURL))

	if ctx != nil && ctx.Value(provider.RequestID) != nil {
		// set ContextID only of speicifed in the context
		vpcp.APIConfig.ContextID = fmt.Sprintf("%v", ctx.Value(provider.RequestID))
		ctxLogger.Info("", zap.Reflect("apiConfig.ContextID", vpcp.APIConfig.ContextID))
	}
	client, err := vpcp.ClientProvider.New(vpcp.APIConfig)
	if err != nil {
		return nil, err
	}

	// Create a token for all other API calls
	token, err := getAccessToken(contextCredentials, ctxLogger)
	if err != nil {
		return nil, err
	}
	ctxLogger.Debug("", zap.Reflect("Token", token.Token))

	err = client.Login(token.Token)
	if err != nil {
		return nil, err
	}

	// Update retry logic default values
	if vpcp.Config.VPCConfig.MaxRetryAttempt > 0 {
		ctxLogger.Debug("", zap.Reflect("MaxRetryAttempt", vpcp.Config.VPCConfig.MaxRetryAttempt))
		maxRetryAttempt = vpcp.Config.VPCConfig.MaxRetryAttempt
	}
	if vpcp.Config.VPCConfig.MaxRetryGap > 0 {
		ctxLogger.Debug("", zap.Reflect("MaxRetryGap", vpcp.Config.VPCConfig.MaxRetryGap))
		maxRetryGap = vpcp.Config.VPCConfig.MaxRetryGap
	}

	//Update retry logic for custom retry with default values
	/*
		Default MaxVPCRetryAttempt = 46 times(~7 mins), MinVPCRetryGap = 3sec , MinVPCRetryGapAttempt = 3sec
		1.) Honour the MinVPCRetryGap only if it is greater than 3 and less than 10 sec
		2.) Honour the MinVPCRetryGapAttempt only if it is greater than 0
		3.) Honour the MaxVPCRetryAttempt only if it is greater than 46 ( ~7 mins default)
	*/
	if vpcp.Config.VPCConfig.MinVPCRetryGap > ConstMinVPCRetryGap && vpcp.Config.VPCConfig.MinVPCRetryGap < ConstantRetryGap {
		minVPCRetryGap = vpcp.Config.VPCConfig.MinVPCRetryGap
	}

	if vpcp.Config.VPCConfig.MinVPCRetryGapAttempt > 0 {
		minVPCRetryGapAttempt = vpcp.Config.VPCConfig.MinVPCRetryGapAttempt
	}

	if vpcp.Config.VPCConfig.MaxVPCRetryAttempt > ConstMaxVPCRetryAttempt {
		maxVPCRetryAttempt = vpcp.Config.VPCConfig.MaxVPCRetryAttempt
	}

	ctxLogger.Info("VPC Retry details for WaitAttach and WaitDetach operations", zap.Reflect("MinVPCRetryGap", minVPCRetryGap), zap.Reflect("MinVPCRetryGapAttempt", minVPCRetryGapAttempt), zap.Reflect("MaxVPCRetryAttempt", maxVPCRetryAttempt))

	vpcSession := &VPCSession{
		VPCAccountID:          contextCredentials.IAMAccountID,
		Config:                vpcp.Config,
		ContextCredentials:    contextCredentials,
		VolumeType:            "vpc-block",
		Provider:              VPC,
		Apiclient:             client,
		APIClientVolAttachMgr: client.VolumeAttachService(),
		Logger:                ctxLogger,
		APIRetry:              NewFlexyRetryDefault(),
		SessionError:          nil,
	}
	return vpcSession, nil
}

// getAccessToken ...
func getAccessToken(creds provider.ContextCredentials, logger *zap.Logger) (token *iam.AccessToken, err error) {
	switch creds.AuthType {
	case provider.IAMAccessToken:
		token = &iam.AccessToken{Token: creds.Credential}
	default:
		err = errors.New("unknown AuthType")
	}
	return
}

// getPrivateEndpoint ...
func getPrivateEndpoint(logger *zap.Logger, publicEndPoint string) string {
	logger.Info("In getPrivateEndpoint, RIaaS public endpoint", zap.Reflect("URL", publicEndPoint))
	if !strings.Contains(publicEndPoint, PrivatePrefix) {
		if len(publicEndPoint) > HTTPSLength {
			return BasePrivateURL + publicEndPoint[HTTPSLength:]
		}
	} else {
		return publicEndPoint
	}
	return ""
}

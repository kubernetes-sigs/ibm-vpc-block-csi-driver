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

	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	utilReasonCode "github.com/IBM/ibmcloud-volume-interface/lib/utils/reasoncode"
	"github.com/IBM/ibmcloud-volume-interface/provider/local"
	vpcprovider "github.com/IBM/ibmcloud-volume-vpc/block/provider"
	vpcconfig "github.com/IBM/ibmcloud-volume-vpc/block/vpcconfig"
	vpcauth "github.com/IBM/ibmcloud-volume-vpc/common/auth"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/riaas"

	"go.uber.org/zap"
)

// IksVpcBlockProvider  handles both IKS and  RIAAS sessions
type IksVpcBlockProvider struct {
	vpcprovider.VPCBlockProvider
	vpcBlockProvider *vpcprovider.VPCBlockProvider // Holds VPC provider. Requires to avoid recursive calls
	iksBlockProvider *vpcprovider.VPCBlockProvider // Holds IKS provider
}

var _ local.Provider = &IksVpcBlockProvider{}

// NewProvider handles both IKS and  RIAAS sessions
func NewProvider(conf *vpcconfig.VPCBlockConfig, logger *zap.Logger) (local.Provider, error) {
	var err error
	//Setup vpc provider
	provider, err := vpcprovider.NewProvider(conf, logger)
	if err != nil {
		logger.Error("Error initializing VPC Provider", zap.Error(err))
		return nil, err
	}
	vpcBlockProvider, _ := provider.(*vpcprovider.VPCBlockProvider)

	// Setup IKS provider
	provider, err = vpcprovider.NewProvider(conf, logger)
	if err != nil {
		logger.Error("Error initializing IKS Provider", zap.Error(err))
		return nil, err
	}
	iksBlockProvider, _ := provider.(*vpcprovider.VPCBlockProvider)

	//Overrider Base URL
	iksBlockProvider.APIConfig.BaseURL = conf.VPCConfig.IKSTokenExchangePrivateURL
	// Setup IKS-VPC dual provider
	iksVpcBlockProvider := &IksVpcBlockProvider{
		VPCBlockProvider: *vpcBlockProvider,
		vpcBlockProvider: vpcBlockProvider,
		iksBlockProvider: iksBlockProvider,
	}

	iksVpcBlockProvider.iksBlockProvider.ContextCF, err = vpcauth.NewVPCContextCredentialsFactory(iksVpcBlockProvider.vpcBlockProvider.Config)
	if err != nil {
		logger.Error("Error initializing context credentials factory", zap.Error(err))
		return nil, err
	}
	//vpcBlockProvider.ApiConfig.BaseURL = conf.VPC.IKSTokenExchangePrivateURL
	return iksVpcBlockProvider, nil
}

// OpenSession opens a session on the provider
func (iksp *IksVpcBlockProvider) OpenSession(ctx context.Context, contextCredentials provider.ContextCredentials, ctxLogger *zap.Logger) (provider.Session, error) {
	ctxLogger.Info("Entering IksVpcBlockProvider.OpenSession")

	defer func() {
		ctxLogger.Debug("Exiting IksVpcBlockProvider.OpenSession")
	}()
	ctxLogger.Info("Opening VPC block session")
	ccf, _ := iksp.vpcBlockProvider.ContextCredentialsFactory(nil)
	ctxLogger.Info("Its IKS dual session. Getttng IAM token for  VPC block session")
	vpcContextCredentials, err := ccf.ForIAMAccessToken(iksp.iksBlockProvider.Config.VPCConfig.APIKey, ctxLogger)
	if err != nil {
		ctxLogger.Error("Error occurred while generating IAM token for VPC", zap.Error(err))
		if util.ErrorReasonCode(err) == utilReasonCode.EndpointNotReachable {
			userErr := userError.GetUserError(string(userError.EndpointNotReachable), err)
			return nil, userErr
		}
		if util.ErrorReasonCode(err) == utilReasonCode.Timeout {
			userErr := userError.GetUserError(string(userError.Timeout), err)
			return nil, userErr
		}
		return nil, err
	}
	session, err := iksp.vpcBlockProvider.OpenSession(ctx, vpcContextCredentials, ctxLogger)
	if err != nil {
		ctxLogger.Error("Error occurred while opening VPCSession", zap.Error(err))
		return nil, err
	}
	vpcSession, _ := session.(*vpcprovider.VPCSession)
	ctxLogger.Info("Opening IKS block session")

	ccf = iksp.iksBlockProvider.ContextCF
	iksp.iksBlockProvider.ClientProvider = riaas.IKSRegionalAPIClientProvider{}

	ctxLogger.Info("Its ISK dual session. Getttng IAM token for  IKS block session")
	iksContextCredentials, err := ccf.ForIAMAccessToken(iksp.iksBlockProvider.Config.VPCConfig.APIKey, ctxLogger)
	if err != nil {
		ctxLogger.Warn("Error occurred while generating IAM token for IKS. But continue with VPC session alone. \n Volume Mount operation will fail but volume provisioning will work", zap.Error(err))
		session = &vpcprovider.VPCSession{
			Logger:       ctxLogger,
			SessionError: err,
		} // Empty session to avoid Nil references.
	} else {
		session, err = iksp.iksBlockProvider.OpenSession(ctx, iksContextCredentials, ctxLogger)
		if err != nil {
			ctxLogger.Error("Error occurred while opening IKSSession", zap.Error(err))
		}
	}

	iksSession, ok := session.(*vpcprovider.VPCSession)
	if ok && iksSession.Apiclient != nil {
		iksSession.APIClientVolAttachMgr = iksSession.Apiclient.IKSVolumeAttachService()
	}
	// Setup Dual Session that handles for VPC and IKS connections
	vpcIksSession := IksVpcSession{
		VPCSession: *vpcSession,
		IksSession: iksSession,
	}
	ctxLogger.Debug("IksVpcSession", zap.Reflect("IksVpcSession", vpcIksSession))
	return &vpcIksSession, nil
}

// ContextCredentialsFactory ...
func (iksp *IksVpcBlockProvider) ContextCredentialsFactory(zone *string) (local.ContextCredentialsFactory, error) {
	return iksp.iksBlockProvider.ContextCF, nil
}

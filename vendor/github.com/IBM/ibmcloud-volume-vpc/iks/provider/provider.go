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
	"github.com/IBM/ibmcloud-volume-interface/provider/local"
	vpcprovider "github.com/IBM/ibmcloud-volume-vpc/block/provider"
	vpcconfig "github.com/IBM/ibmcloud-volume-vpc/block/vpcconfig"
	vpcauth "github.com/IBM/ibmcloud-volume-vpc/common/auth"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/riaas"

	"go.uber.org/zap"
)

//IksVpcBlockProvider  handles both IKS and  RIAAS sessions
type IksVpcBlockProvider struct {
	vpcprovider.VPCBlockProvider
	vpcBlockProvider *vpcprovider.VPCBlockProvider // Holds VPC provider. Requires to avoid recursive calls
	iksBlockProvider *vpcprovider.VPCBlockProvider // Holds IKS provider
}

var _ local.Provider = &IksVpcBlockProvider{}

//NewProvider handles both IKS and  RIAAS sessions
func NewProvider(conf *vpcconfig.VPCBlockConfig, logger *zap.Logger) (local.Provider, error) {
	//Setup vpc provider
	provider, _ := vpcprovider.NewProvider(conf, logger)
	vpcBlockProvider, _ := provider.(*vpcprovider.VPCBlockProvider)
	// Setup IKS provider
	provider, _ = vpcprovider.NewProvider(conf, logger)
	iksBlockProvider, _ := provider.(*vpcprovider.VPCBlockProvider)

	//Overrider Base URL
	iksBlockProvider.APIConfig.BaseURL = conf.VPCConfig.IKSTokenExchangePrivateURL
	// Setup IKS-VPC dual provider
	iksVpcBlockProvider := &IksVpcBlockProvider{
		VPCBlockProvider: *vpcBlockProvider,
		vpcBlockProvider: vpcBlockProvider,
		iksBlockProvider: iksBlockProvider,
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
		userErr := userError.GetUserError(string(userError.AuthenticationFailed), err)
		return nil, userErr
	}
	session, err := iksp.vpcBlockProvider.OpenSession(ctx, vpcContextCredentials, ctxLogger)
	if err != nil {
		ctxLogger.Error("Error occurred while opening VPCSession", zap.Error(err))
		return nil, err
	}
	vpcSession, _ := session.(*vpcprovider.VPCSession)
	ctxLogger.Info("Opening IKS block session")

	//Create ContextCredentialsFactory
	ccf, err = iksp.ContextCredentialsFactory(nil)
	if err != nil {
		ctxLogger.Error("Error while creating the ContextCredentialsFactory", zap.Error(err))
		return nil, err
	}
	iksp.iksBlockProvider.ContextCF = ccf
	iksp.iksBlockProvider.ClientProvider = riaas.IKSRegionalAPIClientProvider{}

	ctxLogger.Info("Its ISK dual session. Getttng IAM token for  IKS block session")
	iksContextCredentials, err := ccf.ForIAMAccessToken(iksp.iksBlockProvider.Config.VPCConfig.APIKey, ctxLogger)
	if err != nil {
		ctxLogger.Warn("Error occurred while generating IAM token for IKS. But continue with VPC session alone. \n Volume Mount operation will fail but volume provisioning will work", zap.Error(err))
		session = &vpcprovider.VPCSession{} // Empty session to avoid Nil references.
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
	return vpcauth.NewVPCContextCredentialsFactory(iksp.vpcBlockProvider.Config)
}

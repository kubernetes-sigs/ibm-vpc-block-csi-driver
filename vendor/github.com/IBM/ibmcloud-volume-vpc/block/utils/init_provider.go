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

// Package utils ...
package utils

import (
	"errors"

	"go.uber.org/zap"
	"golang.org/x/net/context"

	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/IBM/ibmcloud-volume-interface/provider/local"
	vpc_provider "github.com/IBM/ibmcloud-volume-vpc/block/provider"
	vpcconfig "github.com/IBM/ibmcloud-volume-vpc/block/vpcconfig"
	"github.com/IBM/ibmcloud-volume-vpc/common/registry"
	iks_vpc_provider "github.com/IBM/ibmcloud-volume-vpc/iks/provider"
)

// InitProviders initialization for all providers as per configurations
func InitProviders(conf *vpcconfig.VPCBlockConfig, logger *zap.Logger) (registry.Providers, error) {
	var haveProviders bool
	providerRegistry := &registry.ProviderRegistry{}

	// VPC provider registration
	if conf.VPCConfig != nil && conf.VPCConfig.Enabled {
		logger.Info("Configuring VPC Block Provider")
		prov, err := vpc_provider.NewProvider(conf, logger)
		if err != nil {
			logger.Info("VPC block provider error!")
			return nil, err
		}
		providerRegistry.Register(conf.VPCConfig.VPCBlockProviderName, prov)
		haveProviders = true
	}

	// IKS provider registration
	if conf.IKSConfig != nil && conf.IKSConfig.Enabled {
		logger.Info("Configuring IKS-VPC Block Provider")
		prov, err := iks_vpc_provider.NewProvider(conf, logger)
		if err != nil {
			logger.Info("VPC block provider error!")
			return nil, err
		}
		providerRegistry.Register(conf.IKSConfig.IKSBlockProviderName, prov)
		haveProviders = true
	}

	if haveProviders {
		logger.Info("Provider registration done!!!")
		return providerRegistry, nil
	}

	return nil, errors.New("no providers registered")
}

// OpenProviderSession ...
func OpenProviderSession(conf *vpcconfig.VPCBlockConfig, providers registry.Providers, providerID string, ctxLogger *zap.Logger) (session provider.Session, fatal bool, err error) {
	return OpenProviderSessionWithContext(context.TODO(), conf, providers, providerID, ctxLogger)
}

// OpenProviderSessionWithContext ...
func OpenProviderSessionWithContext(ctx context.Context, conf *vpcconfig.VPCBlockConfig, providers registry.Providers, providerID string, ctxLogger *zap.Logger) (session provider.Session, fatal bool, err error) {
	prov, err := providers.Get(providerID)
	if err != nil {
		ctxLogger.Error("Not able to get the said provider, might be its not registered", local.ZapError(err))
		fatal = true
		return
	}

	ccf, err := prov.ContextCredentialsFactory(nil)
	if err != nil {
		fatal = true
		return
	}
	ctxLogger.Info("Calling provider/utils/init_provider.go GenerateContextCredentials")
	contextCredentials, err := GenerateContextCredentials(conf, providerID, ccf, ctxLogger)
	if err == nil {
		session, err = prov.OpenSession(ctx, contextCredentials, ctxLogger)
	}

	if err != nil {
		fatal = true
		ctxLogger.Error("Failed to open provider session", local.ZapError(err), zap.Bool("Fatal", fatal))
	}
	return
}

// GenerateContextCredentials ...
func GenerateContextCredentials(conf *vpcconfig.VPCBlockConfig, providerID string, contextCredentialsFactory local.ContextCredentialsFactory, ctxLogger *zap.Logger) (provider.ContextCredentials, error) {
	ctxLogger.Info("Generating generateContextCredentials for ", zap.String("Provider ID", providerID))

	// Select appropriate authentication strategy
	switch {
	case (conf.VPCConfig != nil && providerID == conf.VPCConfig.VPCBlockProviderName):
		ctxLogger.Info("Calling provider/init_provider.go ForIAMAccessToken")
		return contextCredentialsFactory.ForIAMAccessToken(conf.VPCConfig.APIKey, ctxLogger)

	case (conf.IKSConfig != nil && providerID == conf.IKSConfig.IKSBlockProviderName):
		return provider.ContextCredentials{}, nil // Get credentials  in OpenSession method

	default:
		return provider.ContextCredentials{}, util.NewError("ErrorInsufficientAuthentication",
			"Insufficient authentication credentials")
	}
}

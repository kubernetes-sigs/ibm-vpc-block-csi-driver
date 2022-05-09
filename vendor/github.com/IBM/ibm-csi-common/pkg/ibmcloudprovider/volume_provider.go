/**
 * Copyright 2021 IBM Corp.
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

// Package ibmcloudprovider ...
package ibmcloudprovider

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/IBM/ibm-csi-common/pkg/messages"
	"github.com/IBM/ibm-csi-common/pkg/utils"
	"github.com/IBM/ibmcloud-volume-interface/config"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	"github.com/IBM/ibmcloud-volume-interface/lib/utils/reasoncode"
	"github.com/IBM/ibmcloud-volume-interface/provider/local"
	provider_util "github.com/IBM/ibmcloud-volume-vpc/block/utils"
	vpcconfig "github.com/IBM/ibmcloud-volume-vpc/block/vpcconfig"
	"github.com/IBM/ibmcloud-volume-vpc/common/registry"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// IBMCloudStorageProvider Provider
type IBMCloudStorageProvider struct {
	ProviderName   string
	ProviderConfig *config.Config
	Registry       registry.Providers
	ClusterInfo    *utils.ClusterInfo
}

var _ CloudProviderInterface = &IBMCloudStorageProvider{}

// NewIBMCloudStorageProvider ...
func NewIBMCloudStorageProvider(configPath string, logger *zap.Logger) (*IBMCloudStorageProvider, error) {
	logger.Info("NewIBMCloudStorageProvider-Reading provider configuration...")
	// Load config file
	conf, err := config.ReadConfig(configPath, logger)
	if err != nil {
		logger.Fatal("Error loading configuration")
		return nil, err
	}

	// Decode g2 API Key if it is a satellite cluster.(unmanaged cluster)
	if os.Getenv(strings.ToUpper("IKS_ENABLED")) != "True" && os.Getenv(strings.ToUpper("IS_SATELLITE")) == "True" {
		logger.Info("Decoding apiKey since its a satellite cluster")
		apiKey, err := base64.StdEncoding.DecodeString(conf.VPC.G2APIKey)
		if err != nil {
			return nil, err
		}
		conf.VPC.G2APIKey = string(apiKey)
	}

	// Correct if the G2EndpointURL is of the form "http://".
	conf.VPC.G2EndpointURL = getEndpointURL(conf.VPC.G2EndpointURL, logger)

	// Correct if the G2TokenExchangeURL is of the form "http://"
	conf.VPC.G2TokenExchangeURL = getEndpointURL(conf.VPC.G2TokenExchangeURL, logger)

	// Get only VPC_API_VERSION, in "2019-07-02T00:00:00.000Z" case vpc need only 2019-07-02"
	dateTime, err := time.Parse(time.RFC3339, conf.VPC.APIVersion)
	if err == nil {
		conf.VPC.APIVersion = fmt.Sprintf("%d-%02d-%02d", dateTime.Year(), dateTime.Month(), dateTime.Day())
	} else {
		logger.Warn("Failed to parse VPC_API_VERSION, setting default value")
		conf.VPC.APIVersion = "2020-07-02" // setting default values
	}

	var clusterInfo = &utils.ClusterInfo{}
	logger.Info("Fetching clusterInfo")
	if conf.IKS != nil && conf.IKS.Enabled || os.Getenv("IKS_ENABLED") == "True" {
		clusterInfo, err = utils.NewClusterInfo(logger)
		if err != nil {
			logger.Fatal("Unable to load ClusterInfo", local.ZapError(err))
			return nil, err
		}
		logger.Info("Fetched clusterInfo..")
		if conf.Bluemix.Encryption || conf.VPC.Encryption {
			// api Key if encryption is enabled
			logger.Info("Creating NewAPIKeyImpl...")
			apiKeyImp, err := utils.NewAPIKeyImpl(logger)
			if err != nil {
				logger.Fatal("Unable to create API key getter", local.ZapError(err))
				return nil, err
			}
			logger.Info("Created NewAPIKeyImpl...")
			err = apiKeyImp.UpdateIAMKeys(conf)
			if err != nil {
				logger.Fatal("Unable to get API key", local.ZapError(err))
				return nil, err
			}
		}
	}

	// Update the CSRF  Token
	if conf.Bluemix.PrivateAPIRoute != "" {
		conf.Bluemix.CSRFToken = string([]byte{}) // TODO~ Need to remove it
	}

	if conf.API == nil {
		conf.API = &config.APIConfig{
			PassthroughSecret: string([]byte{}), // // TODO~ Need to remove it
		}
	}
	vpcBlockConfig := &vpcconfig.VPCBlockConfig{
		VPCConfig:    conf.VPC,
		IKSConfig:    conf.IKS,
		APIConfig:    conf.API,
		ServerConfig: conf.Server,
	}
	// Prepare provider registry
	registry, err := provider_util.InitProviders(vpcBlockConfig, logger)
	if err != nil {
		logger.Fatal("Error configuring providers", local.ZapError(err))
	}

	var providerName string
	if isRunningInIKS() && conf.IKS.Enabled {
		providerName = conf.IKS.IKSBlockProviderName
	} else if conf.VPC.Enabled {
		providerName = conf.VPC.VPCBlockProviderName
	}

	cloudProvider := &IBMCloudStorageProvider{
		ProviderName:   providerName,
		ProviderConfig: conf,
		Registry:       registry,
		ClusterInfo:    clusterInfo,
	}
	logger.Info("Successfully read provider configuration")
	return cloudProvider, nil
}

func isRunningInIKS() bool {
	return true //TODO Check the master KUBE version
}

// GetProviderSession ...
func (icp *IBMCloudStorageProvider) GetProviderSession(ctx context.Context, logger *zap.Logger) (provider.Session, error) {
	logger.Info("IBMCloudStorageProvider-GetProviderSession...")
	if icp.ProviderConfig.API == nil {
		icp.ProviderConfig.API = &config.APIConfig{
			PassthroughSecret: string([]byte{}), // // TODO~ Need to remove it
		}
	}

	prov, err := icp.Registry.Get(icp.ProviderName)
	if err != nil {
		logger.Error("Not able to get the said provider, might be its not registered", local.ZapError(err))
		return nil, err
	}

	// Populating vpcBlockConfig which is used to open session
	vpcBlockConfig := &vpcconfig.VPCBlockConfig{
		VPCConfig:    icp.ProviderConfig.VPC,
		IKSConfig:    icp.ProviderConfig.IKS,
		APIConfig:    icp.ProviderConfig.API,
		ServerConfig: icp.ProviderConfig.Server,
	}

	for retryCount := 0; retryCount < utils.MaxRetryAttemptForSessions; retryCount++ {
		session, _, err := provider_util.OpenProviderSessionWithContext(ctx, prov, vpcBlockConfig, icp.ProviderName, logger)
		if err == nil {
			logger.Info("Successfully got the provider session", zap.Reflect("ProviderName", session.ProviderName()))
			return session, nil
		}
		logger.Error("Failed to get provider session", zap.Reflect("Error", err))
		// In the second retry, if there's an error, it will be returned without the need for validating it further.
		if retryCount == 1 {
			return nil, err
		}
		// If the error is related to invalid api key or invalid user, update api key will be called
		if providerError, ok := err.(provider.Error); ok && providerError.Code() == reasoncode.ErrorFailedTokenExchange && (strings.Contains(strings.ToLower(providerError.Error()), messages.APIKeyNotFound) || strings.Contains(strings.ToLower(providerError.Error()), messages.UserNotFound)) {
			// Waiting for minute expecting the API key to be updated in config
			time.Sleep(time.Minute * 1)
			err := icp.UpdateAPIKey(logger)
			if err != nil {
				logger.Error("Failed to update api key in cloud storage provider", zap.Error(err))
				return nil, err
			}
			// Updating the vpc block config with the newly read api key which is further used open provider session in the 2nd retry
			vpcBlockConfig.VPCConfig.APIKey = icp.ProviderConfig.VPC.G2APIKey
			vpcBlockConfig.VPCConfig.G2APIKey = icp.ProviderConfig.VPC.G2APIKey
			// Continuing the open provider session in next attempt after updating the api key
			continue
		}
		// returning error if it isn't related to invalid api key/user
		return nil, err
	}
	return nil, errors.New(messages.ErrAPIKeyNotFound)
}

// GetConfig ...
func (icp *IBMCloudStorageProvider) GetConfig() *config.Config {
	return icp.ProviderConfig
}

// GetClusterInfo ...
func (icp *IBMCloudStorageProvider) GetClusterInfo() *utils.ClusterInfo {
	return icp.ClusterInfo
}

// UpdateAPIKey ...
func (icp *IBMCloudStorageProvider) UpdateAPIKey(logger *zap.Logger) error {
	logger.Info("Updating API key in cloud storage provider")
	// Populating vpc block config structure, which will be used for updating iks and vpc block provider
	vpcBlockConfig := &vpcconfig.VPCBlockConfig{
		VPCConfig:    icp.ProviderConfig.VPC,
		IKSConfig:    icp.ProviderConfig.IKS,
		APIConfig:    icp.ProviderConfig.API,
		ServerConfig: icp.ProviderConfig.Server,
	}
	// Storing a backup of the existing api key, to make sure the newly read api key isn't the same as the old one
	// Hence avoiding fetching session with the same api key again
	vpcAPIKey := vpcBlockConfig.VPCConfig.G2APIKey

	if icp.ProviderConfig.IKS != nil && (icp.ProviderConfig.IKS.Enabled || os.Getenv("IKS_ENABLED") == "True") && icp.ProviderConfig.VPC.Encryption {
		apiKeyImp, err := utils.NewAPIKeyImpl(logger)
		if err != nil {
			logger.Error("Unable to create API key getter", zap.Reflect("Error", err))
			return err
		}
		logger.Info("Created NewAPIKeyImpl...")
		// Call to update cloud storage provider with api key
		err = apiKeyImp.UpdateIAMKeys(icp.ProviderConfig)
		if err != nil {
			logger.Error("Unable to get API key", local.ZapError(err))
			return err
		}
		// If the retrieved API key is the same as previous one, return error
		if vpcAPIKey == icp.ProviderConfig.VPC.G2APIKey {
			logger.Error("API key is not reset")
			return errors.New(messages.ErrAPIKeyNotFound)
		}
		// Updating the api key in vpc block config which will further be used to update the provider
		vpcBlockConfig.VPCConfig.APIKey = icp.ProviderConfig.VPC.G2APIKey
		vpcBlockConfig.VPCConfig.G2APIKey = icp.ProviderConfig.VPC.G2APIKey
	} else {
		// Reading config again to read the api key
		conf := new(config.Config)
		configPath := filepath.Join(config.GetConfPathDir(), utils.ConfigFileName)
		_, err := toml.DecodeFile(configPath, conf)
		if err != nil {
			logger.Error("Failed to parse config file", zap.Error(err))
			return err
		}

		// If the retrieved API key is the same as previous one, return error
		if vpcAPIKey == conf.VPC.G2APIKey {
			logger.Error("API is not reset")
			return errors.New(messages.ErrAPIKeyNotFound)
		}
		// Updating the api key in cloud storage provider and vpc block config which will further be used to update the provider
		icp.ProviderConfig.VPC.APIKey = conf.VPC.APIKey
		icp.ProviderConfig.VPC.G2APIKey = conf.VPC.G2APIKey
		vpcBlockConfig.VPCConfig.APIKey = conf.VPC.G2APIKey
		vpcBlockConfig.VPCConfig.G2APIKey = conf.VPC.G2APIKey
	}

	prov, err := icp.Registry.Get(icp.ProviderName)
	if err != nil {
		logger.Error("Not able to get the said provider, it might not registered", local.ZapError(err))
		return errors.New(messages.ErrUpdatingAPIKey)
	}

	// Updating the api key in provider using the updated vpc block config
	err = prov.UpdateAPIKey(vpcBlockConfig, logger)
	if err != nil {
		logger.Error("Failed to update API key in the provider", local.ZapError(err))
		return errors.New(messages.ErrUpdatingAPIKey)
	}

	return nil
}

// CorrectEndpointURL corrects endpoint url if it is of form "http://"
func getEndpointURL(url string, logger *zap.Logger) string {
	if strings.Contains(url, "http://") {
		logger.Warn("Token exchange endpoint URL is of the form 'http' instead 'https'. Correcting it for valid request.", zap.Reflect("Endpoint URL: ", url))
		return strings.Replace(url, "http", "https", 1)
	}
	return url
}

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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/IBM/ibmcloud-volume-interface/config"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	"github.com/IBM/ibmcloud-volume-interface/provider/local"
	provider_util "github.com/IBM/ibmcloud-volume-vpc/block/utils"
	vpcconfig "github.com/IBM/ibmcloud-volume-vpc/block/vpcconfig"
	"github.com/IBM/ibmcloud-volume-vpc/common/registry"
	utilsConfig "github.com/IBM/secret-utils-lib/pkg/config"
	"github.com/IBM/secret-utils-lib/pkg/k8s_utils"
	sp "github.com/IBM/secret-utils-lib/pkg/secret_provider"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// IBMCloudStorageProvider Provider
type IBMCloudStorageProvider struct {
	ProviderName   string
	ProviderConfig *config.Config
	Registry       registry.Providers
	ClusterID      string
}

var _ CloudProviderInterface = &IBMCloudStorageProvider{}

// NewIBMCloudStorageProvider ...
func NewIBMCloudStorageProvider(clusterVolumeLabel string, k8sClient k8s_utils.KubernetesClient, spObject sp.SecretProviderInterface, logger *zap.Logger) (*IBMCloudStorageProvider, error) {
	logger.Info("NewIBMCloudStorageProvider-Reading provider configuration...")
	// Load config file
	conf, err := config.ReadConfig(k8sClient, logger)
	if err != nil {
		logger.Error("Error loading configuration")
		return nil, err
	}

	// Get only VPC_API_VERSION, in "2019-07-02T00:00:00.000Z" case vpc need only 2019-07-02"
	dateTime, err := time.Parse(time.RFC3339, conf.VPC.APIVersion)
	if err == nil {
		conf.VPC.APIVersion = fmt.Sprintf("%d-%02d-%02d", dateTime.Year(), dateTime.Month(), dateTime.Day())
	} else {
		logger.Warn("Failed to parse VPC_API_VERSION, setting default value")
		conf.VPC.APIVersion = "2022-11-11" // setting default values
	}

	var clusterInfo utilsConfig.ClusterConfig
	logger.Info("Fetching clusterInfo")
	if conf.IKS != nil && conf.IKS.Enabled || os.Getenv("IKS_ENABLED") == "True" {
		clusterInfo, err = utilsConfig.GetClusterInfo(k8sClient, logger)
		if err != nil {
			logger.Error("Unable to load ClusterInfo", local.ZapError(err))
			return nil, err
		}
		logger.Info("Fetched clusterInfo..")
	}

	//Initialize the clusterVolumeLabel once which will be used for tagging by the library.
	conf.VPC.ClusterVolumeLabel = clusterVolumeLabel

	vpcBlockConfig := &vpcconfig.VPCBlockConfig{
		VPCConfig:    conf.VPC,
		IKSConfig:    conf.IKS,
		APIConfig:    conf.API,
		ServerConfig: conf.Server,
	}
	// Prepare provider registry
	registry, err := provider_util.InitProviders(vpcBlockConfig, spObject, logger)
	if err != nil {
		logger.Error("Error configuring providers", local.ZapError(err))
		return nil, err
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
		ClusterID:      clusterInfo.ClusterID,
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

	session, _, err := provider_util.OpenProviderSessionWithContext(ctx, prov, vpcBlockConfig, icp.ProviderName, logger)
	if err == nil {
		logger.Info("Successfully got the provider session", zap.Reflect("ProviderName", session.ProviderName()))
		return session, nil
	}
	logger.Error("Failed to get provider session", zap.Reflect("Error", err))
	return nil, err
}

// GetConfig ...
func (icp *IBMCloudStorageProvider) GetConfig() *config.Config {
	return icp.ProviderConfig
}

// GetClusterID ...
func (icp *IBMCloudStorageProvider) GetClusterID() string {
	return icp.ClusterID
}

// CorrectEndpointURL corrects endpoint url if it is of form "http://"
func getEndpointURL(url string, logger *zap.Logger) string {
	if strings.Contains(url, "http://") {
		logger.Warn("Token exchange endpoint URL is of the form 'http' instead 'https'. Correcting it for valid request.", zap.Reflect("Endpoint URL: ", url))
		return strings.Replace(url, "http", "https", 1)
	}
	return url
}

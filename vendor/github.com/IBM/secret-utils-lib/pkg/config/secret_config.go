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

package config

import (
	"github.com/IBM/secret-utils-lib/pkg/utils"

	"github.com/BurntSushi/toml"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
)

// Config is the parent struct for all the configuration information for -cluster
type Config struct {
	Server    *ServerConfig  `required:"true"`
	Bluemix   *BluemixConfig //`required:"true"`
	Softlayer *SoftlayerConfig
	VPC       *VPCProviderConfig
	IKS       *IKSConfig
	API       *APIConfig
}

// ServerConfig configuration options for the provider server itself
type ServerConfig struct {
	// DebugTrace is a flag to enable the debug level trace within the provider code.
	DebugTrace bool `toml:"debug_trace" envconfig:"DEBUG_TRACE"`
}

// BluemixConfig ...
type BluemixConfig struct {
	IamURL          string `toml:"iam_url"`
	IamClientID     string `toml:"iam_client_id"`
	IamClientSecret string `toml:"iam_client_secret" json:"-"`
	IamAPIKey       string `toml:"iam_api_key" json:"-"`
	RefreshToken    string `toml:"refresh_token" json:"-"`
	APIEndpointURL  string `toml:"containers_api_route"`
	PrivateAPIRoute string `toml:"containers_api_route_private"`
	Encryption      bool   `toml:"encryption"`
	CSRFToken       string `toml:"containers_api_csrf_token" json:"-"`
}

// SoftlayerConfig ...
type SoftlayerConfig struct {
	SoftlayerBlockEnabled        bool   `toml:"softlayer_block_enabled" envconfig:"SOFTLAYER_BLOCK_ENABLED"`
	SoftlayerBlockProviderName   string `toml:"softlayer_block_provider_name" envconfig:"SOFTLAYER_BLOCK_PROVIDER_NAME"`
	SoftlayerFileEnabled         bool   `toml:"softlayer_file_enabled" envconfig:"SOFTLAYER_FILE_ENABLED"`
	SoftlayerFileProviderName    string `toml:"softlayer_file_provider_name" envconfig:"SOFTLAYER_FILE_PROVIDER_NAME"`
	SoftlayerUsername            string `toml:"softlayer_username" json:"-"`
	SoftlayerAPIKey              string `toml:"softlayer_api_key" json:"-"`
	SoftlayerEndpointURL         string `toml:"softlayer_endpoint_url"`
	SoftlayerDataCenter          string `toml:"softlayer_datacenter"`
	SoftlayerTimeout             string `toml:"softlayer_api_timeout" envconfig:"SOFTLAYER_API_TIMEOUT"`
	SoftlayerVolProvisionTimeout string `toml:"softlayer_vol_provision_timeout" envconfig:"SOFTLAYER_VOL_PROVISION_TIMEOUT"`
	SoftlayerRetryInterval       string `toml:"softlayer_api_retry_interval" envconfig:"SOFTLAYER_API_RETRY_INTERVAL"`
	SoftlayerTokenExchangeURL    string `toml:"softlayer_token_exchange_endpoint_url"`
	//Configuration values for JWT tokens
	SoftlayerJWTKID       string `toml:"softlayer_jwt_kid"`
	SoftlayerJWTTTL       int    `toml:"softlayer_jwt_ttl"`
	SoftlayerJWTValidFrom int    `toml:"softlayer_jwt_valid"`

	SoftlayerIMSEndpointURL string `toml:"softlayer_iam_endpoint_url"`
	SoftlayerAPIDebug       bool
}

// VPCProviderConfig configures a specific instance of a VPC provider (e.g. GT/GC/Z)
type VPCProviderConfig struct {
	Enabled bool `toml:"vpc_enabled" envconfig:"VPC_ENABLED"`

	IamClientID     string `toml:"iam_client_id"`
	IamClientSecret string `toml:"iam_client_secret" json:"-"`

	//valid values (gc|g2), if unspecified, GC will take precedence(if both are specified)
	//during e2e test, user can specify its own preferred type during execution
	VPCTypeEnabled       string `toml:"vpc_type_enabled" envconfig:"VPC_TYPE_ENABLED"`
	VPCBlockProviderName string `toml:"vpc_block_provider_name" envconfig:"VPC_BLOCK_PROVIDER_NAME"`
	VPCBlockProviderType string `toml:"provider_type"`
	VPCVolumeType        string `toml:"vpc_volume_type" envconfig:"VPC_VOLUME_TYPE"`

	EndpointURL        string `toml:"gc_riaas_endpoint_url"`
	PrivateEndpointURL string `toml:"gc_riaas_endpoint_private_url"`
	TokenExchangeURL   string `toml:"gc_token_exchange_endpoint_url"`
	APIKey             string `toml:"gc_api_key" json:"-"`
	ResourceGroupID    string `toml:"gc_resource_group_id"`
	VPCAPIGeneration   int    `toml:"vpc_api_generation" envconfig:"VPC_API_GENERATION"`
	APIVersion         string `toml:"api_version,omitempty" envconfig:"VPC_API_VERSION"`

	//NG Properties
	G2EndpointURL        string `toml:"g2_riaas_endpoint_url"`
	G2EndpointPrivateURL string `toml:"g2_riaas_endpoint_private_url"`
	G2TokenExchangeURL   string `toml:"g2_token_exchange_endpoint_url"`
	G2APIKey             string `toml:"g2_api_key" json:"-"`
	G2ResourceGroupID    string `toml:"g2_resource_group_id"`
	G2VPCAPIGeneration   int    `toml:"g2_vpc_api_generation" envconfig:"G2_VPC_API_GENERATION"`
	G2APIVersion         string `toml:"g2_api_version,omitempty" envconfig:"G2_VPC_API_VERSION"`

	Encryption      bool   `toml:"encryption"`
	VPCTimeout      string `toml:"vpc_api_timeout,omitempty" envconfig:"VPC_API_TIMEOUT"`
	MaxRetryAttempt int    `toml:"max_retry_attempt,omitempty" envconfig:"VPC_RETRY_ATTEMPT"`
	MaxRetryGap     int    `toml:"max_retry_gap,omitempty" envconfig:"VPC_RETRY_INTERVAL"`
	// IKSTokenExchangePrivateURL, for private cluster support hence using for all cluster types
	IKSTokenExchangePrivateURL string `toml:"iks_token_exchange_endpoint_private_url"`

	IsIKS bool `toml:"is_iks,omitempty"`
}

//IKSConfig config
type IKSConfig struct {
	Enabled              bool   `toml:"iks_enabled" envconfig:"IKS_ENABLED"`
	IKSBlockProviderName string `toml:"iks_block_provider_name" envconfig:"IKS_BLOCK_PROVIDER_NAME"`
}

// APIConfig config
type APIConfig struct {
	PassthroughSecret string `toml:"PassthroughSecret" json:"-"`
}

// ParseConfig loads the config from file
func ParseConfig(logger *zap.Logger, data string) (*Config, error) {
	configData := new(Config)
	_, err := toml.Decode(data, configData)
	if err != nil {
		logger.Error("Failed to parse config", zap.Error(err))
		return nil, utils.Error{Description: utils.ErrParsingConfig, BackendError: err.Error()}
	}

	err = envconfig.Process("", configData)
	if err != nil {
		logger.Error("Failed to gather environment config variable", zap.Error(err))
		return nil, utils.Error{Description: utils.ErrFetchingENV, BackendError: err.Error()}
	}

	return configData, nil
}

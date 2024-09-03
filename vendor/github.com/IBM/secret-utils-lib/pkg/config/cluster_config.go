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
	"encoding/json"
	"strings"

	"github.com/IBM/secret-utils-lib/pkg/k8s_utils"
	"github.com/IBM/secret-utils-lib/pkg/utils"
	"go.uber.org/zap"
)

const (
	// clusterInfo ...
	clusterInfoCM = "cluster-info"
	// clusterConfigName ...
	clusterConfigName = "cluster-config.json"
	// stageMasterURLsubstr ...
	stageMasterURLsubstr = ".test."
	// tokenExchangePath ...
	tokenExchangePath = "/identity/token"
	// constTrue ...
	constTrue = "True"
)

// ClusterConfig ...
type ClusterConfig struct {
	ClusterID       string `json:"cluster_id"`
	MasterURL       string `json:"master_url"`
	ClusterProvider string `json:"cluster_provider"`
	ClusterType     string `json:"cluster_type"`
}

// GetClusterInfo ...
func GetClusterInfo(kc k8s_utils.KubernetesClient, logger *zap.Logger) (ClusterConfig, error) {
	data, err := k8s_utils.GetConfigMapData(kc, clusterInfoCM, clusterConfigName)
	var cc ClusterConfig
	if err != nil {
		logger.Error("Error fetching cluster info", zap.Error(err))
		return cc, err
	}

	err = json.Unmarshal([]byte(data), &cc)
	if err != nil {
		logger.Error("Error fetching cluster-info configmap", zap.Error(err))
		return cc, utils.Error{Description: utils.ErrFetchingClusterConfig, BackendError: err.Error()}
	}

	return cc, nil
}

// FrameTokenExchangeURL ...
func FrameTokenExchangeURL(kc k8s_utils.KubernetesClient, providerType string, logger *zap.Logger) (string, bool) {

	var isURLprovided = true
	// Fetch token exchange URL from cloud-conf
	cloudConf, err := GetCloudConf(logger, kc)
	if err == nil && cloudConf.TokenExchangeURL != "" {
		return cloudConf.TokenExchangeURL + tokenExchangePath, isURLprovided
	}

	logger.Info("Unable to fetch token exchange URL from cloud-conf")
	clusterInfo, err := GetClusterInfo(kc, logger)
	if err != nil {
		logger.Error("Error fetching cluster info", zap.Error(err))
		return (utils.ProdPrivateIAMURL + tokenExchangePath), !isURLprovided
	}

	secret, err := k8s_utils.GetSecretData(kc, utils.STORAGE_SECRET_STORE_SECRET, utils.SECRET_STORE_FILE)
	if err == nil {
		if secretConfig, err := ParseConfig(logger, secret); err == nil {
			url, isURLprovided, err := GetTokenExchangeURLfromStorageSecretStore(clusterInfo, *secretConfig, providerType)
			if err == nil {
				return url, isURLprovided
			}
		}
	}

	logger.Info("Unable to fetch token exchange URL using secret, forming url using cluster info")
	return FrameTokenExchangeURLFromClusterInfo(clusterInfo, logger)
}

// GetTokenExchangeURLfromStorageSecretStore ...
func GetTokenExchangeURLfromStorageSecretStore(clusterInfo ClusterConfig, config Config, providerType string) (string, bool, error) {

	// Return Private Prod/Stage IAM URL if the cluster is VPC Gen2
	var isURLprovided = false
	if GetIAASProvider(clusterInfo) == utils.VPCGen2 {
		if isEndpointPrivate(config.VPC.G2TokenExchangeURL) {
			isURLprovided = true
			return config.VPC.G2TokenExchangeURL + tokenExchangePath, isURLprovided, nil
		}
		isURLprovided = false
		if isProduction(config.VPC.G2TokenExchangeURL) {
			return utils.ProdPrivateIAMURL + tokenExchangePath, isURLprovided, nil
		}
		return utils.StagePrivateIAMURL + tokenExchangePath, isURLprovided, nil
	}

	// If the cluster is satellite, classic, IPI, return the URL provided in storage-secret-store
	isURLprovided = true
	var url string
	switch providerType {
	case utils.VPC:
		url = config.VPC.G2TokenExchangeURL
	case utils.Bluemix:
		url = config.Bluemix.IamURL
	case utils.Softlayer:
		url = config.Softlayer.SoftlayerTokenExchangeURL
	}

	if url == "" {
		return "", isURLprovided, utils.Error{Description: utils.WarnFetchingTokenExchangeURL}
	}

	return url, isURLprovided, nil
}

// FrameTokenExchangeURLFromClusterInfo ...
func FrameTokenExchangeURLFromClusterInfo(cc ClusterConfig, logger *zap.Logger) (string, bool) {

	var isURLprovided = true
	isSatellite := IsSatellite(cc, logger)
	isProd := isProduction(cc.MasterURL)
	switch {
	case isSatellite && isProd:
		return (utils.ProdPublicIAMURL + tokenExchangePath), isURLprovided
	case isSatellite && !isProd:
		return (utils.StagePublicIAMURL + tokenExchangePath), isURLprovided
	case !isSatellite && isProd:
		return (utils.ProdPrivateIAMURL + tokenExchangePath), !isURLprovided
	case !isSatellite && !isProd:
		return (utils.StagePrivateIAMURL + tokenExchangePath), !isURLprovided
	}

	return (utils.ProdPrivateIAMURL + tokenExchangePath), !isURLprovided
}

// isEndpointPrivate determines if the provided url is private or public endpoint
func isEndpointPrivate(url string) bool {
	if strings.Contains(url, "private") {
		return true
	}
	return false
}

// isProduction determines if the env in which a pod is deployed is stage or production
func isProduction(url string) bool {
	if !strings.Contains(url, "stage") && !strings.Contains(url, "test") {
		return true
	}
	return false
}

// IsSatellite checks if the cluster where the pod is currently running is a satellite cluster or not
func IsSatellite(cc ClusterConfig, logger *zap.Logger) bool {
	if cc.ClusterProvider == utils.SatelliteProvider {
		return true
	}

	return false
}

// GetIAASProvider ...
func GetIAASProvider(cc ClusterConfig) string {
	return cc.ClusterType
}

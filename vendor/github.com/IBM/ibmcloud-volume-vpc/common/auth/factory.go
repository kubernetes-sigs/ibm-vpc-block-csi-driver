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

// Package auth ...
package auth

import (
	"github.com/IBM/ibmcloud-volume-interface/provider/auth"
	"github.com/IBM/ibmcloud-volume-interface/provider/iam"
	vpcconfig "github.com/IBM/ibmcloud-volume-vpc/block/vpcconfig"
	vpciam "github.com/IBM/ibmcloud-volume-vpc/common/iam"
	"github.com/IBM/secret-utils-lib/pkg/k8s_utils"
)

// NewVPCContextCredentialsFactory ...
func NewVPCContextCredentialsFactory(config *vpcconfig.VPCBlockConfig, k8sClient *k8s_utils.KubernetesClient) (*auth.ContextCredentialsFactory, error) {
	authConfig := &iam.AuthConfiguration{
		IamURL:          config.VPCConfig.G2TokenExchangeURL,
		IamClientID:     config.VPCConfig.IamClientID,
		IamClientSecret: config.VPCConfig.IamClientSecret,
	}
	ccf, err := auth.NewContextCredentialsFactory(authConfig, k8sClient, iam.VPC)
	if err != nil {
		return nil, err
	}
	if config.VPCConfig.IKSTokenExchangePrivateURL != "" {
		authIKSConfig := &vpciam.IksAuthConfiguration{
			IamAPIKey:       config.VPCConfig.G2APIKey,
			PrivateAPIRoute: config.VPCConfig.IKSTokenExchangePrivateURL, // Only for private cluster
			CSRFToken:       config.APIConfig.PassthroughSecret,          // required for private cluster
		}
		ccf.TokenExchangeService, err = vpciam.NewTokenExchangeIKSService(authIKSConfig, k8sClient)
		if err != nil {
			return nil, err
		}
	}
	return ccf, nil
}

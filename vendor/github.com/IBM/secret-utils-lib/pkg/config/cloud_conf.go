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

	k8s_utils "github.com/IBM/secret-utils-lib/pkg/k8s_utils"
	"go.uber.org/zap"
)

const (
	cloudConfCM   = "cloud-conf"
	cloudConfData = "cloud-conf.json"
)

type CloudConf struct {
	Region                   string `json:"region"`
	RiaasEndpoint            string `json:"riaas_endpoint"`
	PrivateRIAASEndpoint     string `json:"riaas_private_endpoint"`
	ContainerAPIRoute        string `json:"containers_api_route"`
	PrivateContainerAPIRoute string `json:"containers_api_route_private"`
	ResourceGroupID          string `json:"resource_group_id"`
	TokenExchangeURL         string `json:"token_exchange_url"`
}

// GetCloudConf ...
func GetCloudConf(logger *zap.Logger, k8sClient k8s_utils.KubernetesClient) (CloudConf, error) {
	var cloudConf CloudConf
	data, err := k8s_utils.GetConfigMapData(k8sClient, cloudConfCM, cloudConfData)
	if err != nil {
		return cloudConf, err
	}

	err = json.Unmarshal([]byte(data), &cloudConf)
	return cloudConf, err
}

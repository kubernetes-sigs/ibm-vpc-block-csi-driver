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

// Package utils ...
package utils

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/IBM/ibmcloud-volume-interface/config"
	"go.uber.org/zap"
)

//ClusterInfo contains the cluster information
type ClusterInfo struct {
	ClusterID   string `json:"cluster_id"`
	ClusterName string `json:"cluster_name,omitempty"`
	DataCenter  string `json:"datacenter,omitempty"`
	CustomerID  string `json:"customer_id,omitempty"`
}

//NewClusterInfo loads cluster info
func NewClusterInfo(logger *zap.Logger) (*ClusterInfo, error) {
	configBasePath := config.GetConfPathDir()
	clusterInfo := &ClusterInfo{}
	clusterInfoFile := filepath.Join(configBasePath, ClusterInfoPath)
	clusterInfoContent, err := ioutil.ReadFile(filepath.Clean(clusterInfoFile))
	if err != nil {
		logger.Error("Error while reading  cluster-config.json", zap.Error(err))
		return nil, err
	}
	err = json.Unmarshal(clusterInfoContent, clusterInfo)
	if err != nil {
		logger.Error("Error while parsing cluster-config", zap.Error(err))
		return nil, err
	}
	return clusterInfo, nil
}

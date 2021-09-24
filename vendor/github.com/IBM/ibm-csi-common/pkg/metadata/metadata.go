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

// Package metadata ...
package metadata

import (
	"fmt"

	"github.com/IBM/ibm-csi-common/pkg/utils"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// NodeMetadata is a fakeable interface exposing necessary data
type NodeMetadata interface {
	// GetZone ...
	GetZone() string

	// GetRegion ...
	GetRegion() string

	// GetWorkerID ...
	GetWorkerID() string
}

type nodeMetadataManager struct {
	zone     string
	region   string
	workerID string
}

var _ NodeMetadata = &nodeMetadataManager{}

// NewNodeMetadata ...
func NewNodeMetadata(nodeName string, logger *zap.Logger) (NodeMetadata, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	node, err := clientset.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	nodeLabels := node.ObjectMeta.Labels
	if len(nodeLabels[utils.NodeWorkerIDLabel]) == 0 || len(nodeLabels[utils.NodeRegionLabel]) == 0 || len(nodeLabels[utils.NodeZoneLabel]) == 0 {
		errorMsg := fmt.Errorf("One or few required node label(s) is/are missing [%s, %s, %s]. Node Labels Found = [#%v]", utils.NodeWorkerIDLabel, utils.NodeRegionLabel, utils.NodeZoneLabel, nodeLabels)
		return nil, errorMsg
	}

	return &nodeMetadataManager{
		zone:     nodeLabels[utils.NodeZoneLabel],
		region:   nodeLabels[utils.NodeRegionLabel],
		workerID: nodeLabels[utils.NodeWorkerIDLabel],
	}, nil
}

func (manager *nodeMetadataManager) GetZone() string {
	return manager.zone
}

func (manager *nodeMetadataManager) GetRegion() string {
	return manager.region
}

func (manager *nodeMetadataManager) GetWorkerID() string {
	return manager.workerID
}

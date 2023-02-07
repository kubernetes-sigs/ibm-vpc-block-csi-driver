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
	"context"
	"fmt"
	"strings"

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

// NodeInfo ...
//go:generate counterfeiter -o fake/fake_node_info.go --fake-name FakeNodeInfo . NodeInfo
type NodeInfo interface {
	NewNodeMetadata(logger *zap.Logger) (NodeMetadata, error)
}

// NodeInfoManager ...
type NodeInfoManager struct {
	NodeName string
}

var _ NodeMetadata = &nodeMetadataManager{}

// NewNodeMetadata ...
func (nodeManager *NodeInfoManager) NewNodeMetadata(logger *zap.Logger) (NodeMetadata, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	node, err := clientset.CoreV1().Nodes().Get(context.Background(), nodeManager.NodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	nodeLabels := node.ObjectMeta.Labels
	if len(nodeLabels[utils.NodeRegionLabel]) == 0 || len(nodeLabels[utils.NodeZoneLabel]) == 0 {
		errorMsg := fmt.Errorf("One or few required node label(s) is/are missing [%s, %s]. Node Labels Found = [#%v]", utils.NodeRegionLabel, utils.NodeZoneLabel, nodeLabels) //nolint:golint
		return nil, errorMsg
	}

	var workerID string

	// If the cluster is satellite, the machine-type label equals to UPI
	if nodeLabels[utils.MachineTypeLabel] == utils.UPI {
		// For a satellite cluster, workerID is fetched from vpc-instance-id node label, which is updated by the vpc-node-label-updater (init container)
		workerID = nodeLabels[utils.NodeInstanceIDLabel]
	} else {
		// For managed and IPI cluster, workerID is fetched from the ProviderID in node spec.
		workerID = fetchInstanceID(node.Spec.ProviderID)
		if workerID == "" {
			return nil, fmt.Errorf("Unable to fetch instance ID from node provider ID - %s", node.Spec.ProviderID)
		}
	}

	return &nodeMetadataManager{
		zone:     nodeLabels[utils.NodeZoneLabel],
		region:   nodeLabels[utils.NodeRegionLabel],
		workerID: workerID,
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

// fetchInstanceID fetches instance ID from the provider ID in node spec.
func fetchInstanceID(providerID string) string {
	s := strings.Split(providerID, "/")
	if len(s) != 7 {
		return ""
	}

	return s[6]
}

/**
 * Copyright 2026 IBM Corp.
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

package vpcvolume

import (
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/client"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// SnapshotConsistencyGroupManager operations
type SnapshotConsistencyGroupManager interface {
	// CreateSnapshotConsistencyGroup creates a snapshot consistency group
	CreateSnapshotConsistencyGroup(snapshotGroupReq *models.SnapshotConsistencyGroupRequest, ctxLogger *zap.Logger) (*models.SnapshotConsistencyGroup, error)

	// Delete a snapshot consistency group
	DeleteSnapshotConsistencyGroup(groupID string, ctxLogger *zap.Logger) error

	// Get a snapshot consistency group by ID
	GetSnapshotConsistencyGroup(groupID string, ctxLogger *zap.Logger) (*models.SnapshotConsistencyGroup, error)

	// Get a snapshot consistency group by name
	GetSnapshotConsistencyGroupByName(name, resourceGroupID string, ctxLogger *zap.Logger) (*models.SnapshotConsistencyGroup, error)

	// List snapshot consistency groups
	ListSnapshotConsistencyGroups(limit int, start string, filters *models.ListSnapshotConsistencyGroupFilters, ctxLogger *zap.Logger) (*models.SnapshotConsistencyGroupList, error)
}

// SnapshotConsistencyGroupService ...
type SnapshotConsistencyGroupService struct {
	client client.SessionClient
}

var _ SnapshotConsistencyGroupManager = &SnapshotConsistencyGroupService{}

// NewSnapshotConsistencyGroupManager ...
func NewSnapshotConsistencyGroupManager(client client.SessionClient) SnapshotConsistencyGroupManager {
	return &SnapshotConsistencyGroupService{
		client: client,
	}
}

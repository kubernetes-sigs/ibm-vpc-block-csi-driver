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

package provider

import (
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/metrics"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// GetGroupSnapshot gets a volume group snapshot by ID
func (vpcs *VPCSession) GetGroupSnapshot(groupSnapshotID string) (*provider.GroupSnapshot, error) {
	vpcs.Logger.Info("Entry GetGroupSnapshot", zap.Reflect("groupSnapshotID", groupSnapshotID))
	defer vpcs.Logger.Info("Exit GetGroupSnapshot", zap.Reflect("groupSnapshotID", groupSnapshotID))
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "GetGroupSnapshot", time.Now())

	var (
		result *models.SnapshotConsistencyGroupResponse
		err    error
	)
	err = retry(vpcs.Logger, func() error {
		result, err = vpcs.Apiclient.SnapshotConsistencyGroupService().GetSnapshotConsistencyGroup(groupSnapshotID, vpcs.Logger)
		return err
	})
	if err != nil {
		return nil, userError.GetUserError("GroupSnapshotRetrievalFailed", err, groupSnapshotID)
	}

	vpcs.Logger.Info("Successfully retrieved volume group snapshot details", zap.Reflect("groupSnapshotDetails", result))

	return fromProviderToLibGroupSnapshot(result, vpcs.Logger)
}

// GetGroupSnapshotByName gets a volume group snapshot by name and resourceGroupID
func (vpcs *VPCSession) GetGroupSnapshotByName(name string, resourceGroupID string) (*provider.GroupSnapshot, error) {
	vpcs.Logger.Info("Entry GetGroupSnapshotByName", zap.String("groupSnapshotName", name), zap.String("resourceGroupID", resourceGroupID))
	defer vpcs.Logger.Info("Exit GetGroupSnapshotByName", zap.String("groupSnapshotName", name), zap.String("resourceGroupID", resourceGroupID))
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "GetGroupSnapshotByName", time.Now())

	if len(name) == 0 {
		return nil, userError.GetUserError("InvalidGroupSnapshotName", nil)
	}

	var (
		result *models.SnapshotConsistencyGroupResponse
		err    error
	)
	err = retry(vpcs.Logger, func() error {
		result, err = vpcs.Apiclient.SnapshotConsistencyGroupService().GetSnapshotConsistencyGroupByName(name, resourceGroupID, vpcs.Logger)
		return err
	})
	if err != nil {
		return nil, userError.GetUserError("GroupSnapshotNameLookupFailed", err, name)
	}

	if result == nil {
		return nil, nil
	}

	vpcs.Logger.Info("Successfully retrieved volume group snapshot details", zap.Reflect("groupSnapshotDetails", result))

	return fromProviderToLibGroupSnapshot(result, vpcs.Logger)
}

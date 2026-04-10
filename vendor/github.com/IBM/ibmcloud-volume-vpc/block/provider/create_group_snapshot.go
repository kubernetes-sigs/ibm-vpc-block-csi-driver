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

// CreateGroupSnapshot creates a snapshot consistency group for the given source volume IDs
func (vpcs *VPCSession) CreateGroupSnapshot(sourceVolumeIDs []string, groupSnapshotParameters provider.GroupSnapshotParameters) (*provider.GroupSnapshot, error) {
	vpcs.Logger.Info("Entry CreateGroupSnapshot", zap.Reflect("groupSnapshotParameters", groupSnapshotParameters), zap.Reflect("sourceVolumeIDs", sourceVolumeIDs))
	defer vpcs.Logger.Info("Exit CreateGroupSnapshot", zap.Reflect("groupSnapshotParameters", groupSnapshotParameters), zap.Reflect("sourceVolumeIDs", sourceVolumeIDs))
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "CreateGroupSnapshot", time.Now())

	// Build the snapshot templates for each source volume
	snapshotTemplates := make([]models.GroupSnapshotTemplate, 0, len(sourceVolumeIDs))
	for _, volID := range sourceVolumeIDs {
		snapshotTemplates = append(snapshotTemplates, models.GroupSnapshotTemplate{
			SourceVolume: &models.SourceVolume{
				ID: volID,
			},
		})
	}

	reqBody := &models.SnapshotConsistencyGroupRequest{
		Name: groupSnapshotParameters.Name,
		ResourceGroup: &models.ResourceGroup{
			ID: groupSnapshotParameters.ResourceGroup,
		},
		Snapshots:               snapshotTemplates,
		DeleteSnapshotsOnDelete: true,
	}

	var result *models.SnapshotConsistencyGroup
	var err error
	err = retry(vpcs.Logger, func() error {
		result, err = vpcs.Apiclient.SnapshotConsistencyGroupService().CreateSnapshotConsistencyGroup(reqBody, vpcs.Logger)
		return err
	})
	if err != nil {
		return nil, userError.GetUserError("SnapshotSpaceOrderFailed", err)
	}

	vpcs.Logger.Info("Successfully created snapshot consistency group", zap.Reflect("GroupSnapshot", result))
	groupSnapshotResponse := FromProviderToLibGroupSnapshot(result, vpcs.Logger)
	vpcs.Logger.Info("GroupSnapshotResponse", zap.Reflect("groupSnapshotResponse", groupSnapshotResponse))
	return groupSnapshotResponse, nil
}

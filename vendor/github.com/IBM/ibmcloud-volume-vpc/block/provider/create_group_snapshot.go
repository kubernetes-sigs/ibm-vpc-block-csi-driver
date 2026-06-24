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

	// List all snapshots belonging to this consistency group to get full details (source_volume, etc.)
	var snapshotList *models.SnapshotList
	err = retry(vpcs.Logger, func() error {
		snapshotList, err = vpcs.Apiclient.SnapshotService().ListSnapshots(0, "", &models.LisSnapshotFilters{
			SnapshotConsistencyGroupID: result.ID,
		}, vpcs.Logger)
		return err
	})
	var snapshotDetails []*models.Snapshot
	if err != nil {
		vpcs.Logger.Warn("Failed to list snapshots for consistency group, falling back to snapshot references", zap.Error(err))
	} else if snapshotList != nil {
		snapshotDetails = snapshotList.Snapshots
	}

	groupSnapshotResponse := FromProviderToLibGroupSnapshot(result, snapshotDetails, vpcs.Logger)
	vpcs.Logger.Info("GroupSnapshotResponse", zap.Reflect("groupSnapshotResponse", groupSnapshotResponse))
	return groupSnapshotResponse, nil
}

// DeleteGroupSnapshot deletes a snapshot consistency group
// snapshotIDs contains the list of individual snapshot IDs within the group snapshot (required by CSI spec)
func (vpcs *VPCSession) DeleteGroupSnapshot(groupSnapshotID string, snapshotIDs []string) error {
	vpcs.Logger.Info("Entry DeleteGroupSnapshot", zap.Reflect("groupSnapshotID", groupSnapshotID), zap.Reflect("snapshotIDs", snapshotIDs))
	defer vpcs.Logger.Info("Exit DeleteGroupSnapshot", zap.Reflect("groupSnapshotID", groupSnapshotID))
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "DeleteGroupSnapshot", time.Now())

	err := retry(vpcs.Logger, func() error {
		return vpcs.Apiclient.SnapshotConsistencyGroupService().DeleteSnapshotConsistencyGroup(groupSnapshotID, vpcs.Logger)
	})
	if err != nil {
		return userError.GetUserError("FailedToDeleteSnapshot", err)
	}

	vpcs.Logger.Info("Successfully deleted the snapshot consistency group")
	return nil
}

// GetGroupSnapshot gets a snapshot consistency group by ID
func (vpcs *VPCSession) GetGroupSnapshot(groupSnapshotID string) (*provider.GroupSnapshot, error) {
	vpcs.Logger.Info("Entry GetGroupSnapshot", zap.Reflect("groupSnapshotID", groupSnapshotID))
	defer vpcs.Logger.Info("Exit GetGroupSnapshot", zap.Reflect("groupSnapshotID", groupSnapshotID))

	var result *models.SnapshotConsistencyGroup
	var err error
	err = retry(vpcs.Logger, func() error {
		result, err = vpcs.Apiclient.SnapshotConsistencyGroupService().GetSnapshotConsistencyGroup(groupSnapshotID, vpcs.Logger)
		return err
	})
	if err != nil {
		return nil, userError.GetUserError("SnapshotIDNotFound", err, groupSnapshotID)
	}

	vpcs.Logger.Info("Successfully retrieved group snapshot details", zap.Reflect("groupSnapshotDetails", result))

	// List all snapshots belonging to this consistency group to get full details (source_volume, etc.)
	var snapshotList *models.SnapshotList
	err = retry(vpcs.Logger, func() error {
		snapshotList, err = vpcs.Apiclient.SnapshotService().ListSnapshots(0, "", &models.LisSnapshotFilters{
			SnapshotConsistencyGroupID: result.ID,
		}, vpcs.Logger)
		return err
	})
	var snapshotDetails []*models.Snapshot
	if err != nil {
		vpcs.Logger.Warn("Failed to list snapshots for consistency group, falling back to snapshot references", zap.Error(err))
	} else if snapshotList != nil {
		snapshotDetails = snapshotList.Snapshots
	}

	groupSnapshotResponse := FromProviderToLibGroupSnapshot(result, snapshotDetails, vpcs.Logger)
	return groupSnapshotResponse, nil
}

// GetGroupSnapshotByName gets a snapshot consistency group by name
func (vpcs *VPCSession) GetGroupSnapshotByName(name string, resourceGroupID string) (*provider.GroupSnapshot, error) {
	vpcs.Logger.Debug("Entry of GetGroupSnapshotByName method...")
	defer vpcs.Logger.Debug("Exit from GetGroupSnapshotByName method...")

	vpcs.Logger.Info("Getting group snapshot details from VPC provider...", zap.Reflect("GroupSnapshotName", name))

	if len(name) == 0 {
		return nil, userError.GetUserError("InvalidSnapshotName", nil, name)
	}

	var result *models.SnapshotConsistencyGroup
	var err error
	err = retry(vpcs.Logger, func() error {
		result, err = vpcs.Apiclient.SnapshotConsistencyGroupService().GetSnapshotConsistencyGroupByName(name, resourceGroupID, vpcs.Logger)
		return err
	})
	if err != nil {
		return nil, userError.GetUserError("StorageFindFailedWithSnapshotName", err, name)
	}

	if result == nil {
		return nil, nil
	}

	vpcs.Logger.Info("Successfully retrieved group snapshot details", zap.Reflect("groupSnapshotDetails", result))

	// List all snapshots belonging to this consistency group to get full details (source_volume, etc.)
	var snapshotList *models.SnapshotList
	err = retry(vpcs.Logger, func() error {
		snapshotList, err = vpcs.Apiclient.SnapshotService().ListSnapshots(0, "", &models.LisSnapshotFilters{
			SnapshotConsistencyGroupID: result.ID,
		}, vpcs.Logger)
		return err
	})
	var snapshotDetails []*models.Snapshot
	if err != nil {
		vpcs.Logger.Warn("Failed to list snapshots for consistency group, falling back to snapshot references", zap.Error(err))
	} else if snapshotList != nil {
		snapshotDetails = snapshotList.Snapshots
	}

	groupSnapshotResponse := FromProviderToLibGroupSnapshot(result, snapshotDetails, vpcs.Logger)
	return groupSnapshotResponse, nil
}

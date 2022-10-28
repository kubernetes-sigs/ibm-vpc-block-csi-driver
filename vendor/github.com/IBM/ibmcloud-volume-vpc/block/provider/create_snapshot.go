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

// Package provider ...
package provider

import (
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/metrics"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	"github.com/IBM/ibmcloud-volume-interface/lib/utils/reasoncode"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

const snapshotReadyState = "stable"

// CreateSnapshot creates snapshot
func (vpcs *VPCSession) CreateSnapshot(sourceVolumeID string, snapshotParameters provider.SnapshotParameters) (*provider.Snapshot, error) {
	vpcs.Logger.Info("Entry CreateSnapshot", zap.Reflect("snapshotRequest", snapshotParameters), zap.Reflect("sourceVolumeID", sourceVolumeID))
	defer vpcs.Logger.Info("Exit CreateSnapshot", zap.Reflect("snapshotRequest", snapshotParameters), zap.Reflect("sourceVolumeID", sourceVolumeID))
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "CreateSnapshot", time.Now())
	var err error
	err = vpcs.validateSnapshotRequest(sourceVolumeID)
	if err != nil {
		return nil, err
	}
	var snapshotResult *models.Snapshot

	// Step 1- validate input which are required

	snapshotTemplate := &models.Snapshot{
		Name:          snapshotParameters.Name,
		SourceVolume:  &models.SourceVolume{ID: sourceVolumeID},
		ResourceGroup: &models.ResourceGroup{ID: vpcs.Config.VPCConfig.ResourceGroupID},
	}

	err = retry(vpcs.Logger, func() error {
		snapshotResult, err = vpcs.Apiclient.SnapshotService().CreateSnapshot(snapshotTemplate, vpcs.Logger)
		return err
	})
	if err != nil {
		return nil, userError.GetUserError("SnapshotSpaceOrderFailed", err)
	}

	vpcs.Logger.Info("Successfully created snapshot with backend (vpcclient) call. Snapshot details", zap.Reflect("Snapshot", snapshotResult))
	// Converting volume to lib snapshot type
	snapshotResponse := FromProviderToLibSnapshot(snapshotResult, vpcs.Logger)
	vpcs.Logger.Info("SnapshotResponse", zap.Reflect("snapshotResponse", snapshotResponse))
	return snapshotResponse, err
}

// validateSnapshotRequest validates request for snapshot
func (vpcs *VPCSession) validateSnapshotRequest(sourceVolumeID string) error {
	var err error
	// Check for VolumeID - required validation
	if len(sourceVolumeID) == 0 {
		err = userError.GetUserError(string(reasoncode.ErrorRequiredFieldMissing), nil, "SourceVolumeID")
		vpcs.Logger.Error("snapshorRequest.SourceVolumeID is required", zap.Error(err))
		return err
	}
	return nil
}

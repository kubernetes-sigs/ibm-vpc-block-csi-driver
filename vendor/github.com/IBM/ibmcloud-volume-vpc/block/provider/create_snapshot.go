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
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// CreateSnapshot Create snapshot from given volume
func (vpcs *VPCSession) CreateSnapshot(volumeRequest *provider.Volume, tags map[string]string) (*provider.Snapshot, error) {
	vpcs.Logger.Info("Entry CreateSnapshot", zap.Reflect("volumeRequest", volumeRequest))
	defer vpcs.Logger.Info("Exit CreateSnapshot", zap.Reflect("volumeRequest", volumeRequest))

	if volumeRequest == nil {
		return nil, userError.GetUserError("StorageFindFailedWithVolumeId", nil, "Not a valid volume ID")
	}

	var snapshot *models.Snapshot
	var err error

	// Step 1- validate input which are required
	vpcs.Logger.Info("Requested volume is:", zap.Reflect("Volume", volumeRequest))

	var volume *models.Volume
	err = retry(vpcs.Logger, func() error {
		volume, err = vpcs.Apiclient.VolumeService().GetVolume(volumeRequest.VolumeID, vpcs.Logger)
		return err
	})
	if err != nil {
		return nil, userError.GetUserError("StorageFindFailedWithVolumeId", err, "Not a valid volume ID")
	}

	if volume == nil {
		return nil, userError.GetUserError("StorageFindFailedWithVolumeId", err, volumeRequest.VolumeID, "Not a valid volume ID")
	}

	err = retry(vpcs.Logger, func() error {
		snapshot, err = vpcs.Apiclient.SnapshotService().CreateSnapshot(volumeRequest.VolumeID, snapshot, vpcs.Logger)
		return err
	})
	if err != nil {
		return nil, userError.GetUserError("SnapshotSpaceOrderFailed", err)
	}

	vpcs.Logger.Info("Successfully created snapshot with backend (vpcclient) call")
	vpcs.Logger.Info("Backend created snapshot details", zap.Reflect("Snapshot", snapshot))

	// Converting volume to lib volume type
	volumeResponse := FromProviderToLibVolume(volume, vpcs.Logger)
	if volumeResponse != nil {
		respSnapshot := &provider.Snapshot{
			Volume:               *volumeResponse,
			SnapshotID:           snapshot.ID,
			SnapshotCreationTime: *snapshot.CreatedAt,
		}
		return respSnapshot, nil
	}

	return nil, userError.GetUserError("CoversionNotSuccessful", err, "Not able to prepare provider volume")
}

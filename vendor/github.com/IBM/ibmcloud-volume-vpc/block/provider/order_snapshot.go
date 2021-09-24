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

// OrderSnapshot order snapshot
func (vpcs *VPCSession) OrderSnapshot(volumeRequest provider.Volume) error {
	vpcs.Logger.Info("Entry OrderSnapshot", zap.Reflect("volumeRequest", volumeRequest))
	defer vpcs.Logger.Info("Exit OrderSnapshot", zap.Reflect("volumeRequest", volumeRequest))

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
		return userError.GetUserError("StorageFindFailedWithVolumeId", err, volumeRequest.VolumeID, "Not a valid volume ID")
	}
	vpcs.Logger.Info("Successfully retrieved given volume details from VPC provider", zap.Reflect("VolumeDetails", volume))

	err = retry(vpcs.Logger, func() error {
		snapshot, err = vpcs.Apiclient.SnapshotService().CreateSnapshot(volumeRequest.VolumeID, snapshot, vpcs.Logger)
		return err
	})
	if err != nil {
		return userError.GetUserError("SnapshotSpaceOrderFailed", err)
	}

	vpcs.Logger.Info("Successfully created the snapshot with backend (vpcclient) call.", zap.Reflect("Snapshot", snapshot))
	return nil
}

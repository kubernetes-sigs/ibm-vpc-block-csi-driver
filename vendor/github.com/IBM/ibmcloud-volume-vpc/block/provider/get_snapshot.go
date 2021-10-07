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

// GetSnapshot get snapshot
func (vpcs *VPCSession) GetSnapshot(snapshotID string) (*provider.Snapshot, error) {
	vpcs.Logger.Info("Entry GetSnapshot", zap.Reflect("SnapshotID", snapshotID))
	defer vpcs.Logger.Info("Exit GetSnapshot", zap.Reflect("SnapshotID", snapshotID))

	return nil, nil
}

// GetSnapshotWithVolumeID get snapshot
func (vpcs *VPCSession) GetSnapshotWithVolumeID(volumeID string, snapshotID string) (*provider.Snapshot, error) {
	vpcs.Logger.Info("Entry GetSnapshot", zap.Reflect("SnapshotID", snapshotID))
	defer vpcs.Logger.Info("Exit GetSnapshot", zap.Reflect("SnapshotID", snapshotID))

	var err error
	var snapshot *models.Snapshot

	err = retry(vpcs.Logger, func() error {
		snapshot, err = vpcs.Apiclient.SnapshotService().GetSnapshot(volumeID, snapshotID, vpcs.Logger)
		return err
	})

	if err != nil {
		return nil, userError.GetUserError("FailedToDeleteSnapshot", err)
	}

	vpcs.Logger.Info("Successfully retrieved the snapshot details", zap.Reflect("Snapshot", snapshot))

	volume, err := vpcs.GetVolume(volumeID)
	if err != nil {
		return nil, userError.GetUserError("StorageFindFailedWithVolumeId", err, volume.VolumeID, "Not a valid volume ID")
	}

	respSnapshot := &provider.Snapshot{
		SnapshotID: snapshot.ID,
		Volume:     *volume,
	}

	vpcs.Logger.Info("Successfully retrieved the snapshot details", zap.Reflect("Provider snapshot", respSnapshot))
	return respSnapshot, nil
}

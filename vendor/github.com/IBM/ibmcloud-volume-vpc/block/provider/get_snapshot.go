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

	vpcs.Logger.Info("Getting snapshot details from VPC provider...", zap.Reflect("SnapshotID", snapshotID))

	var snapshot *models.Snapshot
	var err error
	err = retry(vpcs.Logger, func() error {
		snapshot, err = vpcs.Apiclient.SnapshotService().GetSnapshot(snapshotID, vpcs.Logger)
		return err
	})

	if err != nil {
		return nil, userError.GetUserError("SnapshotIDNotFound", err, snapshotID)
	}

	vpcs.Logger.Info("Successfully retrieved snpashot details from VPC backend", zap.Reflect("snapshotDetails", snapshot))
	snapshotResponse := FromProviderToLibSnapshot(snapshot, vpcs.Logger)
	vpcs.Logger.Info("SnapshotResponse", zap.Reflect("snapshotResponse", snapshotResponse))
	return snapshotResponse, err
}

// GetSnapshotByName ...
func (vpcs *VPCSession) GetSnapshotByName(name string) (respSnap *provider.Snapshot, err error) {
	vpcs.Logger.Debug("Entry of GetSnapshotByName method...")
	defer vpcs.Logger.Debug("Exit from GetSnapshotByName method...")

	vpcs.Logger.Info("Basic validation for snapshot Name...", zap.Reflect("SnapshotName", name))
	if len(name) <= 0 {
		err = userError.GetUserError("InvalidSnapshotName", nil, name)
		return
	}

	vpcs.Logger.Info("Getting snapshot details from VPC provider...", zap.Reflect("SnapshotName", name))

	var snapshot *models.Snapshot
	err = retry(vpcs.Logger, func() error {
		snapshot, err = vpcs.Apiclient.SnapshotService().GetSnapshotByName(name, vpcs.Logger)
		return err
	})

	if err != nil {
		return nil, userError.GetUserError("StorageFindFailedWithSnapshotName", err, snapshot)
	}

	vpcs.Logger.Info("Successfully retrieved snpashot details from VPC backend", zap.Reflect("snapshotDetails", snapshot))
	snapshotResponse := FromProviderToLibSnapshot(snapshot, vpcs.Logger)
	vpcs.Logger.Info("SnapshotResponse", zap.Reflect("snapshotResponse", snapshotResponse))
	return snapshotResponse, err
}

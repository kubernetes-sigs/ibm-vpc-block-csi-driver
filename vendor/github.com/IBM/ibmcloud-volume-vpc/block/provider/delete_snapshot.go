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
	"go.uber.org/zap"
)

// DeleteSnapshot delete snapshot
func (vpcs *VPCSession) DeleteSnapshot(snapshot *provider.Snapshot) error {
	vpcs.Logger.Info("Entry DeleteSnapshot", zap.Reflect("snapshot", snapshot))
	defer vpcs.Logger.Info("Exit DeleteSnapshot", zap.Reflect("snapshot", snapshot))

	var err error
	_, err = vpcs.GetSnapshot(snapshot.SnapshotID)
	if err != nil {
		return userError.GetUserError("StorageFindFailedWithSnapshotId", err, snapshot.SnapshotID, "Not a valid snapshot ID")
	}

	err = retry(vpcs.Logger, func() error {
		err = vpcs.Apiclient.SnapshotService().DeleteSnapshot(snapshot.Volume.VolumeID, snapshot.SnapshotID, vpcs.Logger)
		return err
	})

	if err != nil {
		return userError.GetUserError("FailedToDeleteSnapshot", err)
	}

	vpcs.Logger.Info("Successfully deleted the snapshot with backend (vpcclient) call)")
	return nil
}

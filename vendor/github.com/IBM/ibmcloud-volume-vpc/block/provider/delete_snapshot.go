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
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// DeleteSnapshot delete snapshot
func (vpcs *VPCSession) DeleteSnapshot(snapshot *provider.Snapshot) error {
	vpcs.Logger.Info("Entry DeleteSnapshot", zap.Reflect("snapshotID", snapshot.SnapshotID))
	defer vpcs.Logger.Info("Exit DeleteSnapshot", zap.Reflect("snapshotID", snapshot.SnapshotID))
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "DeleteSnapshot", time.Now())

	var err error
	if snapshot == nil {
		err = userError.GetUserError("InvalidSnapshotID", nil, nil)
		return err
	}

	vpcs.Logger.Info("Deleting snapshot from VPC provider...")
	err = retry(vpcs.Logger, func() error {
		err = vpcs.Apiclient.SnapshotService().DeleteSnapshot(snapshot.SnapshotID, vpcs.Logger)
		return err
	})

	if err != nil {
		modelError, ok := err.(*models.Error)
		if ok && len(modelError.Errors) > 0 && string(modelError.Errors[0].Code) == SnapshotNotFound {
			return userError.GetUserError("SnapshotIDNotFound", err)
		}
		return userError.GetUserError("FailedToDeleteSnapshot", err)
	}

	err = WaitForSnapshotDeletion(vpcs, snapshot.SnapshotID)
	if err != nil {
		return userError.GetUserError("FailedToDeleteSnapshot", err, snapshot.SnapshotID)
	}
	vpcs.Logger.Info("Successfully deleted the snapshot with backend (vpcclient) call)")
	return err
}

// WaitForSnapshotDeletion checks the snapshot for valid status
func WaitForSnapshotDeletion(vpcs *VPCSession, snapshotID string) (err error) {
	vpcs.Logger.Debug("Entry of WaitForSnapshotDeletion method...")
	defer vpcs.Logger.Debug("Exit from WaitForSnapshotDeletion method...")
	var skip = false

	vpcs.Logger.Info("Getting snapshot details from VPC provider...", zap.Reflect("snapshotID", snapshotID))

	err = vpcs.APIRetry.FlexyRetry(vpcs.Logger, func() (error, bool) {
		_, err = vpcs.Apiclient.SnapshotService().GetSnapshot(snapshotID, vpcs.Logger)
		// Keep retry, until GetSnapshot returns snapshots_not_found
		if err != nil {
			skip = skipRetry(err.(*models.Error))
			return nil, skip
		}
		return err, false // continue retry as we are not seeing error which means snapshot is available
	})

	if err == nil && skip {
		vpcs.Logger.Info("Snapshot got deleted.", zap.Reflect("snapshotID", snapshotID))
	}
	return err
}

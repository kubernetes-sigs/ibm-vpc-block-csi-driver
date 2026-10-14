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
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"go.uber.org/zap"
)

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
		return userError.GetUserError("FailedToDeleteGroupSnapshot", err, groupSnapshotID)
	}

	vpcs.Logger.Info("Successfully deleted the snapshot consistency group")
	return nil
}

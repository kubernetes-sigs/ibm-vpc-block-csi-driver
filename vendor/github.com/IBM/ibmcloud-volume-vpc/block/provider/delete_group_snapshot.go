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
	"errors"
	"fmt"
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/metrics"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// DeleteGroupSnapshot deletes the group and waits until VPC confirms it is gone.
func (vpcs *VPCSession) DeleteGroupSnapshot(groupSnapshotID string) error {
	vpcs.Logger.Info("Entry DeleteGroupSnapshot", zap.Reflect("groupSnapshotID", groupSnapshotID))
	defer vpcs.Logger.Info("Exit DeleteGroupSnapshot", zap.Reflect("groupSnapshotID", groupSnapshotID))
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "DeleteGroupSnapshot", time.Now())

	err := retry(vpcs.Logger, func() error {
		return vpcs.Apiclient.SnapshotConsistencyGroupService().DeleteSnapshotConsistencyGroup(groupSnapshotID, vpcs.Logger)
	})
	if err != nil {
		return userError.GetUserError("FailedToDeleteGroupSnapshot", err, groupSnapshotID)
	}

	if err := vpcs.waitForGroupSnapshotDeletion(groupSnapshotID); err != nil {
		return userError.GetUserError("FailedToDeleteGroupSnapshot", err, groupSnapshotID)
	}

	vpcs.Logger.Info("Successfully deleted the volume group snapshot")
	return nil
}

// waitForGroupSnapshotDeletion checks until VPC reports the group was not found.
// Other GET errors and exhausted checks are returned to the caller.
func (vpcs *VPCSession) waitForGroupSnapshotDeletion(groupID string) error {
	// flag that indicates the group snapshot is truly deleted
	deleted := false

	err := vpcs.APIRetry.FlexyRetry(vpcs.Logger, func() (error, bool) {
		_, err := vpcs.Apiclient.SnapshotConsistencyGroupService().GetSnapshotConsistencyGroup(groupID, vpcs.Logger)
		if err == nil {
			return nil, false
		}

		if apiErr, ok := errors.AsType[*models.Error](err); ok && apiErr != nil {
			if len(apiErr.Errors) > 0 && apiErr.Errors[0].Code == SnapshotConsistencyGroupNotFound {
				deleted = true
				return nil, true
			}
		}

		return err, true
	})
	if err != nil {
		return fmt.Errorf("could not confirm deletion of volume group snapshot %q: %w", groupID, err)
	}
	if !deleted {
		return fmt.Errorf("volume group snapshot %q still exists after %d retry attempts", groupID, vpcs.APIRetry.maxRetryAttempt)
	}
	return nil
}

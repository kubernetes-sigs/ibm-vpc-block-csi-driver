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

package vpcvolume

import (
	"time"

	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/client"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// DeleteSnapshotConsistencyGroup DELETEs /snapshot_consistency_groups/{id}
func (scg *SnapshotConsistencyGroupService) DeleteSnapshotConsistencyGroup(groupID string, ctxLogger *zap.Logger) error {
	ctxLogger.Debug("Entry Backend DeleteSnapshotConsistencyGroup")
	defer ctxLogger.Debug("Exit Backend DeleteSnapshotConsistencyGroup")

	defer util.TimeTracker("DeleteSnapshotConsistencyGroup", time.Now())

	operation := &client.Operation{
		Name:        "DeleteSnapshotConsistencyGroup",
		Method:      "DELETE",
		PathPattern: snapshotConsistencyGroupIDPath,
	}

	var apiErr models.Error

	request := scg.client.NewRequest(operation)
	ctxLogger.Info("Equivalent curl command", zap.Reflect("URL", request.URL()), zap.Reflect("Operation", operation))

	_, err := request.PathParameter(snapshotConsistencyGroupIDParam, groupID).JSONError(&apiErr).Invoke()
	if err != nil {
		return err
	}

	return nil
}

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

// CreateSnapshotConsistencyGroup POSTs to /snapshot_consistency_groups
func (scg *SnapshotConsistencyGroupService) CreateSnapshotConsistencyGroup(snapshotGroupReq *models.SnapshotConsistencyGroupRequest, ctxLogger *zap.Logger) (*models.SnapshotConsistencyGroup, error) {
	ctxLogger.Debug("Entry Backend CreateSnapshotConsistencyGroup")
	defer ctxLogger.Debug("Exit Backend CreateSnapshotConsistencyGroup")

	defer util.TimeTracker("CreateSnapshotConsistencyGroup", time.Now())

	operation := &client.Operation{
		Name:        "CreateSnapshotConsistencyGroup",
		Method:      "POST",
		PathPattern: snapshotConsistencyGroupsPath,
	}

	var (
		result models.SnapshotConsistencyGroup
		apiErr models.Error
	)

	request := scg.client.NewRequest(operation)
	ctxLogger.Info("Equivalent curl command and payload details", zap.Reflect("URL", request.URL()), zap.Reflect("Payload", snapshotGroupReq), zap.Reflect("Operation", operation))

	_, err := request.JSONBody(snapshotGroupReq).JSONSuccess(&result).JSONError(&apiErr).Invoke()
	if err != nil {
		return nil, err
	}

	return &result, nil
}

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

// GetSnapshotConsistencyGroup GETs /snapshot_consistency_groups/{id}
func (scg *SnapshotConsistencyGroupService) GetSnapshotConsistencyGroup(groupID string, ctxLogger *zap.Logger) (*models.SnapshotConsistencyGroup, error) {
	ctxLogger.Debug("Entry Backend GetSnapshotConsistencyGroup")
	defer ctxLogger.Debug("Exit Backend GetSnapshotConsistencyGroup")

	defer util.TimeTracker("GetSnapshotConsistencyGroup", time.Now())

	operation := &client.Operation{
		Name:        "GetSnapshotConsistencyGroup",
		Method:      "GET",
		PathPattern: snapshotConsistencyGroupIDPath,
	}

	var result models.SnapshotConsistencyGroup
	var apiErr models.Error

	request := scg.client.NewRequest(operation)
	ctxLogger.Info("Equivalent curl command", zap.Reflect("URL", request.URL()), zap.Reflect("Operation", operation))

	req := request.PathParameter(snapshotConsistencyGroupIDParam, groupID)

	_, err := req.JSONSuccess(&result).JSONError(&apiErr).Invoke()
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetSnapshotConsistencyGroupByName GETs /snapshot_consistency_groups filtered by name
func (scg *SnapshotConsistencyGroupService) GetSnapshotConsistencyGroupByName(name, resourceGroupID string, ctxLogger *zap.Logger) (*models.SnapshotConsistencyGroup, error) {
	ctxLogger.Debug("Entry Backend GetSnapshotConsistencyGroupByName")
	defer ctxLogger.Debug("Exit Backend GetSnapshotConsistencyGroupByName")

	defer util.TimeTracker("GetSnapshotConsistencyGroupByName", time.Now())

	filters := &models.ListSnapshotConsistencyGroupFilters{Name: name, ResourceGroupID: resourceGroupID}
	groups, err := scg.ListSnapshotConsistencyGroups(1, "", filters, ctxLogger)
	if err != nil {
		return nil, err
	}

	if groups != nil {
		groupList := groups.SnapshotConsistencyGroups
		if len(groupList) > 0 {
			return groupList[0], nil
		}
	}
	return nil, err
}

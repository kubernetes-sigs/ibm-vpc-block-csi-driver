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
	"strconv"
	"time"

	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/client"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// ListSnapshotConsistencyGroups GETs /snapshot_consistency_groups
func (scg *SnapshotConsistencyGroupService) ListSnapshotConsistencyGroups(limit int, start string, filters *models.ListSnapshotConsistencyGroupFilters, ctxLogger *zap.Logger) (*models.SnapshotConsistencyGroupList, error) {
	ctxLogger.Debug("Entry Backend ListSnapshotConsistencyGroups")
	defer ctxLogger.Debug("Exit Backend ListSnapshotConsistencyGroups")

	defer util.TimeTracker("ListSnapshotConsistencyGroups", time.Now())

	operation := &client.Operation{
		Name:        "ListSnapshotConsistencyGroups",
		Method:      "GET",
		PathPattern: snapshotConsistencyGroupsPath,
	}

	var groups models.SnapshotConsistencyGroupList
	var apiErr models.Error

	request := scg.client.NewRequest(operation)
	ctxLogger.Info("Equivalent curl command", zap.Reflect("URL", request.URL()), zap.Reflect("Operation", operation))

	req := request.JSONSuccess(&groups).JSONError(&apiErr)

	if limit > 0 {
		req.AddQueryValue("limit", strconv.Itoa(limit))
	}

	if start != "" {
		req.AddQueryValue("start", start)
	}

	if filters != nil {
		if filters.ResourceGroupID != "" {
			req.AddQueryValue("resource_group.id", filters.ResourceGroupID)
		}
		if filters.Name != "" {
			req.AddQueryValue("name", filters.Name)
		}
	}

	_, err := req.Invoke()
	if err != nil {
		return nil, err
	}

	return &groups, nil
}

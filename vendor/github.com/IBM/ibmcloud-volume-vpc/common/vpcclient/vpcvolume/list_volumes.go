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

// Package vpcvolume ...
package vpcvolume

import (
	"strconv"
	"time"

	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/client"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// ListVolumes GETs /volumes
func (vs *VolumeService) ListVolumes(limit int, start string, filters *models.ListVolumeFilters, ctxLogger *zap.Logger) (*models.VolumeList, error) {
	ctxLogger.Debug("Entry Backend ListVolumes")
	defer ctxLogger.Debug("Exit Backend ListVolumes")

	defer util.TimeTracker("ListVolumes", time.Now())

	operation := &client.Operation{
		Name:        "ListVolumes",
		Method:      "GET",
		PathPattern: volumesPath,
	}

	var volumes models.VolumeList
	var apiErr models.Error

	request := vs.client.NewRequest(operation)
	ctxLogger.Info("Equivalent curl command", zap.Reflect("URL", request.URL()), zap.Reflect("Operation", operation))

	req := request.JSONSuccess(&volumes).JSONError(&apiErr)

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
		if filters.Tag != "" {
			req.AddQueryValue("tag", filters.Tag)
		}
		if filters.ZoneName != "" {
			req.AddQueryValue("zone.name", filters.ZoneName)
		}
		if filters.VolumeName != "" {
			req.AddQueryValue("name", filters.VolumeName)
		}
	}

	_, err := req.Invoke()
	if err != nil {
		return nil, err
	}

	return &volumes, nil
}

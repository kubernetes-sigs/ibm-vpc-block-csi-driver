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
	"time"

	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/client"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// GetVolume POSTs to /volumes
func (vs *VolumeService) GetVolume(volumeID string, ctxLogger *zap.Logger) (*models.Volume, error) {
	ctxLogger.Debug("Entry Backend GetVolume")
	defer ctxLogger.Debug("Exit Backend GetVolume")

	defer util.TimeTracker("GetVolume", time.Now())

	operation := &client.Operation{
		Name:        "GetVolume",
		Method:      "GET",
		PathPattern: volumeIDPath,
	}

	var volume models.Volume
	var apiErr models.Error

	request := vs.client.NewRequest(operation)
	ctxLogger.Info("Equivalent curl command", zap.Reflect("URL", request.URL()), zap.Reflect("Operation", operation))

	req := request.PathParameter(volumeIDParam, volumeID)
	_, err := req.JSONSuccess(&volume).JSONError(&apiErr).Invoke()
	if err != nil {
		return nil, err
	}

	return &volume, nil
}

// GetVolumeByName GETs /volumes
func (vs *VolumeService) GetVolumeByName(volumeName string, ctxLogger *zap.Logger) (*models.Volume, error) {
	ctxLogger.Debug("Entry Backend GetVolumeByName")
	defer ctxLogger.Debug("Exit Backend GetVolumeByName")

	defer util.TimeTracker("GetVolumeByName", time.Now())

	// Get the volume details for a single volume, ListVolumeFilters will return only 1 volume in list
	filters := &models.ListVolumeFilters{VolumeName: volumeName}
	volumes, err := vs.ListVolumes(1, "", filters, ctxLogger)
	if err != nil {
		return nil, err
	}

	if volumes != nil {
		volumeslist := volumes.Volumes
		if len(volumeslist) > 0 {
			return volumeslist[0], nil
		}
	}
	return nil, err
}

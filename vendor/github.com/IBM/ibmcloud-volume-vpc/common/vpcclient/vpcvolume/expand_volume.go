/**
 * Copyright 2021 IBM Corp.
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

// ExpandVolume PATCH to /volumes
func (vs *VolumeService) ExpandVolume(volumeID string, volumeTemplate *models.Volume, ctxLogger *zap.Logger) (*models.Volume, error) {
	ctxLogger.Debug("Entry Backend ExpandVolume")
	defer ctxLogger.Debug("Exit Backend ExpandVolume")

	defer util.TimeTracker("ExpandVolume", time.Now())

	operation := &client.Operation{
		Name:        "ExpandVolume",
		Method:      "PATCH",
		PathPattern: volumeIDPath,
	}

	var volume models.Volume
	var apiErr models.Error

	request := vs.client.NewRequest(operation)
	req := request.PathParameter(volumeIDParam, volumeID)
	ctxLogger.Info("Equivalent curl command and payload details", zap.Reflect("URL", req.URL()), zap.Reflect("Payload", volumeTemplate), zap.Reflect("Operation", operation))
	_, err := req.JSONBody(volumeTemplate).JSONSuccess(&volume).JSONError(&apiErr).Invoke()
	if err != nil {
		return nil, err
	}

	return &volume, nil
}

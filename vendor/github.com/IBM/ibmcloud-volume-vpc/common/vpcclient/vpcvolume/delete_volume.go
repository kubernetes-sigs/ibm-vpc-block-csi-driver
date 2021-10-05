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

// DeleteVolume POSTs to /volumes
func (vs *VolumeService) DeleteVolume(volumeID string, ctxLogger *zap.Logger) error {
	ctxLogger.Debug("Entry Backend DeleteVolume")
	defer ctxLogger.Debug("Exit Backend DeleteVolume")

	defer util.TimeTracker("DeleteVolume", time.Now())

	operation := &client.Operation{
		Name:        "DeleteVolume",
		Method:      "DELETE",
		PathPattern: volumeIDPath,
	}

	var apiErr models.Error

	request := vs.client.NewRequest(operation)
	ctxLogger.Info("Equivalent curl command", zap.Reflect("URL", request.URL()), zap.Reflect("Operation", operation))

	_, err := request.PathParameter(volumeIDParam, volumeID).JSONError(&apiErr).Invoke()
	if err != nil {
		return err
	}

	return nil
}

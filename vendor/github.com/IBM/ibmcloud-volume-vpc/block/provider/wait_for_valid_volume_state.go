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

// Package provider ...
package provider

import (
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/metrics"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

const (
	validVolumeStatus = "available"
)

// WaitForValidVolumeState checks the volume for valid status
func WaitForValidVolumeState(vpcs *VPCSession, volumeObj *models.Volume) (err error) {
	vpcs.Logger.Debug("Entry of WaitForValidVolumeState method...")
	defer vpcs.Logger.Debug("Exit from WaitForValidVolumeState method...")
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "WaitForValidVolumeState", time.Now())

	var volumeID string
	var volume *models.Volume

	if volumeObj != nil {
		volumeID = volumeObj.ID
		vpcs.Logger.Info("Getting volume details from VPC provider...", zap.Reflect("VolumeID", volumeID))
	}
	err = retry(vpcs.Logger, func() error {
		volume, err = vpcs.Apiclient.VolumeService().GetVolume(volumeID, vpcs.Logger)
		if err != nil {
			return err
		}
		vpcs.Logger.Info("Getting volume details from VPC provider...", zap.Reflect("volume", volume))
		if volume != nil && volume.Status == validVolumeStatus {
			vpcs.Logger.Info("Volume got valid (available) state", zap.Reflect("VolumeDetails", volume))
			if volume.SourceSnapshot != nil {
				volumeObj.SourceSnapshot = volume.SourceSnapshot
			}
			return nil
		}
		return userError.GetUserError("VolumeNotInValidState", err, volumeID)
	})

	if err != nil {
		vpcs.Logger.Info("Volume could not get valid (available) state", zap.Reflect("VolumeDetails", volume))
		return userError.GetUserError("VolumeNotInValidState", err, volumeID)
	}

	return nil
}

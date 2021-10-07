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
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// DeleteVolume deletes the volume
func (vpcs *VPCSession) DeleteVolume(volume *provider.Volume) (err error) {
	vpcs.Logger.Debug("Entry of DeleteVolume method...")
	defer vpcs.Logger.Debug("Exit from DeleteVolume method...")
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "DeleteVolume", time.Now())

	vpcs.Logger.Info("Validating basic inputs for DeleteVolume method...", zap.Reflect("VolumeDetails", volume))
	err = validateVolume(volume)
	if err != nil {
		return err
	}

	vpcs.Logger.Info("Deleting volume from VPC provider...")
	err = retry(vpcs.Logger, func() error {
		err = vpcs.Apiclient.VolumeService().DeleteVolume(volume.VolumeID, vpcs.Logger)
		return err
	})
	if err != nil {
		return userError.GetUserError("failedToDeleteVolume", err, volume.VolumeID)
	}

	err = WaitForVolumeDeletion(vpcs, volume.VolumeID)
	if err != nil {
		return userError.GetUserError("failedToDeleteVolume", err, volume.VolumeID)
	}

	vpcs.Logger.Info("Successfully deleted volume from VPC provider")
	return err
}

// validateVolume validating volume ID
func validateVolume(volume *provider.Volume) (err error) {
	if volume == nil {
		err = userError.GetUserError("InvalidVolumeID", nil, nil)
		return
	}

	if IsValidVolumeIDFormat(volume.VolumeID) {
		return nil
	}
	err = userError.GetUserError("InvalidVolumeID", nil, volume.VolumeID)
	return
}

// WaitForVolumeDeletion checks the volume for valid status
func WaitForVolumeDeletion(vpcs *VPCSession, volumeID string) (err error) {
	vpcs.Logger.Debug("Entry of WaitForVolumeDeletion method...")
	defer vpcs.Logger.Debug("Exit from WaitForVolumeDeletion method...")
	var skip = false

	vpcs.Logger.Info("Getting volume details from VPC provider...", zap.Reflect("VolumeID", volumeID))

	err = vpcs.APIRetry.FlexyRetry(vpcs.Logger, func() (error, bool) {
		_, err = vpcs.Apiclient.VolumeService().GetVolume(volumeID, vpcs.Logger)
		// Keep retry, until GetVolume returns volume not found
		if err != nil {
			skip = skipRetry(err.(*models.Error))
			return nil, skip
		}
		return err, false // continue retry as we are not seeing error which means volume is available
	})

	if err == nil && skip {
		vpcs.Logger.Info("Volume got deleted.", zap.Reflect("volumeID", volumeID))
	}
	return err
}

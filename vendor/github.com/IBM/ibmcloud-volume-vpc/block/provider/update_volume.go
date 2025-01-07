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
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// UpdateVolume POSTs to /volumes
func (vpcs *VPCSession) UpdateVolume(volumeRequest provider.Volume) error {
	var volumeDetails *models.Volume
	var err error

	err = retry(vpcs.Logger, func() error {
		// Get volume details
		volumeDetails, err = vpcs.Apiclient.VolumeService().GetVolume(volumeRequest.VolumeID, vpcs.Logger)

		if err != nil {
			return err
		}
		vpcs.Logger.Info("Getting volume details from VPC provider...", zap.Reflect("volumeDetails", volumeDetails))
		if volumeDetails != nil && volumeDetails.Status == validVolumeStatus {
			vpcs.Logger.Info("Volume got valid (available) state", zap.Reflect("volumeDetails", volumeDetails))
			return nil
		}
		return userError.GetUserError("VolumeNotInValidState", err, volumeRequest.VolumeID)
	})

	if err != nil {
		return userError.GetUserError("UpdateVolumeWithTagsFailed", err)
	}

	vpcs.Logger.Info("Successfully fetched volume... ", zap.Reflect("volumeDetails", volumeDetails))

	// Converting volume to lib volume type
	existVolume := FromProviderToLibVolume(volumeDetails, vpcs.Logger)

	volumeRequest.VPCVolume.Tags = append(volumeRequest.VPCVolume.Tags, existVolume.Tags...)

	volume := models.Volume{
		ID:       volumeRequest.VolumeID,
		UserTags: volumeRequest.VPCVolume.Tags,
		ETag:     existVolume.ETag,
	}

	vpcs.Logger.Info("Calling VPC provider for volume UpdateVolumeWithTags...")

	err = retry(vpcs.Logger, func() error {
		err = vpcs.Apiclient.VolumeService().UpdateVolume(&volume, vpcs.Logger)
		return err
	})

	if err != nil {
		vpcs.Logger.Debug("Failed to update volume with tags from VPC provider", zap.Reflect("BackendError", err))
		return userError.GetUserError("FailedToUpdateVolume", err, volumeRequest.VolumeID)
	}

	return err
}

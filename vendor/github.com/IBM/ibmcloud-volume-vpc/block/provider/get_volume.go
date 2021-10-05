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

// GetVolume gets the volume by using ID
func (vpcs *VPCSession) GetVolume(id string) (respVolume *provider.Volume, err error) {
	vpcs.Logger.Debug("Entry of GetVolume method...")
	defer vpcs.Logger.Debug("Exit from GetVolume method...")

	vpcs.Logger.Info("Basic validation for volume ID...", zap.Reflect("VolumeID", id))
	// validating volume ID
	err = validateVolumeID(id)
	if err != nil {
		return nil, err
	}

	vpcs.Logger.Info("Getting volume details from VPC provider...", zap.Reflect("VolumeID", id))

	var volume *models.Volume
	err = retry(vpcs.Logger, func() error {
		volume, err = vpcs.Apiclient.VolumeService().GetVolume(id, vpcs.Logger)
		return err
	})

	if err != nil {
		return nil, userError.GetUserError("StorageFindFailedWithVolumeId", err, id)
	}

	vpcs.Logger.Info("Successfully retrieved volume details from VPC backend", zap.Reflect("VolumeDetails", volume))

	// Converting volume to lib volume type
	respVolume = FromProviderToLibVolume(volume, vpcs.Logger)
	return respVolume, err
}

// GetVolumeByName ...
func (vpcs *VPCSession) GetVolumeByName(name string) (respVolume *provider.Volume, err error) {
	vpcs.Logger.Debug("Entry of GetVolumeByName method...")
	defer vpcs.Logger.Debug("Exit from GetVolumeByName method...")

	vpcs.Logger.Info("Basic validation for volume Name...", zap.Reflect("VolumeName", name))
	if len(name) <= 0 {
		err = userError.GetUserError("InvalidVolumeName", nil, name)
		return
	}

	vpcs.Logger.Info("Getting volume details from VPC provider...", zap.Reflect("VolumeName", name))

	var volume *models.Volume
	err = retry(vpcs.Logger, func() error {
		volume, err = vpcs.Apiclient.VolumeService().GetVolumeByName(name, vpcs.Logger)
		return err
	})

	if err != nil {
		return nil, userError.GetUserError("StorageFindFailedWithVolumeName", err, name)
	}

	vpcs.Logger.Info("Successfully retrieved volume details from VPC backend", zap.Reflect("VolumeDetails", volume))

	// Converting volume to lib volume type
	respVolume = FromProviderToLibVolume(volume, vpcs.Logger)
	return respVolume, err
}

// validateVolumeID validating basic volume ID
func validateVolumeID(volumeID string) (err error) {
	if IsValidVolumeIDFormat(volumeID) {
		return nil
	}
	err = userError.GetUserError("InvalidVolumeID", nil, volumeID)
	return
}

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
	"strings"

	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// UpdateVolume PATCH to /volumes
func (vpcs *VPCSession) UpdateVolume(volumeRequest provider.Volume) error {
	var volume *models.Volume
	var existVolume *provider.Volume
	var err error

	//Fetch existing volume Tags	
	err = RetryWithMinRetries(vpcs.Logger, func() error {
		// Get volume details
		existVolume, err = vpcs.GetVolume(volumeRequest.VolumeID)

		if err != nil {
			return err
		}
		if existVolume != nil && existVolume.Status == validVolumeStatus {
			vpcs.Logger.Info("Volume got valid (available) state")
			return nil
		}
		return userError.GetUserError("VolumeNotInValidState", err, volumeRequest.VolumeID)
	})

	if err != nil {
		return err
	}

	//If tags are equal then skip the UpdateVolume RIAAS API call
	if ifTagsEqual(existVolume.Tags, volumeRequest.VPCVolume.Tags) {
		vpcs.Logger.Info("There is no change in user tags for volume, skipping the updateVolume for VPC IaaS... ", zap.Reflect("existVolume", existVolume.Tags), zap.Reflect("volumeRequest", volumeRequest.VPCVolume.Tags))
		return nil
	}

	//Append the existing tags with the requested input tags
	existVolume.Tags = append(existVolume.Tags, volumeRequest.VPCVolume.Tags...)

	volume = &models.Volume{
		ID:       volumeRequest.VolumeID,
		UserTags: existVolume.Tags,
		ETag:     existVolume.ETag,
	}

	vpcs.Logger.Info("Calling VPC provider for volume UpdateVolumeWithTags...")

	err = RetryWithMinRetries(vpcs.Logger, func() error {
		err = vpcs.Apiclient.VolumeService().UpdateVolume(volume, vpcs.Logger)
		return err
	})

	if err != nil {
		vpcs.Logger.Error("Failed to update volume tags from VPC provider", zap.Reflect("BackendError", err))
		return userError.GetUserError("FailedToUpdateVolume", err, volumeRequest.VolumeID)
	}

	return err
}

// ifTagsEqual will check if there is change to existing tags
func ifTagsEqual(existingTags []string, newTags []string) bool {
	//Join slice into a string
	tags := strings.ToLower(strings.Join(existingTags, ","))
	for _, v := range newTags {
		if !strings.Contains(tags, strings.ToLower(v)) {
			//Tags are different
			return false
		}
	}
	//Tags are equal
	return true
}

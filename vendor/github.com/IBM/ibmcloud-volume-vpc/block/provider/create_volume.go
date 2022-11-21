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
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/metrics"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

const (
	customProfile = "custom"
	minSize       = 10
)

// CreateVolume Get the volume by using ID
func (vpcs *VPCSession) CreateVolume(volumeRequest provider.Volume) (volumeResponse *provider.Volume, err error) {
	vpcs.Logger.Debug("Entry of CreateVolume method...")
	defer vpcs.Logger.Debug("Exit from CreateVolume method...")
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "CreateVolume", time.Now())

	vpcs.Logger.Info("Basic validation for CreateVolume request... ", zap.Reflect("RequestedVolumeDetails", volumeRequest))
	resourceGroup, iops, err := validateVolumeRequest(&volumeRequest, vpcs.Config.VPCConfig.ClusterVolumeLabel)
	if err != nil {
		return nil, err
	}
	vpcs.Logger.Info("Successfully validated inputs for CreateVolume request... ")

	// Build the template to send to backend
	volumeTemplate := &models.Volume{
		Name:          *volumeRequest.Name,
		Capacity:      int64(*volumeRequest.Capacity),
		Iops:          iops,
		UserTags:      volumeRequest.VPCVolume.Tags,
		ResourceGroup: &resourceGroup,
		Profile: &models.Profile{
			Name: volumeRequest.VPCVolume.Profile.Name,
		},
		Zone: &models.Zone{
			Name: volumeRequest.Az,
		},
	}
	// adding snapshot ID in the request if it is provided to create the volume from snapshot
	if len(volumeRequest.SnapshotID) > 0 {
		volumeTemplate.SourceSnapshot = &models.Snapshot{ID: volumeRequest.SnapshotID}
	}

	var encryptionKeyCRN string
	if volumeRequest.VPCVolume.VolumeEncryptionKey != nil && len(volumeRequest.VPCVolume.VolumeEncryptionKey.CRN) > 0 {
		encryptionKeyCRN = volumeRequest.VPCVolume.VolumeEncryptionKey.CRN
		volumeTemplate.VolumeEncryptionKey = &models.VolumeEncryptionKey{CRN: encryptionKeyCRN}
	}

	vpcs.Logger.Info("Calling VPC provider for volume creation...")
	var volume *models.Volume
	err = retry(vpcs.Logger, func() error {
		volume, err = vpcs.Apiclient.VolumeService().CreateVolume(volumeTemplate, vpcs.Logger)
		return err
	})

	if err != nil {
		vpcs.Logger.Debug("Failed to create volume from VPC provider", zap.Reflect("BackendError", err))
		modelError, ok := err.(*models.Error)
		if ok && len(modelError.Errors) > 0 && string(modelError.Errors[0].Code) == SnapshotIDNotFound {
			return nil, userError.GetUserError("SnapshotIDNotFound", err)
		}
		return nil, userError.GetUserError("FailedToPlaceOrder", err)
	}

	vpcs.Logger.Info("Successfully created volume from VPC provider...", zap.Reflect("VolumeDetails", volume))

	vpcs.Logger.Info("Waiting for volume to be in valid (available) state", zap.Reflect("VolumeDetails", volume))
	err = WaitForValidVolumeState(vpcs, volume)
	if err != nil {
		return nil, userError.GetUserError("VolumeNotInValidState", err, volume.ID)
	}

	// Converting volume to lib volume type
	volumeResponse = FromProviderToLibVolume(volume, vpcs.Logger)
	// VPC does have region yet . So use requested region in response
	volumeResponse.Region = volumeRequest.Region
	// Return reuested tag as is if not tags returned by backend
	if len(volumeResponse.Tags) == 0 && len(volumeRequest.Tags) > 0 {
		volumeResponse.Tags = volumeRequest.Tags
	}
	vpcs.Logger.Info("VolumeResponse", zap.Reflect("volumeResponse", volumeResponse))
	return volumeResponse, err
}

// validateVolumeRequest validating volume request
func validateVolumeRequest(volumeRequest *provider.Volume, clusterVolumeLabel string) (models.ResourceGroup, int64, error) {
	resourceGroup := models.ResourceGroup{}
	var iops int64
	iops = 0
	// Volume name should not be empty
	if volumeRequest.Name == nil {
		return resourceGroup, iops, userError.GetUserError("InvalidVolumeName", nil, nil)
	} else if len(*volumeRequest.Name) == 0 {
		return resourceGroup, iops, userError.GetUserError("InvalidVolumeName", nil, *volumeRequest.Name)
	}

	// Capacity should not be empty
	if volumeRequest.Capacity == nil {
		return resourceGroup, iops, userError.GetUserError("VolumeCapacityInvalid", nil, nil)
	} else if *volumeRequest.Capacity < minSize {
		return resourceGroup, iops, userError.GetUserError("VolumeCapacityInvalid", nil, *volumeRequest.Capacity)
	}

	// Read user provided error, no harm to pass the 0 values to RIaaS in case of tiered profiles
	if volumeRequest.Iops != nil {
		iops = ToInt64(*volumeRequest.Iops)
	}
	if volumeRequest.VPCVolume.Profile == nil {
		return resourceGroup, iops, userError.GetUserError("VolumeProfileEmpty", nil)
	}
	if volumeRequest.VPCVolume.Profile.Name != customProfile && iops > 0 {
		return resourceGroup, iops, userError.GetUserError("VolumeProfileIopsInvalid", nil)
	}

	// validate and add resource group ID or Name whichever is provided by user
	if volumeRequest.VPCVolume.ResourceGroup == nil {
		return resourceGroup, iops, userError.GetUserError("EmptyResourceGroup", nil)
	}

	// validate and add resource group ID or Name whichever is provided by user
	if len(volumeRequest.VPCVolume.ResourceGroup.ID) == 0 && len(volumeRequest.VPCVolume.ResourceGroup.Name) == 0 {
		return resourceGroup, iops, userError.GetUserError("EmptyResourceGroupIDandName", nil)
	}

	if len(volumeRequest.VPCVolume.ResourceGroup.ID) > 0 {
		resourceGroup.ID = volumeRequest.VPCVolume.ResourceGroup.ID
	}
	if len(volumeRequest.VPCVolume.ResourceGroup.Name) > 0 {
		// get the resource group ID from resource group name as Name is not supported by RIaaS
		resourceGroup.Name = volumeRequest.VPCVolume.ResourceGroup.Name
	}

	//Append the clusterVolumeLabel to existing tag list only if it is non-empty
	if len(clusterVolumeLabel) != 0 {
		tagstr := strings.TrimSpace(clusterVolumeLabel)
		clusterTags := strings.Split(tagstr, ",")
		volumeRequest.VPCVolume.Tags = append(volumeRequest.VPCVolume.Tags, clusterTags...)
	}

	return resourceGroup, iops, nil
}

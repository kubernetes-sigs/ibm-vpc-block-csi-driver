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

// GiB ...
const (
	GiB = 1024 * 1024 * 1024
)

// ExpandVolume Get the volume by using ID
func (vpcs *VPCSession) ExpandVolume(expandVolumeRequest provider.ExpandVolumeRequest) (size int64, err error) {
	vpcs.Logger.Debug("Entry of ExpandVolume method...")
	defer vpcs.Logger.Debug("Exit from ExpandVolume method...")
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "ExpandVolume", time.Now())

	// Get volume details
	existVolume, err := vpcs.GetVolume(expandVolumeRequest.VolumeID)
	if err != nil {
		return -1, err
	}
	// Return existing Capacity if its greater or equal to expandable size
	if existVolume.Capacity != nil && int64(*existVolume.Capacity) >= expandVolumeRequest.Capacity {
		return int64(*existVolume.Capacity), nil
	}
	vpcs.Logger.Info("Successfully validated inputs for ExpandVolume request... ")

	newSize := roundUpSize(expandVolumeRequest.Capacity, GiB)

	// Build the template to send to backend
	volumeTemplate := &models.Volume{
		Capacity: newSize,
	}

	vpcs.Logger.Info("Calling VPC provider for volume expand...")
	var volume *models.Volume
	err = retry(vpcs.Logger, func() error {
		volume, err = vpcs.Apiclient.VolumeService().ExpandVolume(expandVolumeRequest.VolumeID, volumeTemplate, vpcs.Logger)
		return err
	})

	if err != nil {
		vpcs.Logger.Debug("Failed to expand volume from VPC provider", zap.Reflect("BackendError", err))
		return -1, userError.GetUserError("FailedToExpandVolume", err, expandVolumeRequest.VolumeID)
	}

	vpcs.Logger.Info("Successfully accepted volume expansion request, now waiting for volume state equal to available")
	err = WaitForValidVolumeState(vpcs, volume)
	if err != nil {
		return -1, userError.GetUserError("VolumeNotInValidState", err, volume.ID)
	}

	vpcs.Logger.Info("Volume got valid (available) state", zap.Reflect("VolumeDetails", volume))
	return expandVolumeRequest.Capacity, nil
}

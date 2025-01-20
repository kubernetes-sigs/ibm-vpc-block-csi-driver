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
	vpc_provider "github.com/IBM/ibmcloud-volume-vpc/block/provider"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// UpdateVolume updates the volume with given information
func (vpcIks *IksVpcSession) UpdateVolume(volumeRequest provider.Volume) (err error) {
	vpcIks.IksSession.Logger.Debug("Entry of UpdateVolume method...")
	defer vpcIks.IksSession.Logger.Debug("Exit from UpdateVolume method...")
	defer metrics.UpdateDurationFromStart(vpcIks.IksSession.Logger, "UpdateVolume", time.Now())

	vpcIks.IksSession.Logger.Info("Basic validation for UpdateVolume request... ", zap.Reflect("RequestedVolumeDetails", volumeRequest))

	// Build the template to send to backend
	volumeTemplate := models.NewVolume(volumeRequest)
	err = validateVolumeRequest(volumeRequest)
	if err != nil {
		return err
	}

	vpcIks.IksSession.Logger.Info("Successfully validated inputs for UpdateVolume request... ")

	vpcIks.IksSession.Logger.Info("Calling  provider for volume update in ETCD...")
	err = vpcIks.IksSession.APIRetry.FlexyRetry(vpcIks.IksSession.Logger, func() (error, bool) {
		err = vpcIks.IksSession.Apiclient.VolumeService().UpdateVolume(&volumeTemplate, vpcIks.IksSession.Logger)
		return err, err == nil || vpc_provider.SkipRetryForIKS(err)
	})

	if err != nil {
		vpcIks.IksSession.Logger.Debug("Failed to update volume", zap.Reflect("BackendError", err))
		return userError.GetUserError("UpdateFailed", err)
	}

	err = vpc_provider.RetryWithMinRetries(vpcIks.IksSession.Logger, func() error {
		vpcIks.IksSession.Logger.Info("Calling  provider for volume update with tags via RIAAS...")
		err = vpcIks.VPCSession.UpdateVolume(volumeRequest)
		return err
	})

	if err != nil {
		vpcIks.IksSession.Logger.Error("Failed to update volume with tags", zap.Reflect("BackendError", err))
		return userError.GetUserError("UpdateFailed", err)
	}

	return err
}

// validateVolumeRequest validating volume request
func validateVolumeRequest(volumeRequest provider.Volume) error {
	// Volume name should not be empty
	if len(volumeRequest.VolumeID) == 0 {
		return userError.GetUserError("ErrorRequiredFieldMissing", nil, "VolumeID")
	}
	// Provider name should not be empty
	if len(volumeRequest.Provider) == 0 {
		return userError.GetUserError("ErrorRequiredFieldMissing", nil, "Provider")
	}
	// VolumeType  should not be empty
	if len(volumeRequest.VolumeType) == 0 {
		return userError.GetUserError("ErrorRequiredFieldMissing", nil, "VolumeType")
	}

	return nil
}

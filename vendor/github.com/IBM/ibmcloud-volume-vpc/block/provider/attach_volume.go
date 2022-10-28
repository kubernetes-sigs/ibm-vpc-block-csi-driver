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
	"github.com/IBM/ibmcloud-volume-interface/lib/utils/reasoncode"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"

	"go.uber.org/zap"
)

// VpcVolumeAttachment ...
const (
	VpcVolumeAttachment = "vpcVolumeAttachment"
	StatusAttached      = "attached"
	StatusAttaching     = "attaching"
	StatusDetaching     = "detaching"
)

// AttachVolume attach volume based on given volume attachment request
func (vpcs *VPCSession) AttachVolume(volumeAttachmentRequest provider.VolumeAttachmentRequest) (*provider.VolumeAttachmentResponse, error) {
	vpcs.Logger.Debug("Entry of AttachVolume method...")
	defer vpcs.Logger.Debug("Exit from AttachVolume method...")
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "AttachVolume", time.Now())
	var err error

	//check if ServiceSession is valid
	if err = isValidServiceSession(vpcs); err != nil {
		return nil, err
	}

	vpcs.Logger.Info("Validating basic inputs for Attach method...", zap.Reflect("volumeAttachRequest", volumeAttachmentRequest))
	err = vpcs.validateAttachVolumeRequest(volumeAttachmentRequest)
	if err != nil {
		return nil, err
	}
	var volumeAttachResult *models.VolumeAttachment
	var varp *provider.VolumeAttachmentResponse
	// If it is Non IKS environment then remove the IKSVolumeAttachment field from request struct which contains clusterID.
	// TO-DO : Enhance this check. Put it in right place
	if !vpcs.Config.VPCConfig.IsIKS {
		volumeAttachmentRequest.IKSVolumeAttachment = nil
	}
	volumeAttachment := models.NewVolumeAttachment(volumeAttachmentRequest)

	err = vpcs.APIRetry.FlexyRetry(vpcs.Logger, func() (error, bool) {
		// First , check if volume is already attached or attaching to given instance
		vpcs.Logger.Info("Checking if volume is already attached by other thread")
		currentVolAttachment, err := vpcs.GetVolumeAttachment(volumeAttachmentRequest)
		if err == nil && currentVolAttachment != nil && currentVolAttachment.Status != StatusDetaching {
			vpcs.Logger.Info("Volume is already attached", zap.Reflect("currentVolAttachment", currentVolAttachment))
			varp = currentVolAttachment
			return nil, true // stop retry volume already attached
		}
		//Try attaching volume if it's not already attached or there is error in getting current volume attachment
		vpcs.Logger.Info("Attaching volume from VPC provider...", zap.Bool("IKSEnabled?", vpcs.Config.VPCConfig.IsIKS))
		volumeAttachResult, err = vpcs.APIClientVolAttachMgr.AttachVolume(&volumeAttachment, vpcs.Logger)
		// Keep retry, until we get the proper volumeAttachResult object
		if err != nil {
			return err, skipRetryForObviousErrors(err, vpcs.Config.VPCConfig.IsIKS)
		}
		varp = volumeAttachResult.ToVolumeAttachmentResponse(vpcs.Config.VPCConfig.VPCBlockProviderType)
		return err, true // stop retry as no error
	})

	if err != nil {
		userErr := userError.GetUserError(string(userError.VolumeAttachFailed), err, volumeAttachmentRequest.VolumeID, volumeAttachmentRequest.InstanceID)
		return nil, userErr
	}
	vpcs.Logger.Info("Successfully attached volume from VPC provider", zap.Reflect("volumeResponse", varp))
	return varp, nil
}

// validateVolume validating volume ID
func (vpcs *VPCSession) validateAttachVolumeRequest(volumeAttachRequest provider.VolumeAttachmentRequest) error {
	var err error
	// Check for InstanceID - required validation
	if len(volumeAttachRequest.InstanceID) == 0 {
		err = userError.GetUserError(string(reasoncode.ErrorRequiredFieldMissing), nil, "InstanceID")
		vpcs.Logger.Error("volumeAttachRequest.InstanceID is required", zap.Error(err))
		return err
	}
	// Check for VolumeID - required validation
	if len(volumeAttachRequest.VolumeID) == 0 {
		err = userError.GetUserError(string(reasoncode.ErrorRequiredFieldMissing), nil, "VolumeID")
		vpcs.Logger.Error("volumeAttachRequest.VolumeID is required", zap.Error(err))
		return err
	}
	return nil
}

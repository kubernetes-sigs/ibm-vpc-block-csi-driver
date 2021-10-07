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
	"errors"

	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// GetVolumeAttachment  get the volume attachment based on the request
func (vpcs *VPCSession) GetVolumeAttachment(volumeAttachmentRequest provider.VolumeAttachmentRequest) (*provider.VolumeAttachmentResponse, error) {
	vpcs.Logger.Debug("Entry of GetVolumeAttachment method...", zap.Reflect("volumeAttachmentRequest", volumeAttachmentRequest))
	defer vpcs.Logger.Debug("Exit from GetVolumeAttachment method...")
	var err error

	//check if ServiceSession is valid
	if err = isValidServiceSession(vpcs); err != nil {
		return nil, err
	}

	vpcs.Logger.Info("Validating basic inputs for GetVolumeAttachment method...", zap.Reflect("volumeAttachRequest", volumeAttachmentRequest))
	err = vpcs.validateAttachVolumeRequest(volumeAttachmentRequest)
	if err != nil {
		return nil, err
	}
	var volumeAttachmentResponse *provider.VolumeAttachmentResponse
	volumeAttachment := models.NewVolumeAttachment(volumeAttachmentRequest)
	if len(volumeAttachment.ID) > 0 {
		//Get volume attachments by ID if it is specified
		volumeAttachmentResponse, err = vpcs.getVolumeAttachmentByID(volumeAttachment)
	} else {
		// Get volume attachment by Volume ID. This is inefficient operation which requires iteration over volume attachment list
		volumeAttachmentResponse, err = vpcs.getVolumeAttachmentByVolumeID(volumeAttachment)
	}
	vpcs.Logger.Info("Volume attachment response", zap.Reflect("volumeAttachmentResponse", volumeAttachmentResponse), zap.Error(err))
	return volumeAttachmentResponse, err
}

func (vpcs *VPCSession) getVolumeAttachmentByID(volumeAttachmentRequest models.VolumeAttachment) (*provider.VolumeAttachmentResponse, error) {
	vpcs.Logger.Debug("Entry of getVolumeAttachmentByID()")
	defer vpcs.Logger.Debug("Exit from getVolumeAttachmentByID()")
	vpcs.Logger.Info("Getting VolumeAttachment from VPC provider...")
	var err error
	var volumeAttachmentResult *models.VolumeAttachment
	/*err = retry(vpcs.Logger, func() error {
		volumeAttachmentResult, err = vpcs.APIClientVolAttachMgr.GetVolumeAttachment(&volumeAttachmentRequest, vpcs.Logger)
		return err
	})*/

	err = vpcs.APIRetry.FlexyRetry(vpcs.Logger, func() (error, bool) {
		volumeAttachmentResult, err = vpcs.APIClientVolAttachMgr.GetVolumeAttachment(&volumeAttachmentRequest, vpcs.Logger)
		// Keep retry, until we get the proper volumeAttachmentRequest object
		if err != nil {
			return err, skipRetryForObviousErrors(err, vpcs.Config.VPCConfig.IsIKS)
		}
		return err, true // stop retry as no error
	})

	if err != nil {
		// API call is failed
		userErr := userError.GetUserError(string(userError.VolumeAttachFindFailed), err, volumeAttachmentRequest.Volume.ID, *volumeAttachmentRequest.InstanceID)
		return nil, userErr
	}

	volumeAttachmentResponse := volumeAttachmentResult.ToVolumeAttachmentResponse(vpcs.Config.VPCConfig.VPCBlockProviderType)
	vpcs.Logger.Info("Successfully retrieved volume attachment", zap.Reflect("volumeAttachmentResponse", volumeAttachmentResponse))
	return volumeAttachmentResponse, err
}

func (vpcs *VPCSession) getVolumeAttachmentByVolumeID(volumeAttachmentRequest models.VolumeAttachment) (*provider.VolumeAttachmentResponse, error) {
	vpcs.Logger.Debug("Entry of getVolumeAttachmentByVolumeID()")
	defer vpcs.Logger.Debug("Exit from getVolumeAttachmentByVolumeID()")
	vpcs.Logger.Info("Getting VolumeAttachmentList from VPC provider...")
	var volumeAttachmentList *models.VolumeAttachmentList
	var err error
	err = vpcs.APIRetry.FlexyRetry(vpcs.Logger, func() (error, bool) {
		volumeAttachmentList, err = vpcs.APIClientVolAttachMgr.ListVolumeAttachments(&volumeAttachmentRequest, vpcs.Logger)
		// Keep retry, until we get the proper volumeAttachmentRequest object
		if err != nil {
			return err, skipRetryForObviousErrors(err, vpcs.Config.VPCConfig.IsIKS)
		}
		return err, true // stop retry as no error
	})

	if err != nil {
		// API call is failed
		userErr := userError.GetUserError(string(userError.VolumeAttachFindFailed), err, volumeAttachmentRequest.Volume.ID, *volumeAttachmentRequest.InstanceID)
		return nil, userErr
	}
	// Iterate over the volume attachment list for given instance
	for _, volumeAttachmentItem := range volumeAttachmentList.VolumeAttachments {
		// Check if volume ID is matching with requested volume ID
		if volumeAttachmentItem.Volume.ID == volumeAttachmentRequest.Volume.ID {
			vpcs.Logger.Info("Successfully found volume attachment", zap.Reflect("volumeAttachment", volumeAttachmentItem))
			volumeResponse := volumeAttachmentItem.ToVolumeAttachmentResponse(vpcs.Config.VPCConfig.VPCBlockProviderType)
			vpcs.Logger.Info("Successfully fetched volume attachment from VPC provider", zap.Reflect("volumeResponse", volumeResponse))
			return volumeResponse, nil
		}
	}
	// No volume attahment found in the  list. So return error
	userErr := userError.GetUserError(string(userError.VolumeAttachFindFailed), errors.New("no VolumeAttachment Found"), volumeAttachmentRequest.Volume.ID, *volumeAttachmentRequest.InstanceID)
	vpcs.Logger.Error("Volume attachment not found", zap.Error(err))
	return nil, userErr
}

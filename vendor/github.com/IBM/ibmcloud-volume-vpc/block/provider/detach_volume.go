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
	"github.com/IBM/ibmcloud-volume-interface/lib/metrics"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"

	"net/http"
	"time"

	"go.uber.org/zap"
)

// DetachVolume detach volume based on given volume attachment request
func (vpcs *VPCSession) DetachVolume(volumeAttachmentTemplate provider.VolumeAttachmentRequest) (*http.Response, error) {
	vpcs.Logger.Debug("Entry of DetachVolume method...")
	defer vpcs.Logger.Debug("Exit from DetachVolume method...")
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "DetachVolume", time.Now())
	var err error

	//check if ServiceSession is valid
	if err = isValidServiceSession(vpcs); err != nil {
		return nil, err
	}

	vpcs.Logger.Info("Validating basic inputs for detach method...", zap.Reflect("volumeAttachmentTemplate", volumeAttachmentTemplate))
	err = vpcs.validateAttachVolumeRequest(volumeAttachmentTemplate)
	if err != nil {
		return nil, err
	}

	var response *http.Response
	var volumeAttachment models.VolumeAttachment

	err = vpcs.APIRetry.FlexyRetry(vpcs.Logger, func() (error, bool) {
		// First , check if volume is already attached to given instance
		vpcs.Logger.Info("Checking if volume is already attached ")
		currentVolAttachment, err := vpcs.GetVolumeAttachment(volumeAttachmentTemplate)
		if err == nil && currentVolAttachment.Status != StatusDetaching {
			// If no error and current volume is not already in detaching state ( i.e in attached or attaching state) attempt to detach
			vpcs.Logger.Info("Found volume attachment", zap.Reflect("currentVolAttachment", currentVolAttachment))
			volumeAttachment := models.NewVolumeAttachment(volumeAttachmentTemplate)
			volumeAttachment.ID = currentVolAttachment.VPCVolumeAttachment.ID
			vpcs.Logger.Info("Detaching volume from VPC provider...")
			response, err = vpcs.APIClientVolAttachMgr.DetachVolume(&volumeAttachment, vpcs.Logger) //nolint:bodyclose

			if err != nil {
				return err, skipRetryForObviousErrors(err, vpcs.Config.VPCConfig.IsIKS) // Retry in case of all errors
			}
		}
		vpcs.Logger.Info("No volume attachment found for", zap.Reflect("currentVolAttachment", currentVolAttachment), zap.Error(err))
		// consider volume detach success if its  already  in Detaching or VolumeAttachment is not found
		response = &http.Response{
			StatusCode: http.StatusOK,
		}
		return nil, true // skip retry if volume is not found OR alreadd in detaching state
	})
	if err != nil {
		userErr := userError.GetUserError(string(userError.VolumeDetachFailed), err, volumeAttachmentTemplate.VolumeID, volumeAttachmentTemplate.InstanceID, volumeAttachment.ID)
		vpcs.Logger.Error("Volume detach failed with error", zap.Error(err))
		return response, userErr
	}
	vpcs.Logger.Info("Successfully detached volume from VPC provider", zap.Reflect("resp", response))
	return response, nil
}

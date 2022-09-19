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
	"go.uber.org/zap"
)

// WaitForAttachVolume waits for volume to be attached to node. e.g waits till status becomes attached
func (vpcs *VPCSession) WaitForAttachVolume(volumeAttachmentTemplate provider.VolumeAttachmentRequest) (*provider.VolumeAttachmentResponse, error) {
	vpcs.Logger.Debug("Entry of WaitForAttachVolume method...")
	defer vpcs.Logger.Debug("Exit from WaitForAttachVolume method...")
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "WaitForAttachVolume", time.Now())
	var err error

	//check if ServiceSession is valid
	if err = isValidServiceSession(vpcs); err != nil {
		return nil, err
	}

	vpcs.Logger.Info("Validating basic inputs for WaitForAttachVolume method...", zap.Reflect("volumeAttachmentTemplate", volumeAttachmentTemplate))
	err = vpcs.validateAttachVolumeRequest(volumeAttachmentTemplate)
	if err != nil {
		return nil, err
	}

	var currentVolAttachment *provider.VolumeAttachmentResponse
	err = vpcs.APIRetry.FlexyRetryWithCustomGap(vpcs.Logger, func() (error, bool) {
		currentVolAttachment, err = vpcs.GetVolumeAttachment(volumeAttachmentTemplate)
		if err != nil {
			// Need to stop retry as there is an error while getting attachment
			// considering that vpcs.GetVolumeAttachment already re-tried
			return err, true
		}
		// Stop retry in case of volume is attached
		return err, currentVolAttachment != nil && currentVolAttachment.Status == StatusAttached
	})
	// Success case, checks are required in case of timeout happened and volume is still not attached state
	if err == nil && (currentVolAttachment != nil && currentVolAttachment.Status == StatusAttached) {
		return currentVolAttachment, nil
	}

	userErr := userError.GetUserError(string(userError.VolumeAttachTimedOut), nil, volumeAttachmentTemplate.VolumeID, volumeAttachmentTemplate.InstanceID)
	vpcs.Logger.Info("Wait for attach timed out", zap.Error(userErr))

	return nil, userErr
}

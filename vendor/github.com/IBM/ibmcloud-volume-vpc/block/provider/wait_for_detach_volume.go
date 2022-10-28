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
	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"go.uber.org/zap"
)

// WaitForDetachVolume waits for volume to be detached from node. e.g waits till no volume attachment is found
func (vpcs *VPCSession) WaitForDetachVolume(volumeAttachmentTemplate provider.VolumeAttachmentRequest) error {
	vpcs.Logger.Debug("Entry of WaitForDetachVolume method...")
	defer vpcs.Logger.Debug("Exit from WaitForDetachVolume method...")
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "WaitForDetachVolume", time.Now())
	var err error

	//check if ServiceSession is valid
	if err = isValidServiceSession(vpcs); err != nil {
		return err
	}

	vpcs.Logger.Info("Validating basic inputs for WaitForDetachVolume method...", zap.Reflect("volumeAttachmentTemplate", volumeAttachmentTemplate))
	err = vpcs.validateAttachVolumeRequest(volumeAttachmentTemplate)
	if err != nil {
		return err
	}

	err = vpcs.APIRetry.FlexyRetryWithCustomGap(vpcs.Logger, func() (error, bool) {
		_, err := vpcs.GetVolumeAttachment(volumeAttachmentTemplate)
		// In case of error we should not retry as there are two conditions for error
		// 1- some issues at endpoint side --> Which is already covered in vpcs.GetVolumeAttachment
		// 2- Attachment not found i.e err != nil --> in this case we should not re-try as it has been deleted
		if err != nil {
			return err, true
		}
		return err, false
	})

	// Could be a success case
	if err != nil {
		if errMsg, ok := err.(util.Message); ok {
			if errMsg.Code == userError.VolumeAttachFindFailed {
				vpcs.Logger.Info("Volume detachment is complete")
				return nil
			}
		}
	}

	userErr := userError.GetUserError(string(userError.VolumeDetachTimedOut), err, volumeAttachmentTemplate.VolumeID, volumeAttachmentTemplate.InstanceID)
	vpcs.Logger.Info("Wait for detach timed out", zap.Error(userErr))
	return userErr
}

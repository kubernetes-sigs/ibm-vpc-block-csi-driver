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

// Package instances ...
package instances

import (
	"net/http"
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/metrics"
	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/client"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// DetachVolume retrives the volume attach status with givne volume attachment details
func (vs *VolumeAttachService) DetachVolume(volumeAttachmentTemplate *models.VolumeAttachment, ctxLogger *zap.Logger) (*http.Response, error) {
	methodName := "VolumeAttachService.DetachVolume"
	defer util.TimeTracker(methodName, time.Now())
	defer metrics.UpdateDurationFromStart(ctxLogger, methodName, time.Now())

	operation := &client.Operation{
		Name:        "DetachVolume",
		Method:      "DELETE",
		PathPattern: vs.pathPrefix + instanceIDattachmentIDPath,
	}

	apiErr := vs.receiverError

	operationRequest := vs.client.NewRequest(operation)
	operationRequest = vs.populatePathPrefixParameters(operationRequest, volumeAttachmentTemplate)
	operationRequest = operationRequest.PathParameter(attachmentIDParam, volumeAttachmentTemplate.ID)

	ctxLogger.Info("Equivalent curl command details", zap.Reflect("URL", operationRequest.URL()), zap.Reflect("volumeAttachmentTemplate", volumeAttachmentTemplate), zap.Reflect("Operation", operation))
	ctxLogger.Info("Pathparameters", zap.Reflect(instanceIDParam, volumeAttachmentTemplate.InstanceID), zap.Reflect(attachmentIDParam, volumeAttachmentTemplate.ID))

	resp, err := operationRequest.JSONError(apiErr).Invoke()
	if err != nil {
		ctxLogger.Error("Error occurred while deleting volume attachment", zap.Error(err))
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// volume Attachment is deleted. So do not want to retry
			ctxLogger.Info("Exit DetachVolume", zap.Any("resp", resp.StatusCode), zap.Error(err), zap.Error(apiErr))
			return resp, apiErr
		}
	}
	ctxLogger.Info("Exit DetachVolume", zap.Error(err), zap.Error(apiErr))
	return resp, err
}

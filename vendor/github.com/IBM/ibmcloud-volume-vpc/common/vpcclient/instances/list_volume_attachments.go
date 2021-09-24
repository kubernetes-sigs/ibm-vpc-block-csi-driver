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
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/metrics"
	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/client"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// ListVolumeAttachments retrives the list volume attachments with givne volume attachment details
func (vs *VolumeAttachService) ListVolumeAttachments(volumeAttachmentTemplate *models.VolumeAttachment, ctxLogger *zap.Logger) (*models.VolumeAttachmentList, error) {
	methodName := "VolumeAttachService.ListVolumeAttachments"
	defer util.TimeTracker(methodName, time.Now())
	defer metrics.UpdateDurationFromStart(ctxLogger, methodName, time.Now())

	operation := &client.Operation{
		Name:        "ListVolumeAttachment",
		Method:      "GET",
		PathPattern: vs.pathPrefix + instanceIDvolumeAttachmentPath,
	}

	var volumeAttachmentList models.VolumeAttachmentList
	apiErr := vs.receiverError

	operationRequest := vs.client.NewRequest(operation)

	ctxLogger.Info("Equivalent curl command details", zap.Reflect("URL", operationRequest.URL()), zap.Reflect("volumeAttachmentTemplate", volumeAttachmentTemplate), zap.Reflect("Operation", operation))
	operationRequest = vs.populatePathPrefixParameters(operationRequest, volumeAttachmentTemplate)

	_, err := operationRequest.JSONSuccess(&volumeAttachmentList).JSONError(apiErr).Invoke()
	if err != nil {
		ctxLogger.Error("Error occurred while getting volume attachments list", zap.Error(err))
		return nil, err
	}
	ctxLogger.Info("Successfully retrieved the volume attachments")
	return &volumeAttachmentList, nil
}

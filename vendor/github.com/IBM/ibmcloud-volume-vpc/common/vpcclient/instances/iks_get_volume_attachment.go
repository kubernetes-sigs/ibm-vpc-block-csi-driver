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

	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/client"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// GetVolumeAttachment retrives the volume attach status with given volume attachment details
func (vs *IKSVolumeAttachService) GetVolumeAttachment(volumeAttachmentTemplate *models.VolumeAttachment, ctxLogger *zap.Logger) (*models.VolumeAttachment, error) {
	defer util.TimeTracker("IKS GetVolumeAttachment", time.Now())

	operation := &client.Operation{
		Name:        "GetVolumeAttachment",
		Method:      "GET",
		PathPattern: vs.pathPrefix + "getAttachment",
	}

	apiErr := vs.receiverError
	var volumeAttachment models.VolumeAttachment

	operationRequest := vs.client.NewRequest(operation)
	operationRequest = operationRequest.SetQueryValue(IksClusterQueryKey, *volumeAttachmentTemplate.ClusterID)
	operationRequest = operationRequest.SetQueryValue(IksWorkerQueryKey, *volumeAttachmentTemplate.InstanceID)
	operationRequest = operationRequest.SetQueryValue(IksVolumeAttachmentIDQueryKey, volumeAttachmentTemplate.ID)

	ctxLogger.Info("Equivalent curl command and query parameters", zap.Reflect("URL", operationRequest.URL()), zap.Reflect(IksClusterQueryKey, *volumeAttachmentTemplate.ClusterID), zap.Reflect(IksWorkerQueryKey, *volumeAttachmentTemplate.InstanceID), zap.Reflect(IksVolumeAttachmentIDQueryKey, volumeAttachmentTemplate.ID))

	_, err := operationRequest.JSONSuccess(&volumeAttachment).JSONError(apiErr).Invoke()
	if err != nil {
		ctxLogger.Error("Error occurred while getting volume attachment", zap.Error(err))
		return nil, err
	}
	ctxLogger.Info("Successfully retrieved the volume attachment", zap.Reflect("volumeAttachment", volumeAttachment))
	return &volumeAttachment, err
}

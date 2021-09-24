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

// AttachVolume attached volume to instances with givne volume attachment details
func (vs *IKSVolumeAttachService) AttachVolume(volumeAttachmentTemplate *models.VolumeAttachment, ctxLogger *zap.Logger) (*models.VolumeAttachment, error) {
	defer util.TimeTracker("IKS AttachVolume", time.Now())

	operation := &client.Operation{
		Name:        "AttachVolume",
		Method:      "POST",
		PathPattern: vs.pathPrefix + "createAttachment",
	}

	var volumeAttachment models.VolumeAttachment
	apiErr := vs.receiverError

	operationRequest := vs.client.NewRequest(operation)

	operationRequest = operationRequest.SetQueryValue(IksClusterQueryKey, *volumeAttachmentTemplate.ClusterID)
	operationRequest = operationRequest.SetQueryValue(IksWorkerQueryKey, *volumeAttachmentTemplate.InstanceID)
	vol := *volumeAttachmentTemplate.Volume
	operationRequest = operationRequest.SetQueryValue(IksVolumeQueryKey, vol.ID)

	ctxLogger.Info("Equivalent curl command and query parameters", zap.Reflect("URL", operationRequest.URL()), zap.Reflect("Payload", volumeAttachmentTemplate), zap.Reflect("Operation", operation), zap.Reflect(IksClusterQueryKey, volumeAttachmentTemplate.ClusterID), zap.Reflect(IksWorkerQueryKey, volumeAttachmentTemplate.InstanceID), zap.Reflect(IksVolumeQueryKey, vol.ID))

	_, err := operationRequest.JSONBody(volumeAttachmentTemplate).JSONSuccess(&volumeAttachment).JSONError(apiErr).Invoke()
	if err != nil {
		return nil, err
	}

	ctxLogger.Info("Successfully attached the volume")
	return &volumeAttachment, nil
}

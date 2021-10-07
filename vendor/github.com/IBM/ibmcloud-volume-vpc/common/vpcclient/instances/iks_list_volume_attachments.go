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

// ListVolumeAttachments retrives the list volume attachments with givne volume attachment details
func (vs *IKSVolumeAttachService) ListVolumeAttachments(volumeAttachmentTemplate *models.VolumeAttachment, ctxLogger *zap.Logger) (*models.VolumeAttachmentList, error) {
	defer util.TimeTracker("IKS ListVolumeAttachments", time.Now())

	operation := &client.Operation{
		Name:        "ListVolumeAttachment",
		Method:      "GET",
		PathPattern: vs.pathPrefix + "getAttachmentsList",
	}

	var volumeAttachmentList models.VolumeAttachmentList

	apiErr := vs.receiverError
	vs.client = vs.client.WithQueryValue(IksClusterQueryKey, *volumeAttachmentTemplate.ClusterID)
	vs.client = vs.client.WithQueryValue(IksWorkerQueryKey, *volumeAttachmentTemplate.InstanceID)

	operationRequest := vs.client.NewRequest(operation)

	ctxLogger.Info("Equivalent curl command and query parameters", zap.Reflect("URL", operationRequest.URL()), zap.Reflect("volumeAttachmentTemplate", volumeAttachmentTemplate), zap.Reflect("Operation", operation), zap.Reflect(IksClusterQueryKey, *volumeAttachmentTemplate.ClusterID), zap.Reflect(IksWorkerQueryKey, *volumeAttachmentTemplate.InstanceID))

	_, err := operationRequest.JSONSuccess(&volumeAttachmentList).JSONError(apiErr).Invoke()
	if err != nil {
		ctxLogger.Error("Error occurred while getting volume attachments list", zap.Error(err))
		return nil, err
	}
	ctxLogger.Info("Successfully retrieved the volume attachments")
	return &volumeAttachmentList, nil
}

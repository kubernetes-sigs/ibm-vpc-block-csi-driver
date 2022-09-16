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

	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/client"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// VolumeAttachManager operations
//
//go:generate counterfeiter -o fakes/volume_attach_service.go --fake-name VolumeAttachService . VolumeAttachManager
type VolumeAttachManager interface {
	// Create the volume with authorisation by passing required information in the volume object
	AttachVolume(*models.VolumeAttachment, *zap.Logger) (*models.VolumeAttachment, error)
	// GetVolumeAttachment retrives the single VolumeAttachment based on the instance ID and attachmentID
	GetVolumeAttachment(*models.VolumeAttachment, *zap.Logger) (*models.VolumeAttachment, error)
	// ListVolumeAttachments retrives the VolumeAttachment list for given server
	ListVolumeAttachments(*models.VolumeAttachment, *zap.Logger) (*models.VolumeAttachmentList, error)
	// Delete the volume
	DetachVolume(*models.VolumeAttachment, *zap.Logger) (*http.Response, error)
}

// VolumeAttachService ...
type VolumeAttachService struct {
	client                       client.SessionClient
	pathPrefix                   string
	receiverError                error
	populatePathPrefixParameters func(request *client.Request, volumeAttachmentTemplate *models.VolumeAttachment) *client.Request
}

// IKSVolumeAttachService ...
type IKSVolumeAttachService struct {
	client        client.SessionClient
	pathPrefix    string
	receiverError error
}

var _ VolumeAttachManager = &VolumeAttachService{}

// New ...
func New(clientIn client.SessionClient) VolumeAttachManager {
	err := models.Error{}
	return &VolumeAttachService{
		client:        clientIn,
		pathPrefix:    VpcPathPrefix,
		receiverError: &err,
		populatePathPrefixParameters: func(request *client.Request, volumeAttachmentTemplate *models.VolumeAttachment) *client.Request {
			request.PathParameter(instanceIDParam, *volumeAttachmentTemplate.InstanceID)
			return request
		},
	}
}

var _ VolumeAttachManager = &IKSVolumeAttachService{}

// NewIKSVolumeAttachmentManager ...
func NewIKSVolumeAttachmentManager(clientIn client.SessionClient) VolumeAttachManager {
	err := models.IksError{}
	return &IKSVolumeAttachService{
		client:        clientIn,
		pathPrefix:    IksPathPrefix,
		receiverError: &err,
	}
}

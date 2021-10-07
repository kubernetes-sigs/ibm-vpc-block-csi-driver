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
	"net/http"
	"time"
)

const (
	// SUCCESS ...
	SUCCESS = "Success"
	// FAILURE ...
	FAILURE = "Failure"
	// NOTSUPPORTED ...
	NOTSUPPORTED = "Not supported"
)

// VolumeAttachManager ...
type VolumeAttachManager interface {
	//Attach method attaches a volume/ fileset to a server
	//Its non bloking call and does not wait to complete the attachment
	AttachVolume(attachRequest VolumeAttachmentRequest) (*VolumeAttachmentResponse, error)
	//Detach detaches the volume/ fileset from the server
	//Its non bloking call and does not wait to complete the detachment
	DetachVolume(detachRequest VolumeAttachmentRequest) (*http.Response, error)

	//WaitForAttachVolume waits for the volume to be attached to the host
	//Return error if wait is timed out OR there is other error
	WaitForAttachVolume(attachRequest VolumeAttachmentRequest) (*VolumeAttachmentResponse, error)

	//WaitForDetachVolume waits for the volume to be detached from the host
	//Return error if wait is timed out OR there is other error
	WaitForDetachVolume(detachRequest VolumeAttachmentRequest) error

	//GetAttachAttachment retirves the current status of given volume attach request
	GetVolumeAttachment(attachRequest VolumeAttachmentRequest) (*VolumeAttachmentResponse, error)
}

// VolumeAttachmentResponse used for both attach and detach operation
type VolumeAttachmentResponse struct {
	VolumeAttachmentRequest
	//Status status of the volume attachment success, failed, attached, attaching, detaching
	Status    string     `json:"status,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// VolumeAttachmentRequest  used for both attach and detach operation
type VolumeAttachmentRequest struct {
	VolumeID   string `json:"volumeID"`
	InstanceID string `json:"instanceID"`
	// Only for SL provider
	SoftlayerOptions map[string]string `json:"softlayerOptions,omitempty"`
	// Only for VPC provider
	VPCVolumeAttachment *VolumeAttachment `json:"vpcVolumeAttachment"`
	// Only IKS provider
	IKSVolumeAttachment *IKSVolumeAttachment `json:"iksVolumeAttachment"`
}

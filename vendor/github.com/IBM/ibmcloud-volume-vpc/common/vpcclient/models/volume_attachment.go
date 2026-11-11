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

// Package models ...
package models

import (
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
)

// Device ...
type Device struct {
	ID string `json:"id"`
}

// BMSVolumeIP holds an IP address entry returned in the BMS volume attachment IPs list
type BMSVolumeIP struct {
	Address string `json:"address,omitempty"`
}

// VolumeAttachment for riaas client
type VolumeAttachment struct {
	ID   string `json:"id,omitempty"`
	Href string `json:"href,omitempty"`
	Name string `json:"name,omitempty"`
	// Status of volume attachment named - attaching, attached, detaching (VSI) or available (BMS)
	Status string `json:"status,omitempty"`
	Type   string `json:"type,omitempty"` //boot, data
	// InstanceID this volume is attached to (also used as BMS server ID)
	InstanceID *string    `json:"-"`
	ClusterID  *string    `json:"clusterID,omitempty"`
	Device     *Device    `json:"device,omitempty"`
	Volume     *Volume    `json:"volume,omitempty"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
	// If set to true, when deleting the instance the volume will also be deleted
	DeleteVolumeOnInstanceDelete bool `json:"delete_volume_on_instance_delete,omitempty"`

	// BMS SDP-specific fields (only populated for bare metal server attachments)
	// Protocol is the host protocol, e.g. "nvme_tcp"
	Protocol string `json:"interface_protocol,omitempty"`
	// DeleteVolumeOnBMSDelete mirrors the VSI delete-on-delete flag for BMS
	DeleteVolumeOnBMSDelete bool `json:"delete_volume_on_bare_metal_server_delete,omitempty"`
	// NVMeQualifiedName is the NQN for the volume on BMS
	NVMeQualifiedName string `json:"nvme_qualified_name,omitempty"`
	// IPs is the list of NVMe-oF target IPs returned by the BMS attach API
	IPs []BMSVolumeIP `json:"ips,omitempty"`
}

// VolumeAttachmentList ...
type VolumeAttachmentList struct {
	VolumeAttachments []VolumeAttachment `json:"volume_attachments,omitempty"`
}

// NewVolumeAttachment creates VolumeAttachment from VolumeAttachmentRequest
func NewVolumeAttachment(volumeAttachmentRequest provider.VolumeAttachmentRequest) VolumeAttachment {
	va := VolumeAttachment{
		InstanceID: &volumeAttachmentRequest.InstanceID,
		Volume: &Volume{
			ID: volumeAttachmentRequest.VolumeID,
		},
	}
	if volumeAttachmentRequest.VPCVolumeAttachment != nil {
		va.ID = volumeAttachmentRequest.VPCVolumeAttachment.ID
		va.Href = volumeAttachmentRequest.VPCVolumeAttachment.Href
		va.Name = volumeAttachmentRequest.VPCVolumeAttachment.Name
		va.DeleteVolumeOnInstanceDelete = volumeAttachmentRequest.VPCVolumeAttachment.DeleteVolumeOnInstanceDelete
	}
	if volumeAttachmentRequest.IKSVolumeAttachment != nil {
		va.ClusterID = volumeAttachmentRequest.IKSVolumeAttachment.ClusterID
	}
	if volumeAttachmentRequest.BMSVolumeAttachment != nil {
		va.Protocol = volumeAttachmentRequest.BMSVolumeAttachment.Protocol
	}
	return va
}

// ToVolumeAttachmentResponse converts VolumeAttachment to VolumeAttachmentResponse.
// Pass isBMS=true when the attachment was made via the BMS SDP API so that
// the correct terminal status and device-path prefix are applied.
func (va *VolumeAttachment) ToVolumeAttachmentResponse(providerType string, isBMS bool) *provider.VolumeAttachmentResponse {
	varp := &provider.VolumeAttachmentResponse{
		VolumeAttachmentRequest: provider.VolumeAttachmentRequest{
			VolumeID: va.Volume.ID,
			VPCVolumeAttachment: &provider.VolumeAttachment{
				DeleteVolumeOnInstanceDelete: va.DeleteVolumeOnInstanceDelete,
				ID:                           va.ID,
				Href:                         va.Href,
				Name:                         va.Name,
				Type:                         va.Type,
			},
		},
		Status:    va.Status,
		CreatedAt: va.CreatedAt,
	}
	if va.InstanceID != nil {
		varp.InstanceID = *va.InstanceID
	}

	if isBMS {
		// BMS SDP: terminal status is "available"; device path uses NVMe UUID prefix
		// with the full volume UUID (no truncation).
		if va.Status == VolumeAvailable && va.Volume != nil && va.Volume.ID != "" {
			varp.VolumeAttachmentRequest.VPCVolumeAttachment.DevicePath = GTypeBMSDevicePrefix + va.Volume.ID
		}
	} else {
		// VSI path: terminal status is "attached"; device path uses virtio or classic prefix.
		if va.Status == VolumeAttached && va.Device != nil && va.Device.ID != "" {
			if providerType == GTypeG2 {
				if len(va.Device.ID) >= GTypeG2DeviceIDLength {
					varp.VolumeAttachmentRequest.VPCVolumeAttachment.DevicePath = GTypeG2DevicePrefix + va.Device.ID[:GTypeG2DeviceIDLength]
				}
			} else { //GC
				varp.VolumeAttachmentRequest.VPCVolumeAttachment.DevicePath = GTypeClassicDevicePrefix + va.Device.ID
			}
		}
	}
	return varp
}

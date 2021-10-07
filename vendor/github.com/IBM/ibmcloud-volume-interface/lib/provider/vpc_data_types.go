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

import "time"

// VPCVolume specific	parameters
type VPCVolume struct {
	Href                string               `json:"href,omitempty"`
	ResourceGroup       *ResourceGroup       `json:"resource_group,omitempty"`
	VolumeEncryptionKey *VolumeEncryptionKey `json:"encryption_key,omitempty"`
	Profile             *Profile             `json:"profile,omitempty"`
	CRN                 string               `json:"crn,omitempty"`
	VPCBlockVolume
	VPCFileVolume
}

// VPCBlockVolume specific parameters
type VPCBlockVolume struct {
	Tags              []string            `json:"volume_tags,omitempty"`
	VolumeAttachments *[]VolumeAttachment `json:"volume_attachments,omitempty"`
}

// VPCFileVolume specific parameters
type VPCFileVolume struct {
	VolumeAccessPoints *[]VolumeAccessPoint `json:"volume_access_points,omitempty"`
	InitialOwner       *InitialOwner        `json:"initial_owner,omitempty"`
}

// VPC ...
type VPC struct {
	ID   string `json:"id"`
	CRN  string `json:"crn,omitempty"`
	Href string `json:"href,omitempty"`
	Name string `json:"name,omitempty"`
}

// VolumeAccessPoint ...
type VolumeAccessPoint struct {
	ID   string `json:"id,omitempty"`
	Href string `json:"href,omitempty"`
	Name string `json:"name,omitempty"`
	// Status of volume target named - deleted, deleting, failed, pending_deletion, stable, updating, waiting, suspended
	Status    string     `json:"status,omitempty"`
	MountPath *string    `json:"mount_path,omitempty"`
	VPC       *VPC       `json:"vpc,omitempty"`
	Zone      *Zone      `json:"zone,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// Zone ...
type Zone struct {
	Name string `json:"name,omitempty"`
	Href string `json:"href,omitempty"`
}

// InitialOwner ...
type InitialOwner struct {
	GroupID int64 `json:"gid,omitempty"`
	UserID  int64 `json:"uid,omitempty"`
}

// GenerationType ...
type GenerationType string

// String ...
func (i GenerationType) String() string { return string(i) }

// ResourceGroup ...
type ResourceGroup struct {
	Href string `json:"href,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Profile ...
type Profile struct {
	Href string `json:"href,omitempty"`
	Name string `json:"name,omitempty"`
	CRN  string `json:"crn,omitempty"`
}

// VolumeAttachment ...
type VolumeAttachment struct {
	Href string `json:"href,omitempty"`
	// ID volume attachment identifier
	ID string `json:"id,omitempty"`
	// Name volume attachment named
	Name string `json:"name,omitempty"`
	// Type of the volume - boot,data
	Type string `json:"type,omitempty"`
	// If set to true, when deleting the instance the volume will also be deleted
	DeleteVolumeOnInstanceDelete bool `json:"delete_volume_on_instance_delete,omitempty"`
	// device path for attachment
	DevicePath string `json:"device_path,omitempty"`
}

// VolumeEncryptionKey ...
type VolumeEncryptionKey struct {
	CRN string `json:"crn,omitempty"`
}

//IKSVolumeAttachment  encapulates IKS related attachment parameters
type IKSVolumeAttachment struct {
	ClusterID *string `json:"clusterID,omitempty"`
}

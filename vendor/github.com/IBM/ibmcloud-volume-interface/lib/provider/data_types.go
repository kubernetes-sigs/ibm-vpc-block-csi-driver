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
)

// VolumeProvider ...
type VolumeProvider string

// VolumeProviderType ...
type VolumeProviderType string

// VolumeType ...
type VolumeType string

// SnapshotTags ...
type SnapshotTags map[string]string

// Volume ...
type Volume struct {
	// ID of the storage volume, for which we can track the volume
	VolumeID string `json:"volumeID,omitempty"` // order id should be there in the pv object as k10 looks for that in pv object

	// volume provider name
	Provider VolumeProvider `json:"provider"`

	// volume type block or file
	VolumeType VolumeType `json:"volumeType"`

	// Volume provider type i.e  Endurance or Performance or any other name
	ProviderType VolumeProviderType `json:"providerType,omitempty"`

	// The Capacity of the volume, in GiB
	Capacity *int `json:"capacity"`

	// Volume IOPS for performance storage type only
	Iops *string `json:"iops"`

	// for endurance storage type only
	Tier *string `json:"tier"`

	// region of the volume
	Region string `json:"region,omitempty"`

	// Availability zone/datacenter/location of the storage volume
	Az string `json:"az,omitempty"`

	// billing type monthly or hourly
	BillingType string `json:"billingType,omitempty"`

	// Time stamp when volume creation was initiated
	CreationTime time.Time `json:"creationTime"`

	// storage_as_a_service|enterprise|performance     default from SL is storage_as_a_service
	ServiceOffering *string `json:"serviceOffering,omitempty"`

	// Name of a device
	Name *string `json:"name,omitempty"`

	// Backend Ipaddress  OR Hostname of a device. Applicable for file storage only
	BackendIPAddress *string `json:"backendIpAddress,omitempty"`

	// Service address for  mounting NFS volume  Applicable for file storage only
	FileNetworkMountAddress *string `json:"fileNetworkMountAddress,omitempty"`

	// VolumeNotes notes field as a map for all note fileds
	// will keep   {"plugin":"{plugin_name}","region":"{region}","cluster":"{cluster_id}","type":"Endurance","pvc":"{pvc_name}","pv":"{pv_name}","storgeclass":"{storage_class}","reclaim":"Delete/retain"}
	VolumeNotes map[string]string `json:"volumeNotes,omitempty"`

	// LunID the lun of volume, Only for Softlayer block
	LunID string `json:"lunId,omitempty"`

	// Attributes map of specific storage provider volume attributes
	Attributes map[string]string `json:"attributes,omitempty"`

	// IscsiTargetIPAddresses list of target IP addresses for iscsi. Applicable for Iscsi block storage only
	IscsiTargetIPAddresses []string `json:"iscsiTargetIpAddresses,omitempty"`

	// Only for VPC volume provider
	VPCVolume

	// ID of snapshot to be restored
	SnapshotID string `json:"snapshotID,omitempty"`
}

// Snapshot ...
type Snapshot struct {
	VolumeID string `json:"volumeID"`

	// a unique Snapshot ID which created by the provider
	SnapshotID string `json:"snapshotID"`

	// The size of the snapshot, in bytes
	SnapshotSize int64 `json:"snapshotSize"`

	// Time stamp when snapshot creation was initiated
	SnapshotCreationTime time.Time `json:"snapCreationTime"`

	// tags for the snapshot
	SnapshotTags SnapshotTags `json:"tags,omitempty"`

	// status of snapshot
	ReadyToUse bool `json:"readyToUse"`

	// VPC contains vpc fields
	VPC
}

// SnapshotList ...
type SnapshotList struct {
	Next      string      `json:"next"`
	Snapshots []*Snapshot `json:"snapshots"`
}

// VolumeAuthorization capture details of autorization to be made
type VolumeAuthorization struct {
	// Volume to update the authorization
	Volume Volume `json:"volume"`
	// List of subnets to authorize. It might be SubnetIDs or CIDR based on the providers implementaions
	// For example, IBM Softlyaer provider  expects SubnetIDs to be passed
	Subnets []string `json:"subnets,omitempty"`
	// List of HostIPs to authorize
	HostIPs []string `json:"hostIPs,omitempty"`
}

// VolumeList ...
type VolumeList struct {
	Next    string    `json:"next,omitempty"`
	Volumes []*Volume `json:"volumes"`
}

// ExpandVolumeRequest ...
type ExpandVolumeRequest struct {
	// VolumeID id for the volume
	VolumeID string `json:"volumeID"`

	// changed Volume name
	Name *string `json:"name,omitempty"`

	// The new Capacity of the volume, in GiB
	Capacity int64 `json:"capacity"`
}

// SnapshotParameters ...
type SnapshotParameters struct {
	// Name of snapshot
	Name string `json:"name,omitempty"`

	// tags for the snapshot
	SnapshotTags SnapshotTags `json:"tags,omitempty"`
}

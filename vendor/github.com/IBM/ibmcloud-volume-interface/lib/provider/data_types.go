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

	// volume to be restored from snapshot.
	Snapshot
}

// Snapshot ...
type Snapshot struct {
	VolumeID string `json:"volumeID"`

	// a unique Snapshot ID which created by the provider
	SnapshotID string `json:"snapshotID,omitempty"`

	// a unique Snapshot CRN assigned by cloud provider to snapshot
	SnapshotCRN string `json:"snapshotCRN,omitempty"`

	// The group this snapshot belongs to; empty for standalone snapshots.
	GroupSnapshotID string `json:"groupSnapshotID,omitempty"`

	// The size of the snapshot, in bytes
	SnapshotSize int64 `json:"snapshotSize"`

	// Time stamp when snapshot creation was initiated
	SnapshotCreationTime time.Time `json:"snapCreationTime"`

	// tags for the snapshot
	SnapshotTags SnapshotTags `json:"tags,omitempty"`

	// status of snapshot
	ReadyToUse bool `json:"readyToUse"`

	// Common cloud resource details.
	ResourceReference
}

// SnapshotList ...
type SnapshotList struct {
	Next      string      `json:"next"`
	Snapshots []*Snapshot `json:"snapshots"`
}

// GroupSnapshot represents a group of snapshots taken consistently
type GroupSnapshot struct {
	// The unique group snapshotID assigned by the cloud provider
	GroupSnapshotID string `json:"groupSnapshotID"`

	// The group snapshot CRN assigned by cloud provider
	GroupSnapshotCRN string `json:"groupSnapshotCRN"`

	// Time stamp when group snapshot creation was initiated
	GroupSnapshotCreationTime time.Time `json:"groupSnapCreationTime"`

	// List of individual member snapshots in this group
	Snapshots []*Snapshot `json:"snapshots"`

	// status of the group snapshot
	ReadyToUse bool `json:"readyToUse"`

	// Common cloud resource details.
	ResourceReference
}

// GroupSnapshotParameters contains the parameters for creating a group snapshot
type GroupSnapshotParameters struct {
	Name          string `json:"name"`
	ResourceGroup string `json:"resourceGroup"`
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

// VolumePerformance holds the performance attributes of a volume that can be modified.
// It is embedded in both ModifyVolumeRequest and ModifyVolumeResponse.
// The field is named Bandwidth (not Throughput) to match the VPC API and VPCVolume.Bandwidth;
// the CSI "throughput" parameter maps to it.
type VolumePerformance struct {
	// IOPS of the volume
	Iops int64 `json:"iops,omitempty"`

	// Bandwidth (throughput) of the volume, in Mbps
	Bandwidth int32 `json:"bandwidth,omitempty"`
}

// ModifyVolumeRequest ...
type ModifyVolumeRequest struct {
	// VolumeID id for the volume
	VolumeID string `json:"volumeID"`

	// changed Volume name
	Name string `json:"name,omitempty"`

	// The new performance attributes (IOPS, Bandwidth) of the volume
	VolumePerformance
}

// ModifyVolumeResponse holds the updated attributes returned after a ModifyVolume call.
// Using a struct keeps the VolumeManager interface stable: new return fields can be
// added here in the future without changing the method signature.
type ModifyVolumeResponse struct {
	// VolumeID id of the modified volume
	VolumeID string `json:"volumeID"`

	// Href of the modified volume
	Href string `json:"href"`

	// CRN of the modified volume
	CRN string `json:"crn"`

	// The updated performance attributes (IOPS, Bandwidth) of the volume
	VolumePerformance
}

// SnapshotParameters ...
type SnapshotParameters struct {
	// Name of snapshot
	Name string `json:"name,omitempty"`

	// tags for the snapshot
	SnapshotTags SnapshotTags `json:"tags,omitempty"`

	// ResourceGroup is the resourceGroupID
	ResourceGroup string `json:"resourceGroup,omitempty"`
}

// UpdatePVC ...
type UpdatePVC struct {
	ID         string   `json:"id,omitempty"`
	CRN        string   `json:"crn,omitempty"`
	Name       string   `json:"name,omitempty"`
	Capacity   int64    `json:"capacity,omitempty"`
	Iops       int64    `json:"iops,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Cluster    string   `json:"cluster,omitempty"`
	Provider   string   `json:"provider,omitempty"`
	Status     string   `json:"status,omitempty"`
	VolumeType string   `json:"volume_type,omitempty"`
}

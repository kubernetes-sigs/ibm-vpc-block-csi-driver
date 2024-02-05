/**
 * Copyright 2021 IBM Corp.
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

// Package utils ...
package utils

const (
	// KB in bytes
	KB = 1000
	// MB in bytes
	MB = 1000 * KB
	// GB in bytes
	GB = 1000 * MB
	// TB in bytes
	TB = 1000 * GB

	// MinimumVolumeSizeInBytes minimum size of the volume in bytes
	MinimumVolumeSizeInBytes int64 = 10 * GB
	// MinimumVolumeDiskSizeInGb minimum size of the volume in GB
	MinimumVolumeDiskSizeInGb = 10
	// MaximumVolumeDiskSizeInGb ...
	MaximumVolumeDiskSizeInGb = 2000
	// DefaultVolumeDiskSizeinGb default size of the volume in GB
	DefaultVolumeDiskSizeinGb = 10
	// MaxRetryAttemptForSessions ...
	MaxRetryAttemptForSessions = 2
)

const (
	// ClusterInfoPath ...
	ClusterInfoPath = "cluster_info/cluster-config.json"

	// NodeZoneLabel  Zone Label attached to node
	NodeZoneLabel = "failure-domain.beta.kubernetes.io/zone"

	// NodeRegionLabel Region Label attached to node
	NodeRegionLabel = "failure-domain.beta.kubernetes.io/region"

	// NodeInstanceIDLabel VPC ID label attached to satellite host
	NodeInstanceIDLabel = "ibm-cloud.kubernetes.io/vpc-instance-id"

	// MachineTypeLabel is the node label used to identify the cluster type (upi,ipi,etc)
	MachineTypeLabel = "ibm-cloud.kubernetes.io/machine-type"

	// UPI is the expected value assigned to machine-type label on satellite cluster nodes
	UPI = "upi"

	// VolumeIDLabel ...
	VolumeIDLabel = "volumeId"

	// VolumeCRNLabel ...
	VolumeCRNLabel = "volumeCRN"

	// ClusterIDLabel ...
	ClusterIDLabel = "clusterID"

	// IOPSLabel ...
	IOPSLabel = "iops"

	// ZoneLabel ...
	ZoneLabel = "zone"

	// ConfigFileName ...
	ConfigFileName = "slclient.toml"
)

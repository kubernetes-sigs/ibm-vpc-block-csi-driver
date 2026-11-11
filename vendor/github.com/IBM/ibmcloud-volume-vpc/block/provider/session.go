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
	"sync"

	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	vpcconfig "github.com/IBM/ibmcloud-volume-vpc/block/vpcconfig"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/instances"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/riaas"
	"go.uber.org/zap"
)

// bmsOnceState holds the sync.Once state for lazy BMS manager init.
// It is heap-allocated so that VPCSession remains copyable (sync.Once must
// not be copied after first use, but VPCSession is copied in the IKS path).
type bmsOnceState struct {
	once sync.Once
	mgr  instances.VolumeAttachManager
}

// VPCSession implements lib.Session
type VPCSession struct {
	provider.DefaultVolumeProvider
	VPCAccountID          string
	Config                *vpcconfig.VPCBlockConfig
	ContextCredentials    provider.ContextCredentials
	VolumeType            provider.VolumeType
	Provider              provider.VolumeProvider
	Apiclient             riaas.RegionalAPI
	APIClientVolAttachMgr instances.VolumeAttachManager
	APIVersion            string
	Logger                *zap.Logger
	APIRetry              FlexyRetry
	SessionError          error

	// bmsState is heap-allocated so that copying VPCSession (as the IKS path
	// does) does not violate the sync.Once noCopy constraint.
	// Only initialised on the first BMS request; nil on IKS clusters.
	bmsState *bmsOnceState
}

// getBMSVolAttachMgr returns the lazily-initialised BMS volume-attachment
// manager.  It is safe for concurrent use via sync.Once.
func (vpcs *VPCSession) getBMSVolAttachMgr() instances.VolumeAttachManager {
	if vpcs.bmsState == nil {
		vpcs.bmsState = &bmsOnceState{}
	}
	vpcs.bmsState.once.Do(func() {
		vpcs.bmsState.mgr = vpcs.Apiclient.BMSVolumeAttachService()
	})
	return vpcs.bmsState.mgr
}

const (
	// VPC storage provider
	VPC = provider.VolumeProvider("VPC")
	// VolumeType ...
	VolumeType = provider.VolumeType("vpc-block")
	// SnapshotMask ...
	SnapshotMask = "id,username,capacityGb,createDate,snapshotCapacityGb,parentVolume[snapshotSizeBytes],parentVolume[snapshotCapacityGb],parentVolume[id],parentVolume[storageTierLevel],parentVolume[notes],storageType[keyName],serviceResource[datacenter[name]],billingItem[location,hourlyFlag],provisionedIops,lunId,originalVolumeName,storageTierLevel,notes"
)

var (
	// DeleteVolumeReason ...
	DeleteVolumeReason = "deleted by ibm-volume-lib on behalf of user request"
)

// Close at present does nothing
func (*VPCSession) Close() {
	// Do nothing for now
}

// GetProviderDisplayName returns the name of the VPC provider
func (vpcs *VPCSession) GetProviderDisplayName() provider.VolumeProvider {
	return VPC
}

// ProviderName ...
func (vpcs *VPCSession) ProviderName() provider.VolumeProvider {
	return VPC
}

// Type ...
func (vpcs *VPCSession) Type() provider.VolumeType {
	return VolumeType
}

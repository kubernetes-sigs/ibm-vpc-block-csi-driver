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

	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	vpcprovider "github.com/IBM/ibmcloud-volume-vpc/block/provider"
)

// IksVpcSession implements lib.Session for VPC IKS dual session
type IksVpcSession struct {
	vpcprovider.VPCSession                         // Holds VPC/Riaas session by default
	IksSession             *vpcprovider.VPCSession // Holds IKS session
}

var _ provider.Session = &IksVpcSession{}

const (
	// Provider storage provider
	Provider = provider.VolumeProvider("IKS-VPC-Block")
	// VolumeType ...
	VolumeType = provider.VolumeType("VPC-Block")
)

// Close at present does nothing
func (vpcIks *IksVpcSession) Close() {
	// Do nothing for now
}

// GetProviderDisplayName returns the name of the VPC provider
func (vpcIks *IksVpcSession) GetProviderDisplayName() provider.VolumeProvider {
	return Provider
}

// ProviderName ...
func (vpcIks *IksVpcSession) ProviderName() provider.VolumeProvider {
	return Provider
}

// Type ...
func (vpcIks *IksVpcSession) Type() provider.VolumeType {
	return VolumeType
}

// AttachVolume attach volume based on given volume attachment request
func (vpcIks *IksVpcSession) AttachVolume(volumeAttachmentRequest provider.VolumeAttachmentRequest) (*provider.VolumeAttachmentResponse, error) {
	vpcIks.Logger.Debug("Entry of IksVpcSession.AttachVolume method...")
	defer vpcIks.Logger.Debug("Exit from IksVpcSession.AttachVolume method...")
	return vpcIks.IksSession.AttachVolume(volumeAttachmentRequest)
}

// DetachVolume attach volume based on given volume attachment request
func (vpcIks *IksVpcSession) DetachVolume(volumeAttachmentRequest provider.VolumeAttachmentRequest) (*http.Response, error) {
	vpcIks.IksSession.Logger.Debug("Entry of IksVpcSession.DetachVolume method...")
	defer vpcIks.Logger.Debug("Exit from IksVpcSession.DetachVolume method...")
	return vpcIks.IksSession.DetachVolume(volumeAttachmentRequest)
}

// GetVolumeAttachment attach volume based on given volume attachment request
func (vpcIks *IksVpcSession) GetVolumeAttachment(volumeAttachmentRequest provider.VolumeAttachmentRequest) (*provider.VolumeAttachmentResponse, error) {
	vpcIks.Logger.Debug("Entry of IksVpcSession.GetVolumeAttachment method...")
	defer vpcIks.Logger.Debug("Exit from IksVpcSession.GetVolumeAttachment method...")
	return vpcIks.IksSession.GetVolumeAttachment(volumeAttachmentRequest)
}

// WaitForAttachVolume attach volume based on given volume attachment request
func (vpcIks *IksVpcSession) WaitForAttachVolume(volumeAttachmentRequest provider.VolumeAttachmentRequest) (*provider.VolumeAttachmentResponse, error) {
	vpcIks.Logger.Debug("Entry of IksVpcSession.WaitForAttachVolume method...")
	defer vpcIks.Logger.Debug("Exit from IksVpcSession.WaitForAttachVolume method...")
	return vpcIks.IksSession.WaitForAttachVolume(volumeAttachmentRequest)
}

// WaitForDetachVolume attach volume based on given volume attachment request
func (vpcIks *IksVpcSession) WaitForDetachVolume(volumeAttachmentRequest provider.VolumeAttachmentRequest) error {
	vpcIks.Logger.Debug("Entry of IksVpcSession.WaitForDetachVolume method...")
	defer vpcIks.Logger.Debug("Exit from IksVpcSession.WaitForDetachVolume method...")
	return vpcIks.IksSession.WaitForDetachVolume(volumeAttachmentRequest)
}

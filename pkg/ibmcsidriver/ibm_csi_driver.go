/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package ibmcsidriver ...
package ibmcsidriver

import (
	"fmt"

	cloudProvider "github.com/IBM/ibmcloud-volume-vpc/pkg/ibmcloudprovider"
	commonError "github.com/IBM/ibm-csi-common/pkg/messages"
	nodeMetadata "github.com/IBM/ibm-csi-common/pkg/metadata"
	mountManager "github.com/IBM/ibm-csi-common/pkg/mountmanager"
	"github.com/IBM/ibm-csi-common/pkg/utils"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"go.uber.org/zap"
)

// IBMCSIDriver ...
type IBMCSIDriver struct {
	name          string
	vendorVersion string
	logger        *zap.Logger
	region        string
	accountID     string
	ids           *CSIIdentityServer
	ns            *CSINodeServer
	cs            *CSIControllerServer

	vcap  []*csi.VolumeCapability_AccessMode
	cscap []*csi.ControllerServiceCapability
	nscap []*csi.NodeServiceCapability
}

// GetIBMCSIDriver ...
func GetIBMCSIDriver() *IBMCSIDriver {
	return &IBMCSIDriver{}
}

// SetupIBMCSIDriver ...
func (icDriver *IBMCSIDriver) SetupIBMCSIDriver(provider cloudProvider.CloudProviderInterface, mounter mountManager.Mounter, statsUtil StatsUtils, metadata nodeMetadata.NodeMetadata, nodeInfo nodeMetadata.NodeInfo, lgr *zap.Logger, name, vendorVersion string) error {
	icDriver.logger = lgr
	icDriver.logger.Info("IBMCSIDriver-SetupIBMCSIDriver setting up IBM CSI Driver...")

	if provider == nil {
		return fmt.Errorf("provider not initialized")
	}

	if mounter == nil {
		return fmt.Errorf("mounter not initialized")
	}

	if name == "" {
		return fmt.Errorf("driver name missing")
	}

	// Setup messaging
	commonError.MessagesEn = commonError.InitMessages()

	//icDriver.provider = provider
	icDriver.name = name
	icDriver.vendorVersion = vendorVersion

	// Adding Capabilities Todo: Review Access Modes Below
	vcam := []csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
	}

	_ = icDriver.AddVolumeCapabilityAccessModes(vcam) // #nosec G104: Attempt to AddVolumeCapabilityAccessModes only on best-effort basis.Error cannot be usefully handled.
	csc := []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
		// csi.ControllerServiceCapability_RPC_GET_CAPACITY,
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
		csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
		// csi.ControllerServiceCapability_RPC_PUBLISH_READONLY,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
	}
	_ = icDriver.AddControllerServiceCapabilities(csc) // #nosec G104: Attempt to AddControllerServiceCapabilities only on best-effort basis.Error cannot be usefully handled.

	ns := []csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
		csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
		csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
	}
	_ = icDriver.AddNodeServiceCapabilities(ns) // #nosec G104: Attempt to AddNodeServiceCapabilities only on best-effort basis.Error cannot be usefully handled.

	// Set up CSI RPC Servers
	icDriver.ids = NewIdentityServer(icDriver)
	icDriver.ns = NewNodeServer(icDriver, mounter, statsUtil, metadata)
	icDriver.cs = NewControllerServer(icDriver, provider)

	icDriver.logger.Info("Successfully setup IBM CSI driver")

	// Set up Region
	regionMetadata, err := nodeInfo.NewNodeMetadata(lgr)
	if err != nil {
		return fmt.Errorf("Controller_Helper: Failed to initialize node metadata: error: %v", err)
	}
	icDriver.region = regionMetadata.GetRegion()
	icDriver.accountID = regionMetadata.GetAccountID()

	return nil
}

// AddVolumeCapabilityAccessModes ...
func (icDriver *IBMCSIDriver) AddVolumeCapabilityAccessModes(vc []csi.VolumeCapability_AccessMode_Mode) error {
	icDriver.logger.Info("IBMCSIDriver-AddVolumeCapabilityAccessModes...", zap.Reflect("VolumeCapabilityAccessModes", vc))
	var vca []*csi.VolumeCapability_AccessMode
	for _, c := range vc {
		icDriver.logger.Info("Enabling volume access mode", zap.Reflect("Mode", c.String()))
		vca = append(vca, utils.NewVolumeCapabilityAccessMode(c))
	}
	icDriver.vcap = vca
	icDriver.logger.Info("Successfully enabled Volume Capability Access Modes")
	return nil
}

// AddControllerServiceCapabilities ...
func (icDriver *IBMCSIDriver) AddControllerServiceCapabilities(cl []csi.ControllerServiceCapability_RPC_Type) error {
	icDriver.logger.Info("IBMCSIDriver-AddControllerServiceCapabilities...", zap.Reflect("ControllerServiceCapabilities", cl))
	var csc []*csi.ControllerServiceCapability
	for _, c := range cl {
		icDriver.logger.Info("Adding controller service capability", zap.Reflect("Capability", c.String()))
		csc = append(csc, utils.NewControllerServiceCapability(c))
	}
	icDriver.cscap = csc
	icDriver.logger.Info("Successfully added Controller Service Capabilities")
	return nil
}

// AddNodeServiceCapabilities ...
func (icDriver *IBMCSIDriver) AddNodeServiceCapabilities(nl []csi.NodeServiceCapability_RPC_Type) error {
	icDriver.logger.Info("IBMCSIDriver-AddNodeServiceCapabilities...", zap.Reflect("NodeServiceCapabilities", nl))
	var nsc []*csi.NodeServiceCapability
	for _, n := range nl {
		icDriver.logger.Info("Adding node service capability", zap.Reflect("NodeServiceCapabilities", n.String()))
		nsc = append(nsc, utils.NewNodeServiceCapability(n))
	}
	icDriver.nscap = nsc
	icDriver.logger.Info("Successfully added Node Service Capabilities")
	return nil
}

// ValidateControllerServiceRequest ...
/*func (icDriver *IBMCSIDriver) ValidateControllerServiceRequest(c csi.ControllerServiceCapability_RPC_Type) error {
	icDriver.logger.Info("In Driver's ValidateControllerServiceRequest ...", zap.Reflect("ControllerServiceRequest", c))
	if c == csi.ControllerServiceCapability_RPC_UNKNOWN {
		return nil
	}

	for _, cap := range icDriver.cscap {
		if c == cap.GetRpc().Type {
			return nil
		}
	}

	return status.Error(codes.InvalidArgument, "Invalid controller service request")
}*/

// NewIdentityServer ...
func NewIdentityServer(icDriver *IBMCSIDriver) *CSIIdentityServer {
	return &CSIIdentityServer{
		Driver: icDriver,
	}
}

// NewNodeServer ...
func NewNodeServer(icDriver *IBMCSIDriver, mounter mountManager.Mounter, statsUtil StatsUtils, nodeMetadata nodeMetadata.NodeMetadata) *CSINodeServer {
	return &CSINodeServer{
		Driver:   icDriver,
		Mounter:  mounter,
		Stats:    statsUtil,
		Metadata: nodeMetadata,
	}
}

// NewControllerServer ...
func NewControllerServer(icDriver *IBMCSIDriver, provider cloudProvider.CloudProviderInterface) *CSIControllerServer {
	return &CSIControllerServer{
		Driver:      icDriver,
		CSIProvider: provider,
	}
}

// Run ...
func (icDriver *IBMCSIDriver) Run(endpoint string) {
	icDriver.logger.Info("IBMCSIDriver-Run...", zap.Reflect("Endpoint", endpoint))
	icDriver.logger.Info("CSI Driver Name", zap.Reflect("Name", icDriver.name))

	//Start the nonblocking GRPC
	s := NewNonBlockingGRPCServer(icDriver.logger)
	// TODO(#34): Only start specific servers based on a flag.
	// In the future have this only run specific combinations of servers depending on which version this is.
	// The schema for that was in util. basically it was just s.start but with some nil servers.

	s.Start(endpoint, icDriver.ids, icDriver.cs, icDriver.ns)
	s.Wait()
}

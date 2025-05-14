/*
Copyright 2025 The Kubernetes Authors.

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
	commonError "github.com/IBM/ibm-csi-common/pkg/messages"
	"github.com/IBM/ibm-csi-common/pkg/utils"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// CSIIdentityServer ...
type CSIIdentityServer struct {
	Driver *IBMCSIDriver
	csi.UnimplementedIdentityServer
}

// GetPluginInfo ...
func (csiIdentity *CSIIdentityServer) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	ctxLogger.Info("CSIIdentityServer-GetPluginInfo...", zap.Reflect("Request", req))

	if csiIdentity.Driver == nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.DriverNotConfigured, requestID, nil)
	}

	return &csi.GetPluginInfoResponse{
		Name:          csiIdentity.Driver.name,
		VendorVersion: csiIdentity.Driver.vendorVersion,
	}, nil
}

// GetPluginCapabilities ...
func (csiIdentity *CSIIdentityServer) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	ctxLogger, _ := utils.GetContextLogger(ctx, false)
	ctxLogger.Info("CSIIdentityServer-GetPluginCapabilities...", zap.Reflect("Request", req))

	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_VOLUME_ACCESSIBILITY_CONSTRAINTS,
					},
				},
			},
			/* TODO Add Volume Expansion {
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_VolumeExpansion_ONLINE,
					},
				},
			}, */
		},
	}, nil
}

// Probe ...
func (csiIdentity *CSIIdentityServer) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{}, nil
}

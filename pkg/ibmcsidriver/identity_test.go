/*
Copyright 2021 The Kubernetes Authors.

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
	"testing"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestGetPluginInfo(t *testing.T) {
	vendorVersion := "test-vendor-version-1.1.2"
	driver := "mydriver"

	icDriver := initIBMCSIDriver(t)
	if icDriver == nil {
		t.Fatalf("Failed to setup IBM CSI Driver")
	}

	// Get the plugin response by using driver
	resp, err := icDriver.ids.GetPluginInfo(context.Background(), &csi.GetPluginInfoRequest{})
	if err != nil {
		t.Fatalf("GetPluginInfo returned unexpected error: %v", err)
	}

	if resp.GetName() != driver {
		t.Fatalf("Response name expected: %v, got: %v", driver, resp.GetName())
	}

	respVer := resp.GetVendorVersion()
	if respVer != vendorVersion {
		t.Fatalf("Vendor version expected: %v, got: %v", vendorVersion, respVer)
	}

	// set driver as nil
	icDriver.ids.Driver = nil
	resp, err = icDriver.ids.GetPluginInfo(context.Background(), &csi.GetPluginInfoRequest{})
	assert.NotNil(t, err)
	assert.Nil(t, resp)
}

func TestGetPluginCapabilities(t *testing.T) {
	icDriver := initIBMCSIDriver(t)
	if icDriver == nil {
		t.Fatalf("Failed to setup IBM CSI Driver")
	}

	resp, err := icDriver.ids.GetPluginCapabilities(context.Background(), &csi.GetPluginCapabilitiesRequest{})
	if err != nil {
		t.Fatalf("GetPluginCapabilities returned unexpected error: %v", err)
	}

	for _, capability := range resp.GetCapabilities() {
		switch capability.GetService().GetType() {
		case csi.PluginCapability_Service_CONTROLLER_SERVICE:
		case csi.PluginCapability_Service_VOLUME_ACCESSIBILITY_CONSTRAINTS:
		default:
			t.Fatalf("Unknown capability: %v", capability.GetService().GetType())
		}
	}
}

func TestProbe(t *testing.T) {
	icDriver := initIBMCSIDriver(t)
	if icDriver == nil {
		t.Fatalf("Failed to setup IBM CSI Driver")
	}

	_, err := icDriver.ids.Probe(context.Background(), &csi.ProbeRequest{})
	if err != nil {
		t.Fatalf("Probe returned unexpected error: %v", err)
	}
}

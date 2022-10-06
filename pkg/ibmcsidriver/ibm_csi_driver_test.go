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

	cloudProvider "github.com/IBM/ibm-csi-common/pkg/ibmcloudprovider"
	nodeMetadata "github.com/IBM/ibm-csi-common/pkg/metadata"
	mountManager "github.com/IBM/ibm-csi-common/pkg/mountmanager"
	"github.com/stretchr/testify/assert"
)

func initIBMCSIDriver(t *testing.T) *IBMCSIDriver {
	vendorVersion := "test-vendor-version-1.1.2"
	driver := "mydriver"
	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()
	icDriver := GetIBMCSIDriver()
	nodeMeta = nodeMetadata.InitMetadata
	// Create fake provider and mounter
	provider, _ := cloudProvider.NewFakeIBMCloudStorageProvider("", logger)
	mounter := mountManager.NewFakeNodeMounter()
	statsUtil := &MockStatUtils{}

	fakeNodeData := nodeMetadata.FakeNodeMetadata{}
	fakeNodeData.GetRegionReturns("testregion")
	fakeNodeData.GetZoneReturns("testzone")
	fakeNodeData.GetWorkerIDReturns("testworker")

	// Setup the IBM CSI driver
	err := icDriver.SetupIBMCSIDriver(provider, mounter, statsUtil, &fakeNodeData, logger, driver, vendorVersion)
	if err != nil {
		t.Fatalf("Failed to setup IBM CSI Driver: %v", err)
	}

	return icDriver
}

func TestSetupIBMCSIDriver(t *testing.T) {
	// success setting up driver
	driver := initIBMCSIDriver(t)
	assert.NotNil(t, driver)

	// common code
	// Creating test logger
	vendorVersion := "test-vendor-version-1.1.2"
	name := "mydriver"
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()
	icDriver := GetIBMCSIDriver()

	// Create fake provider and mounter
	provider, _ := cloudProvider.NewFakeIBMCloudStorageProvider("", logger)
	mounter := mountManager.NewFakeNodeMounter()
	statsUtil := &MockStatUtils{}

	fakeNodeData := nodeMetadata.FakeNodeMetadata{}
	fakeNodeData.GetRegionReturns("testregion")
	fakeNodeData.GetZoneReturns("testzone")
	fakeNodeData.GetWorkerIDReturns("testworker")

	// Failed setting up driver, provider nil
	err := icDriver.SetupIBMCSIDriver(nil, mounter, statsUtil, &fakeNodeData, logger, name, vendorVersion)
	assert.NotNil(t, err)

	// Failed setting up driver, mounter nil
	err = icDriver.SetupIBMCSIDriver(provider, nil, statsUtil, &fakeNodeData, logger, name, vendorVersion)
	assert.NotNil(t, err)

	// Failed setting up driver, name empty
	err = icDriver.SetupIBMCSIDriver(provider, mounter, statsUtil, &fakeNodeData, logger, "", vendorVersion)
	assert.NotNil(t, err)
}

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
//Package ibmcsidriver ...
package ibmcsidriver

import (
	"testing"

	cloudProvider "github.com/IBM/ibm-csi-common/pkg/ibmcloudprovider"
	"github.com/stretchr/testify/assert"
)

func TestFindDevicePathSource(t *testing.T) {
	testCases := []struct {
		name        string
		req         string
		expResponse string
		expError    error
	}{
		{
			name:        "Valid device path",
			req:         "/tmp",
			expResponse: "/tmp",
			expError:    nil,
		},
		{
			name:        "nvme device path",
			req:         "tmp1234422344",
			expResponse: "tmp1234422344",
			expError:    nil,
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	icDriver := initIBMCSIDriver(t)
	for _, tc := range testCases {
		t.Logf("Test case: %s", tc.name)
		response, err := icDriver.ns.findDevicePathSource(logger, tc.req, "")
		if tc.expError != nil {
			assert.Equal(t, tc.expError, err)
		}
		assert.Equal(t, tc.expResponse, response)
	}
}

func TestProcessMount(t *testing.T) {
	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	icDriver := initIBMCSIDriver(t)
	ops := []string{"a", "b"}
	response, err := icDriver.ns.processMount(logger, "processMount", "/staging", "/targetpath", "ext4", ops)
	t.Logf("Response %v, error %v", response, err)
}

func TestUdevadmTrigger(t *testing.T) {
	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	icDriver := initIBMCSIDriver(t)
	err := icDriver.ns.udevadmTrigger(logger)
	t.Logf("Response error %v", err)
}

func TestProcessMountForBlock(t *testing.T) {
	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	icDriver := initIBMCSIDriver(t)
	ops := []string{"bind"}
	response, err := icDriver.ns.processMountForBlock(logger, "ProcessMountForBlock", "/dev/sda", "/targetpath", "volumeidxxx", ops)
	t.Logf("Response %v, error %v", response, err)
}

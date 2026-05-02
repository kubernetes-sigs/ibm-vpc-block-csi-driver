/*
Copyright 2021-2026 The Kubernetes Authors.

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

	cloudProvider "github.com/IBM/ibmcloud-volume-vpc/pkg/ibmcloudprovider"
	"github.com/stretchr/testify/assert"
	testingexec "k8s.io/utils/exec/testing"
)

func TestFindDevicePathSource(t *testing.T) {
	testCases := []struct {
		name        string
		req         string
		expResponse string
		expectError bool
	}{
		{
			name:        "Device path not found",
			req:         "/dev/nonexistent",
			expResponse: "",
			expectError: true,
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Set environment variable to skip sleep in tests
	t.Setenv("UDEVADM_SLEEP_DURATION", "0s")

	// Mock udevadm command for cross-platform testing
	actionList := []testingexec.FakeCommandAction{
		makeFakeCmd(
			&testingexec.FakeCmd{
				CombinedOutputScript: []testingexec.FakeAction{
					func() ([]byte, []byte, error) {
						return []byte(""), nil, nil
					},
				},
			},
			"udevadm",
		),
	}

	icDriver := initIBMCSIDriver(t, actionList...)
	for _, tc := range testCases {
		t.Logf("Test case: %s", tc.name)
		response, err := icDriver.ns.findDevicePathSource(logger, tc.req, "")
		if tc.expectError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
			assert.Equal(t, tc.expResponse, response)
		}
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

	// Set environment variable to skip sleep in tests
	t.Setenv("UDEVADM_SLEEP_DURATION", "0s")

	// Mock udevadm command for cross-platform testing
	actionList := []testingexec.FakeCommandAction{
		makeFakeCmd(
			&testingexec.FakeCmd{
				CombinedOutputScript: []testingexec.FakeAction{
					func() ([]byte, []byte, error) {
						return []byte(""), nil, nil
					},
				},
			},
			"udevadm",
		),
	}

	icDriver := initIBMCSIDriver(t, actionList...)
	err := icDriver.ns.udevadmTrigger(logger)
	assert.Nil(t, err)
	t.Logf("Response error %v", err)
}

func TestProcessMountForBlock(t *testing.T) {
	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Set environment variable to skip sleep in tests
	t.Setenv("UDEVADM_SLEEP_DURATION", "0s")

	// Mock udevadm command for cross-platform testing
	actionList := []testingexec.FakeCommandAction{
		makeFakeCmd(
			&testingexec.FakeCmd{
				CombinedOutputScript: []testingexec.FakeAction{
					func() ([]byte, []byte, error) {
						return []byte(""), nil, nil
					},
				},
			},
			"udevadm",
		),
	}

	icDriver := initIBMCSIDriver(t, actionList...)
	ops := []string{"bind"}
	response, err := icDriver.ns.processMountForBlock(logger, "ProcessMountForBlock", "/dev/sda", "/targetpath", "volumeidxxx", ops)
	// Expect error since device path doesn't exist in test environment
	assert.NotNil(t, err)
	assert.Nil(t, response)
	t.Logf("Response %v, error %v", response, err)
}

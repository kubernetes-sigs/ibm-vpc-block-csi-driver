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
	"time"

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
			name:        "Empty device path",
			req:         "",
			expResponse: "",
			expectError: true,
		},
		{
			name:        "Device path not found after udevadm",
			req:         "/dev/nonexistent",
			expResponse: "",
			expectError: true,
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Set environment variables for fast retry in tests
	t.Setenv("UDEVADM_MAX_RETRIES", "2")
	t.Setenv("UDEVADM_RETRY_INTERVAL", "10ms")

	// Mock udevadm command for cross-platform testing (trigger + settle)
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
		response, err := icDriver.ns.findDevicePathSource(logger, tc.req, "test-volume-id")
		if tc.expectError {
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), "device path")
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

	testCases := []struct {
		name        string
		expectError bool
		setupMock   func() []testingexec.FakeCommandAction
	}{
		{
			name:        "Successful udevadm trigger and settle",
			expectError: false,
			setupMock: func() []testingexec.FakeCommandAction {
				return []testingexec.FakeCommandAction{
					// First call: udevadm trigger
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
					// Second call: udevadm settle
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
			},
		},
		{
			name:        "Failed udevadm trigger",
			expectError: true,
			setupMock: func() []testingexec.FakeCommandAction {
				return []testingexec.FakeCommandAction{
					makeFakeCmd(
						&testingexec.FakeCmd{
							CombinedOutputScript: []testingexec.FakeAction{
								func() ([]byte, []byte, error) {
									return []byte("udevadm error"), nil, assert.AnError
								},
							},
						},
						"udevadm",
					),
				}
			},
		},
		{
			name:        "Successful trigger but settle fails (non-critical)",
			expectError: false,
			setupMock: func() []testingexec.FakeCommandAction {
				return []testingexec.FakeCommandAction{
					// First call: udevadm trigger (success)
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
					// Second call: udevadm settle (fails but non-critical)
					makeFakeCmd(
						&testingexec.FakeCmd{
							CombinedOutputScript: []testingexec.FakeAction{
								func() ([]byte, []byte, error) {
									return []byte("settle timeout"), nil, assert.AnError
								},
							},
						},
						"udevadm",
					),
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actionList := tc.setupMock()
			icDriver := initIBMCSIDriver(t, actionList...)
			err := icDriver.ns.udevadmTrigger(logger)

			if tc.expectError {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "udevadm trigger failed")
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestWaitForDevicePath(t *testing.T) {
	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	testCases := []struct {
		name          string
		devicePath    string
		maxRetries    int
		retryInterval time.Duration
		expectError   bool
		errorContains string
	}{
		{
			name:          "Device path not found after retries",
			devicePath:    "/dev/nonexistent",
			maxRetries:    3,
			retryInterval: 10 * time.Millisecond,
			expectError:   true,
			errorContains: "did not appear after 3 attempts",
		},
		{
			name:          "Single retry with short interval",
			devicePath:    "/dev/nonexistent",
			maxRetries:    1,
			retryInterval: 10 * time.Millisecond,
			expectError:   true,
			errorContains: "did not appear after 1 attempts",
		},
		{
			name:          "Multiple retries verify timing",
			devicePath:    "/dev/nonexistent",
			maxRetries:    5,
			retryInterval: 10 * time.Millisecond,
			expectError:   true,
			errorContains: "did not appear after 5 attempts",
		},
	}

	icDriver := initIBMCSIDriver(t)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			startTime := time.Now()
			err := icDriver.ns.waitForDevicePath(logger, tc.devicePath, tc.maxRetries, tc.retryInterval)
			elapsed := time.Since(startTime)

			if tc.expectError {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
				// Verify that we actually waited (at least some retries happened)
				minExpectedTime := time.Duration(tc.maxRetries-1) * tc.retryInterval
				assert.GreaterOrEqual(t, elapsed, minExpectedTime, "Should have waited for retries")
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestFindDevicePathSourceWithRetry(t *testing.T) {
	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	testCases := []struct {
		name        string
		devicePath  string
		volumeID    string
		setupEnv    func(*testing.T)
		expectError bool
		setupMock   func() []testingexec.FakeCommandAction
	}{
		{
			name:       "Device not found even after udevadm and retries",
			devicePath: "/dev/nonexistent",
			volumeID:   "test-volume-2",
			setupEnv: func(t *testing.T) {
				t.Setenv("UDEVADM_MAX_RETRIES", "2")
				t.Setenv("UDEVADM_RETRY_INTERVAL", "10ms")
			},
			expectError: true,
			setupMock: func() []testingexec.FakeCommandAction {
				return []testingexec.FakeCommandAction{
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
			},
		},
		{
			name:       "Empty device path",
			devicePath: "",
			volumeID:   "test-volume-3",
			setupEnv: func(t *testing.T) {
				// No env setup needed
			},
			expectError: true,
			setupMock:   func() []testingexec.FakeCommandAction { return nil },
		},
		{
			name:       "Custom retry configuration",
			devicePath: "/dev/nonexistent",
			volumeID:   "test-volume-4",
			setupEnv: func(t *testing.T) {
				t.Setenv("UDEVADM_MAX_RETRIES", "3")
				t.Setenv("UDEVADM_RETRY_INTERVAL", "5ms")
			},
			expectError: true,
			setupMock: func() []testingexec.FakeCommandAction {
				return []testingexec.FakeCommandAction{
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
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupEnv(t)
			actionList := tc.setupMock()

			var icDriver *IBMCSIDriver
			if len(actionList) > 0 {
				icDriver = initIBMCSIDriver(t, actionList...)
			} else {
				icDriver = initIBMCSIDriver(t)
			}

			_, err := icDriver.ns.findDevicePathSource(logger, tc.devicePath, tc.volumeID)

			if tc.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestWaitForDevicePathEnvironmentVariables(t *testing.T) {
	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	testCases := []struct {
		name             string
		maxRetriesEnv    string
		retryIntervalEnv string
		expectedRetries  int
		expectedInterval time.Duration
	}{
		{
			name:             "Custom values from env vars",
			maxRetriesEnv:    "3",
			retryIntervalEnv: "50ms",
			expectedRetries:  3,
			expectedInterval: 50 * time.Millisecond,
		},
		{
			name:             "Invalid max retries falls back to default",
			maxRetriesEnv:    "invalid",
			retryIntervalEnv: "10ms",
			expectedRetries:  15,
			expectedInterval: 10 * time.Millisecond,
		},
		{
			name:             "Zero retries ignored, uses default",
			maxRetriesEnv:    "0",
			retryIntervalEnv: "10ms",
			expectedRetries:  15,
			expectedInterval: 10 * time.Millisecond,
		},
		{
			name:             "Negative retries ignored, uses default",
			maxRetriesEnv:    "-5",
			retryIntervalEnv: "10ms",
			expectedRetries:  15,
			expectedInterval: 10 * time.Millisecond,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			if tc.maxRetriesEnv != "" {
				t.Setenv("UDEVADM_MAX_RETRIES", tc.maxRetriesEnv)
			}
			if tc.retryIntervalEnv != "" {
				t.Setenv("UDEVADM_RETRY_INTERVAL", tc.retryIntervalEnv)
			}

			// Mock udevadm command (trigger + settle)
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

			// Test with a non-existent device to verify retry behavior
			devicePath := "/dev/nonexistent-test-device"
			startTime := time.Now()
			_, err := icDriver.ns.findDevicePathSource(logger, devicePath, "test-volume")
			elapsed := time.Since(startTime)

			// Should fail since device doesn't exist
			assert.NotNil(t, err)

			// Verify timing - should have waited approximately (retries-1) * interval
			// We use retries-1 because the last attempt doesn't sleep
			minExpectedTime := time.Duration(tc.expectedRetries-1) * tc.expectedInterval
			// Allow some tolerance for test execution overhead
			maxExpectedTime := minExpectedTime + (500 * time.Millisecond)

			assert.GreaterOrEqual(t, elapsed, minExpectedTime,
				"Should have waited at least %v but only waited %v", minExpectedTime, elapsed)
			assert.LessOrEqual(t, elapsed, maxExpectedTime,
				"Should not have waited more than %v but waited %v", maxExpectedTime, elapsed)
		})
	}
}

func TestProcessMountForBlock(t *testing.T) {
	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Set environment variables for fast retry in tests
	t.Setenv("UDEVADM_MAX_RETRIES", "2")
	t.Setenv("UDEVADM_RETRY_INTERVAL", "10ms")

	// Mock udevadm command for cross-platform testing (trigger + settle)
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

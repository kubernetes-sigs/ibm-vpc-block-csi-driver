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

	cloudProvider "github.com/IBM/ibmcloud-volume-vpc/pkg/ibmcloudprovider"
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

func TestValidateMkfsOptions(t *testing.T) {
	tests := []struct {
		name      string
		mkfsOpts  string
		fsType    string
		expectErr bool
	}{
		{
			name:      "Valid ext4 options",
			mkfsOpts:  "-E lazy_itable_init=0 -m0",
			fsType:    "ext4",
			expectErr: false,
		},
		{
			name:      "Valid ext4 options with multiple -E flags",
			mkfsOpts:  "-E lazy_itable_init=0,lazy_journal_init=0 -O ^has_journal",
			fsType:    "ext4",
			expectErr: false,
		},
		{
			name:      "Valid xfs options",
			mkfsOpts:  "-f -b size=4096",
			fsType:    "xfs",
			expectErr: false,
		},
		{
			name:      "Valid xfs force option",
			mkfsOpts:  "-f",
			fsType:    "xfs",
			expectErr: false,
		},
		{
			name:      "Command injection attempt with semicolon",
			mkfsOpts:  "-E test; rm -rf /",
			fsType:    "ext4",
			expectErr: true,
		},
		{
			name:      "Command injection attempt with pipe",
			mkfsOpts:  "-E test | cat /etc/passwd",
			fsType:    "ext4",
			expectErr: true,
		},
		{
			name:      "Command injection attempt with ampersand",
			mkfsOpts:  "-E test & echo hacked",
			fsType:    "ext4",
			expectErr: true,
		},
		{
			name:      "Command injection attempt with dollar sign",
			mkfsOpts:  "-E test$HOME",
			fsType:    "ext4",
			expectErr: true,
		},
		{
			name:      "Command injection attempt with backtick",
			mkfsOpts:  "-E test`whoami`",
			fsType:    "ext4",
			expectErr: true,
		},
		{
			name:      "Command injection attempt with command substitution",
			mkfsOpts:  "-E test$(whoami)",
			fsType:    "ext4",
			expectErr: true,
		},
		{
			name:      "Command injection attempt with redirection",
			mkfsOpts:  "-E test > /tmp/file",
			fsType:    "ext4",
			expectErr: true,
		},
		{
			name:      "Command injection attempt with quotes",
			mkfsOpts:  "-E 'test'",
			fsType:    "ext4",
			expectErr: true,
		},
		{
			name:      "Command injection attempt with double quotes",
			mkfsOpts:  "-E \"test\"",
			fsType:    "ext4",
			expectErr: true,
		},
		{
			name:      "Command injection attempt with backslash",
			mkfsOpts:  "-E test\\nrm",
			fsType:    "ext4",
			expectErr: true,
		},
		{
			name:      "Invalid option for filesystem type",
			mkfsOpts:  "-Z invalid",
			fsType:    "ext4",
			expectErr: true,
		},
		{
			name:      "Invalid option for xfs",
			mkfsOpts:  "-Z invalid",
			fsType:    "xfs",
			expectErr: true,
		},
		{
			name:      "Empty options",
			mkfsOpts:  "",
			fsType:    "ext4",
			expectErr: false,
		},
		{
			name:      "Whitespace only",
			mkfsOpts:  "   ",
			fsType:    "ext4",
			expectErr: false,
		},
		{
			name:      "Valid ext3 options",
			mkfsOpts:  "-E lazy_itable_init=0 -m0",
			fsType:    "ext3",
			expectErr: false,
		},
		{
			name:      "Valid ext2 options",
			mkfsOpts:  "-b 4096 -m0",
			fsType:    "ext2",
			expectErr: false,
		},
		{
			name:      "Multiple valid ext4 options",
			mkfsOpts:  "-b 4096 -E lazy_itable_init=0 -F -m0 -O ^has_journal",
			fsType:    "ext4",
			expectErr: false,
		},
		{
			name:      "Multiple valid xfs options",
			mkfsOpts:  "-f -b size=4096 -d agcount=4",
			fsType:    "xfs",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMkfsOptions(tt.mkfsOpts, tt.fsType)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestIsValidMkfsOption(t *testing.T) {
	tests := []struct {
		name     string
		option   string
		fsType   string
		expected bool
	}{
		// ext4 tests
		{name: "ext4 -E option", option: "-E", fsType: "ext4", expected: true},
		{name: "ext4 -E with value", option: "-Elazy_itable_init=0", fsType: "ext4", expected: true},
		{name: "ext4 -m option", option: "-m0", fsType: "ext4", expected: true},
		{name: "ext4 -m with space", option: "-m", fsType: "ext4", expected: true},
		{name: "ext4 -b option", option: "-b", fsType: "ext4", expected: true},
		{name: "ext4 -b with value", option: "-b4096", fsType: "ext4", expected: true},
		{name: "ext4 -F option", option: "-F", fsType: "ext4", expected: true},
		{name: "ext4 -i option", option: "-i", fsType: "ext4", expected: true},
		{name: "ext4 -I option", option: "-I", fsType: "ext4", expected: true},
		{name: "ext4 -J option", option: "-J", fsType: "ext4", expected: true},
		{name: "ext4 -N option", option: "-N", fsType: "ext4", expected: true},
		{name: "ext4 -O option", option: "-O", fsType: "ext4", expected: true},
		{name: "ext4 -O with value", option: "-O^has_journal", fsType: "ext4", expected: true},
		{name: "ext4 -T option", option: "-T", fsType: "ext4", expected: true},
		{name: "ext4 invalid option", option: "-Z", fsType: "ext4", expected: false},
		{name: "ext4 invalid option -X", option: "-X", fsType: "ext4", expected: false},

		// ext3 tests
		{name: "ext3 -E option", option: "-E", fsType: "ext3", expected: true},
		{name: "ext3 -m option", option: "-m0", fsType: "ext3", expected: true},
		{name: "ext3 invalid option", option: "-Z", fsType: "ext3", expected: false},

		// ext2 tests
		{name: "ext2 -b option", option: "-b", fsType: "ext2", expected: true},
		{name: "ext2 -m option", option: "-m", fsType: "ext2", expected: true},
		{name: "ext2 invalid option", option: "-Z", fsType: "ext2", expected: false},

		// xfs tests
		{name: "xfs -f option", option: "-f", fsType: "xfs", expected: true},
		{name: "xfs -b option", option: "-b", fsType: "xfs", expected: true},
		{name: "xfs -b with value", option: "-bsize=4096", fsType: "xfs", expected: true},
		{name: "xfs -d option", option: "-d", fsType: "xfs", expected: true},
		{name: "xfs -d with value", option: "-dagcount=4", fsType: "xfs", expected: true},
		{name: "xfs -i option", option: "-i", fsType: "xfs", expected: true},
		{name: "xfs -l option", option: "-l", fsType: "xfs", expected: true},
		{name: "xfs -m option", option: "-m", fsType: "xfs", expected: true},
		{name: "xfs -n option", option: "-n", fsType: "xfs", expected: true},
		{name: "xfs -r option", option: "-r", fsType: "xfs", expected: true},
		{name: "xfs -s option", option: "-s", fsType: "xfs", expected: true},
		{name: "xfs invalid option", option: "-Z", fsType: "xfs", expected: false},
		{name: "xfs invalid option -X", option: "-X", fsType: "xfs", expected: false},

		// Invalid format tests
		{name: "option without dash", option: "nodash", fsType: "ext4", expected: false},
		{name: "empty option", option: "", fsType: "ext4", expected: false},
		{name: "just dash", option: "-", fsType: "ext4", expected: false},

		// Unknown filesystem type
		{name: "unknown fs type", option: "-b", fsType: "btrfs", expected: false},
		{name: "unknown fs type with valid ext4 option", option: "-E", fsType: "unknown", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidMkfsOption(tt.option, tt.fsType)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for option '%s' with fsType '%s'", tt.expected, result, tt.option, tt.fsType)
			}
		})
	}
}

func TestExtractFormatOptions(t *testing.T) {
	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	tests := []struct {
		name          string
		volumeContext map[string]string
		fsType        string
		expectedOpts  []string
		expectNil     bool
	}{
		{
			name:          "No mkfsOptions",
			volumeContext: map[string]string{},
			fsType:        "ext4",
			expectNil:     true,
		},
		{
			name: "Valid ext4 options",
			volumeContext: map[string]string{
				MkfsOptions: "-E lazy_itable_init=0 -O ^has_journal",
			},
			fsType:       "ext4",
			expectedOpts: []string{"-E", "lazy_itable_init=0", "-O", "^has_journal"},
		},
		{
			name: "Valid ext4 options with comma-separated values",
			volumeContext: map[string]string{
				MkfsOptions: "-E lazy_itable_init=0,lazy_journal_init=0 -m0",
			},
			fsType:       "ext4",
			expectedOpts: []string{"-E", "lazy_itable_init=0,lazy_journal_init=0", "-m0"},
		},
		{
			name: "Valid xfs options",
			volumeContext: map[string]string{
				MkfsOptions: "-f -b size=4096",
			},
			fsType:       "xfs",
			expectedOpts: []string{"-f", "-b", "size=4096"},
		},
		{
			name: "Empty mkfsOptions",
			volumeContext: map[string]string{
				MkfsOptions: "",
			},
			fsType:    "ext4",
			expectNil: true,
		},
		{
			name: "Whitespace only mkfsOptions",
			volumeContext: map[string]string{
				MkfsOptions: "   ",
			},
			fsType:       "ext4",
			expectedOpts: []string{}, // strings.Fields returns empty slice for whitespace
		},
		{
			name: "Invalid options with dangerous characters - should return nil",
			volumeContext: map[string]string{
				MkfsOptions: "-E test; rm -rf /",
			},
			fsType:    "ext4",
			expectNil: true, // Should be rejected and return nil
		},
		{
			name: "Invalid options with pipe - should return nil",
			volumeContext: map[string]string{
				MkfsOptions: "-E test | cat /etc/passwd",
			},
			fsType:    "ext4",
			expectNil: true,
		},
		{
			name: "Invalid option for filesystem type - should return nil",
			volumeContext: map[string]string{
				MkfsOptions: "-Z invalid",
			},
			fsType:    "ext4",
			expectNil: true,
		},
		{
			name: "Valid ext3 options",
			volumeContext: map[string]string{
				MkfsOptions: "-E lazy_itable_init=0 -m0",
			},
			fsType:       "ext3",
			expectedOpts: []string{"-E", "lazy_itable_init=0", "-m0"},
		},
		{
			name: "Multiple xfs options",
			volumeContext: map[string]string{
				MkfsOptions: "-f -b size=4096 -d agcount=4",
			},
			fsType:       "xfs",
			expectedOpts: []string{"-f", "-b", "size=4096", "-d", "agcount=4"},
		},
		{
			name: "VolumeContext with other keys",
			volumeContext: map[string]string{
				"profile":   "general-purpose",
				MkfsOptions: "-m0",
				"encrypted": "false",
			},
			fsType:       "ext4",
			expectedOpts: []string{"-m0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFormatOptions(logger, tt.volumeContext, tt.fsType)

			if tt.expectNil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
			} else {
				if result == nil {
					t.Errorf("Expected %v, got nil", tt.expectedOpts)
				} else {
					assert.Equal(t, tt.expectedOpts, result, "Format options mismatch")
				}
			}
		})
	}
}

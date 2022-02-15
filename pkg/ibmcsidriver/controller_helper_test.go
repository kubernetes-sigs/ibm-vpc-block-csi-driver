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
	"fmt"
	"testing"

	cloudProvider "github.com/IBM/ibm-csi-common/pkg/ibmcloudprovider"
	"github.com/IBM/ibm-csi-common/pkg/utils"
	"github.com/IBM/ibmcloud-volume-interface/config"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
)

const (
	exceededZoneName      = "testzone-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	exceededRegionName    = "us-south-test-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	exceededResourceGID   = "myresourcegroups-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	exceededTag           = "tag-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	exceededEncryptionKey = "key-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

func TestGetRequestedCapacity(t *testing.T) {
	testCases := []struct {
		testCaseName  string
		capRange      *csi.CapacityRange
		expectedValue int64
		expectedError error
	}{
		{
			testCaseName:  "Check minimum size supported by volume provider in case of nil passed as input",
			capRange:      &csi.CapacityRange{},
			expectedValue: utils.MinimumVolumeSizeInBytes,
			expectedError: nil,
		},
		{
			testCaseName:  "Capacity range is nil",
			capRange:      nil,
			expectedValue: utils.MinimumVolumeSizeInBytes,
			expectedError: nil,
		},
		{
			testCaseName: "Check minimum size supported by volume provider",
			capRange: &csi.CapacityRange{RequiredBytes: 1024,
				LimitBytes: utils.MinimumVolumeSizeInBytes},
			expectedValue: utils.MinimumVolumeSizeInBytes,
			expectedError: nil,
		},
		{
			testCaseName: "Check size passed as actual value",
			capRange: &csi.CapacityRange{RequiredBytes: 11811160064,
				LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes}, // MinimumVolumeSizeInBytes->10737418240
			expectedValue: 11811160064,
			expectedError: nil,
		},
		{
			testCaseName: "Expected error check-success",
			capRange: &csi.CapacityRange{RequiredBytes: 1073741824 * 30,
				LimitBytes: utils.MinimumVolumeSizeInBytes}, // MinimumVolumeSizeInBytes->10737418240
			expectedValue: 0,
			expectedError: fmt.Errorf("limit bytes %v is less than required bytes %v", utils.MinimumVolumeSizeInBytes, 1073741824*30),
		},
		{
			testCaseName: "Expected error check against limit byte-success",
			capRange: &csi.CapacityRange{RequiredBytes: utils.MinimumVolumeSizeInBytes - 100,
				LimitBytes: 10737418230}, // MinimumVolumeSizeInBytes->10737418240
			expectedValue: 0,
			expectedError: fmt.Errorf("limit bytes %v is less than minimum volume size: %v", 10737418230, utils.MinimumVolumeSizeInBytes),
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.testCaseName, func(t *testing.T) {
			sizeCap, err := getRequestedCapacity(testcase.capRange)
			if testcase.expectedError != nil {
				assert.Equal(t, err, testcase.expectedError)
			} else {
				expectedValue := testcase.expectedValue
				if sizeCap != expectedValue {
					t.Fatalf("Response expected: %v, got: %v", expectedValue, sizeCap)
				} else {
					assert.Equal(t, sizeCap, expectedValue)
				}
			}
		})
	}
}

func TestAreVolumeCapabilitiesSupported(t *testing.T) {
	testCases := []struct {
		testCaseName  string
		volumeCap     []*csi.VolumeCapability
		expectedValue bool
	}{
		{
			testCaseName:  "Supported volume capability-success",
			volumeCap:     []*csi.VolumeCapability{{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
			expectedValue: true,
		},
		{
			testCaseName:  "Unsupported volume capability",
			volumeCap:     []*csi.VolumeCapability{{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY}}},
			expectedValue: false,
		},
	}

	// Setup test driver
	icDriver := initIBMCSIDriver(t)
	if icDriver == nil {
		t.Fatalf("Failed to setup IBM CSI Driver")
	}

	for _, testcase := range testCases {
		t.Run(testcase.testCaseName, func(t *testing.T) {
			status := areVolumeCapabilitiesSupported(testcase.volumeCap, icDriver.vcap)
			assert.Equal(t, testcase.expectedValue, status)
		})
	}
}

func isVolumeSame(expected *provider.Volume, actual *provider.Volume) bool {
	if actual == nil && expected == nil {
		return true
	}

	if actual == nil || expected == nil {
		return false
	}
	return *actual.Name == *expected.Name &&
		*actual.Capacity == *expected.Capacity &&
		actual.Az == expected.Az &&
		actual.Region == expected.Region
}

func TestGetVolumeParameters(t *testing.T) {
	volumeName := "volName"
	volumeSize := 11
	noIops := ""
	testCases := []struct {
		testCaseName   string
		request        *csi.CreateVolumeRequest
		expectedVolume *provider.Volume
		expectedStatus bool
		expectedError  error
	}{
		{
			testCaseName: "Valid create volume request-success",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 11811160064, LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes},
				VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
				Parameters: map[string]string{Profile: "general-purpose",
					Zone:          "testzone",
					Region:        "us-south-test",
					Tag:           "test-tag",
					ResourceGroup: "myresourcegroups",
					Encrypted:     "false",
					EncryptionKey: "key",
					ClassVersion:  "",
					Generation:    "generation",
					IOPS:          noIops,
				},
			},
			expectedVolume: &provider.Volume{Name: &volumeName,
				Capacity: &volumeSize,
				VPCVolume: provider.VPCVolume{VPCBlockVolume: provider.VPCBlockVolume{
					Tags: []string{createdByIBM},
				},
					Profile:       &provider.Profile{Name: "general-purpose"},
					ResourceGroup: &provider.ResourceGroup{ID: "myresourcegroups"},
				},
				Region: "us-south-test",
				Iops:   &noIops,
				Az:     "testzone",
			},
			expectedStatus: true,
			expectedError:  nil,
		},
		{
			testCaseName:   "Wrong profile name",
			request:        &csi.CreateVolumeRequest{Parameters: map[string]string{Profile: "wrong-profile"}},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("%s:<%v> unsupported profile. Supported profiles are: %v", Profile, "wrong-profile", SupportedProfile),
		},
		{
			testCaseName: "Max length exceeded for zone name",
			request: &csi.CreateVolumeRequest{Parameters: map[string]string{
				Zone: exceededZoneName,
			},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("%s:<%v> exceeds %d chars", Zone, exceededZoneName, ZoneNameMaxLen),
		},
		{
			testCaseName: "Max length exceeded for region name",
			request: &csi.CreateVolumeRequest{Parameters: map[string]string{
				Region: exceededRegionName,
			},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("%s:<%v> exceeds %d chars", Region, exceededRegionName, RegionMaxLen),
		},
		{
			testCaseName: "Max length exceeded for resource group ID",
			request: &csi.CreateVolumeRequest{Parameters: map[string]string{
				ResourceGroup: exceededResourceGID,
			},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("%s:<%v> exceeds %d chars", ResourceGroup, exceededResourceGID, ResourceGroupIDMaxLen),
		},
		{
			testCaseName: "Max length exceeded for tag",
			request: &csi.CreateVolumeRequest{Parameters: map[string]string{
				Tag: exceededTag,
			},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("%s:<%v> exceeds %d chars", Tag, exceededTag, TagMaxLen),
		},
		{
			testCaseName: "Wrong encrypted key's value",
			request: &csi.CreateVolumeRequest{Parameters: map[string]string{
				Encrypted: "noTrueNoFalse",
			},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("'<%v>' is invalid, value of '%s' should be [true|false]", "noTrueNoFalse", Encrypted),
		},
		{
			testCaseName: "Max length exceeded for encryption key",
			request: &csi.CreateVolumeRequest{Parameters: map[string]string{
				EncryptionKey: exceededEncryptionKey,
			},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("%s: exceeds %d bytes", EncryptionKey, EncryptionKeyMaxLen),
		},
		{
			testCaseName: "Unsupported parameter",
			request: &csi.CreateVolumeRequest{Parameters: map[string]string{
				"NotDefineParam": "value",
			},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("<%s> is an invalid parameter", "NotDefineParam"),
		},
		{
			testCaseName: "Invalid capacity range",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 1073741824 * 30, LimitBytes: utils.MinimumVolumeSizeInBytes},
				Parameters: map[string]string{Profile: "10iops-tier",
					IOPS: "10",
				},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("invalid PVC capacity size: '%v'", fmt.Errorf("limit bytes %v is less than required bytes %v", utils.MinimumVolumeSizeInBytes, 1073741824*30)),
		},
		{
			testCaseName: "Override parameter with secrets-wrong secret parameter",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 11811160064, LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes},
				VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
				Parameters:         map[string]string{Profile: "general-purpose"},
				Secrets:            map[string]string{"NotSupportedSecretParam": "value"},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("<%s> is an invalid parameter", "NotSupportedSecretParam"),
		},
		{
			testCaseName: "Empty volume capabilities",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 11811160064, LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes},
				VolumeCapabilities: nil,
				Parameters:         map[string]string{Profile: "general-purpose"},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("volume capabilities are empty"),
		},
		{
			testCaseName: "Region present but zone not given as parameter",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 11811160064, LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes},
				VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
				Parameters: map[string]string{Profile: "general-purpose",
					Region:        "us-south-test",
					Tag:           "test-tag",
					ResourceGroup: "myresourcegroups",
					Encrypted:     "false",
					EncryptionKey: "key",
					ClassVersion:  "",
					Generation:    "generation",
					IOPS:          noIops,
				},
			},
			expectedVolume: &provider.Volume{Name: &volumeName,
				Capacity: &volumeSize,
				VPCVolume: provider.VPCVolume{VPCBlockVolume: provider.VPCBlockVolume{
					Tags: []string{createdByIBM},
				},
					Profile:       &provider.Profile{Name: "general-purpose"},
					ResourceGroup: &provider.ResourceGroup{ID: "myresourcegroups"},
				},
				Region: "us-south-test",
				Iops:   &noIops,
				Az:     "testzone",
			},
			expectedStatus: true,
			expectedError:  fmt.Errorf("zone parameter is empty in storage class for region us-south-test"),
		},
		{
			testCaseName: "Region and Zone not given as parameter from SC",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 11811160064, LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes},
				VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
				Parameters: map[string]string{Profile: "general-purpose",
					Tag:           "test-tag",
					ResourceGroup: "myresourcegroups",
					Encrypted:     "false",
					EncryptionKey: "key",
					ClassVersion:  "",
					Generation:    "generation",
					IOPS:          noIops,
				},
				AccessibilityRequirements: &csi.TopologyRequirement{Preferred: []*csi.Topology{{Segments: map[string]string{
					utils.NodeRegionLabel: "myregion",
					utils.NodeZoneLabel:   "myzone",
				},
				},
				},
				},
			},
			expectedVolume: &provider.Volume{Name: &volumeName,
				Capacity: &volumeSize,
				VPCVolume: provider.VPCVolume{VPCBlockVolume: provider.VPCBlockVolume{
					Tags: []string{createdByIBM},
				},
					Profile:       &provider.Profile{Name: "general-purpose"},
					ResourceGroup: &provider.ResourceGroup{ID: "myresourcegroups"},
				},
				Region: "myregion",
				Iops:   &noIops,
				Az:     "myzone",
			},
			expectedStatus: true,
			expectedError:  nil,
		},
	}

	// Set up
	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	testConfig := &config.Config{
		Server: &config.ServerConfig{
			DebugTrace: true,
		},
		VPC: &config.VPCProviderConfig{
			Enabled:              true,
			VPCBlockProviderName: "vpc-classic",
			EndpointURL:          "TestEndpointURL",
			VPCTimeout:           "30s",
			MaxRetryAttempt:      5,
			MaxRetryGap:          10,
			APIVersion:           "TestAPIVersion",
			ResourceGroupID:      "10000000",
		},
		IKS: &config.IKSConfig{
			Enabled:              true,
			IKSBlockProviderName: "iks-block",
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.testCaseName, func(t *testing.T) {
			actualVolume, err := getVolumeParameters(logger, testcase.request, testConfig)
			if testcase.expectedError != nil {
				assert.Equal(t, err, testcase.expectedError)
			} else {
				assert.Equal(t, testcase.expectedStatus, isVolumeSame(testcase.expectedVolume, actualVolume))
			}
		})
	}
}

func TestIsValidCapacityIOPS4CustomClass(t *testing.T) {
	testCases := []struct {
		testCaseName   string
		requestSize    int
		requestIops    int
		expectedStatus bool
		expectedError  error
	}{
		{
			testCaseName:   "Valid capacity IOPS",
			requestSize:    20,
			requestIops:    110,
			expectedStatus: true,
			expectedError:  nil,
		},
		{
			testCaseName:   "Invalid capacity",
			requestSize:    5,
			requestIops:    110,
			expectedStatus: false,
			expectedError:  fmt.Errorf("invalid PVC size for custom class: <%v>. Should be in range [%d - %d]GiB", 5, utils.MinimumVolumeDiskSizeInGb, utils.MaximumVolumeDiskSizeInGb),
		},
		{
			testCaseName:   "Invalid IOPS",
			requestSize:    20,
			requestIops:    5,
			expectedStatus: false,
			expectedError:  fmt.Errorf("invalid IOPS: <%v> for capacity: <%vGiB>. Should be in range [%d - %d]", 5, 20, customCapacityIopsRanges[0].minIops, customCapacityIopsRanges[0].maxIops),
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.testCaseName, func(t *testing.T) {
			isValid, err := isValidCapacityIOPS4CustomClass(testcase.requestSize, testcase.requestIops)
			if testcase.expectedError != nil {
				assert.Equal(t, err, testcase.expectedError)
			} else {
				assert.Equal(t, testcase.expectedStatus, isValid)
			}
		})
	}
}

func TestOverrideParams(t *testing.T) {
	volumeName := "volName"
	volumeSize := 11 // in Gib which is equal to 11811160064 byte
	noIops := ""
	iops110 := "110"
	secretInvalidIops := "aa5" // For 10GB
	testCases := []struct {
		testCaseName   string
		request        *csi.CreateVolumeRequest
		expectedVolume *provider.Volume
		expectedStatus bool
		expectedError  error
	}{
		{
			testCaseName: "Valid overwrite-success",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 11811160064, LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes},
				VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
				Parameters: map[string]string{Profile: "general-purpose",
					Zone:          "testzone",
					Region:        "us-south-test",
					Tag:           "test",
					ResourceGroup: "myresourcegroups",
					Encrypted:     "false",
					EncryptionKey: "123",
					IOPS:          noIops,
				},
				Secrets: map[string]string{
					Zone:          "secret-zone",
					Region:        "secret-us-south-test",
					Tag:           "secret-tag",
					ResourceGroup: "secret-rg",
					Encrypted:     "false",
					EncryptionKey: "1231",
					IOPS:          noIops,
				},
			},
			expectedVolume: &provider.Volume{Name: &volumeName,
				Capacity: &volumeSize,
				VPCVolume: provider.VPCVolume{VPCBlockVolume: provider.VPCBlockVolume{
					Tags: []string{createdByIBM},
				},
					Profile:       &provider.Profile{Name: "general-purpose"},
					ResourceGroup: &provider.ResourceGroup{ID: "secret-rg"},
				},
				Region: "secret-us-south-test",
				Iops:   &noIops,
				Az:     "secret-zone",
			},
			expectedStatus: true,
			expectedError:  nil,
		},
		{
			testCaseName: "Secret wrong encrypted value",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 11811160064, LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes},
				Secrets: map[string]string{
					Encrypted: "false11",
				},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("<%v> is invalid, value for '%s' should be [true|false]", "false11", Encrypted),
		},
		{
			testCaseName: "Resource group ID size exceeded",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 11811160064, LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes},
				Secrets: map[string]string{
					ResourceGroup: exceededResourceGID,
				},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("%s:<%v> exceeds %d bytes ", ResourceGroup, exceededResourceGID, ResourceGroupIDMaxLen),
		},
		{
			testCaseName: "Encryption key size exceeded",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 11811160064, LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes},
				Secrets: map[string]string{
					EncryptionKey: exceededEncryptionKey,
				},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("%s exceeds %d bytes", EncryptionKey, EncryptionKeyMaxLen),
		},
		{
			testCaseName: "Tag key size exceeded",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 11811160064, LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes},
				Secrets: map[string]string{
					Tag: exceededTag,
				},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("%s:<%v> exceeds %d chars", Tag, exceededTag, TagMaxLen),
		},
		{
			testCaseName: "Zone key size exceeded",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 11811160064, LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes},
				Secrets: map[string]string{
					Zone: exceededZoneName,
				},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("%s:<%v> exceeds %d chars", Zone, exceededZoneName, ZoneNameMaxLen),
		},
		{
			testCaseName: "Region key size exceeded",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 11811160064, LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes},
				Secrets: map[string]string{
					Region: exceededRegionName,
				},
			},
			expectedVolume: &provider.Volume{},
			expectedStatus: true,
			expectedError:  fmt.Errorf("%s:<%v> exceeds %d chars", Region, exceededRegionName, RegionMaxLen),
		},
		{
			testCaseName: "Valid IOPS for custom class",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 11811160064, LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes},
				Parameters: map[string]string{Profile: "custom",
					Zone:          "testzone",
					Region:        "us-south-test",
					Tag:           "test",
					ResourceGroup: "myresourcegroups",
					Encrypted:     "false",
					EncryptionKey: "123",
					IOPS:          noIops,
				},
				Secrets: map[string]string{
					IOPS: iops110,
				},
			},
			expectedVolume: &provider.Volume{Name: &volumeName,
				Capacity: &volumeSize,
				VPCVolume: provider.VPCVolume{VPCBlockVolume: provider.VPCBlockVolume{
					Tags: []string{createdByIBM},
				},
					Profile:       &provider.Profile{Name: "custom"},
					ResourceGroup: &provider.ResourceGroup{ID: "myresourcegroups"},
				},
				Az:   "testzone",
				Iops: &iops110,
			},
			expectedStatus: true,
			expectedError:  nil,
		},
		{
			testCaseName: "Secret invalid IOPS for custom class",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 11811160064, LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes},
				Parameters: map[string]string{Profile: "custom",
					Zone:          "testzone",
					Region:        "us-south-test",
					Tag:           "test",
					ResourceGroup: "myresourcegroups",
					Encrypted:     "false",
					EncryptionKey: "123",
					IOPS:          noIops,
				},
				Secrets: map[string]string{
					IOPS: secretInvalidIops,
				},
			},
			expectedVolume: &provider.Volume{Name: &volumeName,
				Capacity:  &volumeSize,
				VPCVolume: provider.VPCVolume{Profile: &provider.Profile{Name: "custom"}},
			},
			expectedStatus: false,
			expectedError:  fmt.Errorf("%v:<%v> invalid value", IOPS, secretInvalidIops),
		},
		{
			testCaseName: "Nil volume as input/output",
			request: &csi.CreateVolumeRequest{Name: volumeName, CapacityRange: &csi.CapacityRange{RequiredBytes: 11811160064, LimitBytes: utils.MinimumVolumeSizeInBytes + utils.MinimumVolumeSizeInBytes},
				Parameters: map[string]string{Profile: "custom"},
				Secrets: map[string]string{
					IOPS: iops110,
				},
			},
			expectedVolume: nil,
			expectedStatus: true,
			expectedError:  fmt.Errorf("invalid volume parameter"),
		},
	}

	testConfig := &config.Config{
		Server: &config.ServerConfig{
			DebugTrace: true,
		},
		VPC: &config.VPCProviderConfig{
			Enabled:              true,
			VPCBlockProviderName: "vpc-classic",
			EndpointURL:          "TestEndpointURL",
			VPCTimeout:           "30s",
			MaxRetryAttempt:      5,
			MaxRetryGap:          10,
			APIVersion:           "TestAPIVersion",
			ResourceGroupID:      "10000000",
		},
		IKS: &config.IKSConfig{
			Enabled:              true,
			IKSBlockProviderName: "iks-block",
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	for _, testcase := range testCases {
		t.Run(testcase.testCaseName, func(t *testing.T) {
			volumeOut := testcase.expectedVolume
			err := overrideParams(logger, testcase.request, testConfig, volumeOut)
			if testcase.expectedError != nil {
				assert.Equal(t, err, testcase.expectedError)
			} else {
				assert.Equal(t, testcase.expectedStatus, isVolumeSame(testcase.expectedVolume, volumeOut))
			}
		})
	}
}

func isCSIResponseSame(expectedVolume *csi.CreateVolumeResponse, actualCSIVolume *csi.CreateVolumeResponse) bool {
	if expectedVolume == nil && actualCSIVolume == nil {
		return true
	}

	if expectedVolume == nil || actualCSIVolume == nil {
		return false
	}

	return expectedVolume.Volume.VolumeId == actualCSIVolume.Volume.VolumeId &&
		expectedVolume.Volume.CapacityBytes == actualCSIVolume.Volume.CapacityBytes &&
		expectedVolume.Volume.GetAccessibleTopology()[0].GetSegments()[utils.NodeRegionLabel] == actualCSIVolume.Volume.GetAccessibleTopology()[0].GetSegments()[utils.NodeRegionLabel] &&
		expectedVolume.Volume.GetAccessibleTopology()[0].GetSegments()[utils.NodeZoneLabel] == actualCSIVolume.Volume.GetAccessibleTopology()[0].GetSegments()[utils.NodeZoneLabel]
}

func TestCheckIfVolumeExists(t *testing.T) {
}

func TestCreateCSIVolumeResponse(t *testing.T) {
	volumeID := "volID"
	threeIops := "3"
	testCases := []struct {
		testCaseName   string
		requestVol     provider.Volume
		requestCap     int64
		requestZones   []string
		clusterID      string
		expectedVolume *csi.CreateVolumeResponse
		expectedStatus bool
	}{
		{
			testCaseName: "Valid volume response",
			requestVol: provider.Volume{VolumeID: volumeID,
				VPCVolume: provider.VPCVolume{VPCBlockVolume: provider.VPCBlockVolume{
					Tags: []string{createdByIBM},
				},
					Profile:       &provider.Profile{Name: "general-purpose"},
					ResourceGroup: &provider.ResourceGroup{ID: "myresourcegroups"},
				},
				Region: "us-south-test",
				Iops:   &threeIops,
				Az:     "testzone",
			},
			requestCap:   20,
			clusterID:    "1234",
			requestZones: []string{"", ""},
			expectedVolume: &csi.CreateVolumeResponse{
				Volume: &csi.Volume{
					CapacityBytes: 20,
					VolumeId:      volumeID,
					VolumeContext: map[string]string{VolumeIDLabel: volumeID, IOPSLabel: threeIops, utils.NodeRegionLabel: "us-south-test", utils.NodeZoneLabel: "testzone"},
					AccessibleTopology: []*csi.Topology{{
						Segments: map[string]string{
							utils.NodeRegionLabel: "us-south-test",
							utils.NodeZoneLabel:   "testzone",
						},
					},
					},
				},
			},
			expectedStatus: true,
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.testCaseName, func(t *testing.T) {
			actualCSIVolume := createCSIVolumeResponse(testcase.requestVol, testcase.requestCap, testcase.requestZones, testcase.clusterID)
			assert.Equal(t, testcase.expectedStatus, isCSIResponseSame(testcase.expectedVolume, actualCSIVolume))
		})
	}
}

func isControllerPublishVolume(expected *csi.ControllerPublishVolumeResponse, actual *csi.ControllerPublishVolumeResponse) bool {
	if expected == nil && actual == nil {
		return true
	}

	if expected == nil || actual == nil {
		return false
	}

	return expected.PublishContext[PublishInfoVolumeID] == actual.PublishContext[PublishInfoVolumeID]
}

func TestCreateControllerPublishVolumeResponse(t *testing.T) {
	testCases := []struct {
		testCaseName              string
		requestVolAttResponse     provider.VolumeAttachmentResponse
		extraPublishInfo          map[string]string
		expectedCtlPubVolResponse *csi.ControllerPublishVolumeResponse
		expectedStatus            bool
	}{
		{
			testCaseName: "Valid controller volume response",
			requestVolAttResponse: provider.VolumeAttachmentResponse{
				Status: "available",
				VolumeAttachmentRequest: provider.VolumeAttachmentRequest{
					VolumeID:            "volumeID",
					InstanceID:          "instanceID",
					VPCVolumeAttachment: &provider.VolumeAttachment{DevicePath: "/dev/xbv"},
				},
			},
			extraPublishInfo: map[string]string{},
			expectedCtlPubVolResponse: &csi.ControllerPublishVolumeResponse{
				PublishContext: map[string]string{
					PublishInfoVolumeID:   "volumeID",
					PublishInfoNodeID:     "instanceID",
					PublishInfoStatus:     "available",
					PublishInfoDevicePath: "/dev/xbv",
				},
			},
			expectedStatus: true,
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.testCaseName, func(t *testing.T) {
			actualCtlPubVol := createControllerPublishVolumeResponse(testcase.requestVolAttResponse, testcase.extraPublishInfo)
			assert.Equal(t, testcase.expectedStatus, isControllerPublishVolume(testcase.expectedCtlPubVolResponse, actualCtlPubVol))
		})
	}
}

func TestPickTargetTopologyParams(t *testing.T) {
	//pickTargetTopologyParams(top *csi.TopologyRequirement) (map[string]string, error)
	testCases := []struct {
		testCaseName    string
		requestTopology *csi.TopologyRequirement
		expectedOutput  map[string]string
		expectedError   error
	}{
		{
			testCaseName: "Valid pick target for topology",
			requestTopology: &csi.TopologyRequirement{Preferred: []*csi.Topology{{Segments: map[string]string{
				utils.NodeRegionLabel: "us-south-test",
				utils.NodeZoneLabel:   "testzone",
			},
			},
			},
			},
			expectedOutput: map[string]string{utils.NodeRegionLabel: "us-south-test",
				utils.NodeZoneLabel: "testzone",
			},
			expectedError: nil,
		},
		{
			testCaseName:    "Nil pick target for topology",
			requestTopology: &csi.TopologyRequirement{Preferred: []*csi.Topology{}},
			expectedOutput:  nil,
			expectedError:   fmt.Errorf("could not get zones from preferred topology: %v", fmt.Errorf("preferred topologies specified but no segments")),
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.testCaseName, func(t *testing.T) {
			actualCtlPubVol, err := pickTargetTopologyParams(testcase.requestTopology)
			if testcase.expectedError == nil {
				assert.Equal(t, testcase.expectedOutput, actualCtlPubVol)
			} else {
				assert.Equal(t, testcase.expectedError, err)
			}
		})
	}
}

func TestGetPrefedTopologyParams(t *testing.T) {
	testCases := []struct {
		testCaseName    string
		requestTopology []*csi.Topology
		expectedOutput  map[string]string
		expectedError   error
	}{
		{
			testCaseName: "Valid preferred topology params",
			requestTopology: []*csi.Topology{{Segments: map[string]string{
				utils.NodeRegionLabel: "us-south-test",
				utils.NodeZoneLabel:   "testzone",
			},
			},
			},
			expectedOutput: map[string]string{utils.NodeRegionLabel: "us-south-test",
				utils.NodeZoneLabel: "testzone",
			},
			expectedError: nil,
		},
		{
			testCaseName:    "With nil preferred topology params",
			requestTopology: []*csi.Topology{},
			expectedOutput:  nil,
			expectedError:   fmt.Errorf("preferred topologies specified but no segments"),
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.testCaseName, func(t *testing.T) {
			actualCtlPubVol, err := getPrefedTopologyParams(testcase.requestTopology)
			if testcase.expectedError == nil {
				assert.Equal(t, testcase.expectedOutput, actualCtlPubVol)
			} else {
				assert.Equal(t, testcase.expectedError, err)
			}
		})
	}
}

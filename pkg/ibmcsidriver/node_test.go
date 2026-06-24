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

// Package ibmcsidriver ...
package ibmcsidriver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"k8s.io/utils/exec"
	testingexec "k8s.io/utils/exec/testing"

	"github.com/IBM/ibm-csi-common/pkg/utils"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const defaultVolumeID = "csiprovidervolumeid"
const defaultTargetPath = "/mnt/test"
const defaultStagingPath = "/staging"
const defaultVolumePath = "/var/volpath"

const errorDeviceInfo = "/for/errordevicepath"
const errorBlockDevice = "/for/errorblock"
const notBlockDevice = "/for/notblocktest"

type MockStatUtils struct {
}

func (su *MockStatUtils) FSInfo(path string) (int64, int64, int64, int64, int64, int64, error) {
	return 1, 1, 1, 1, 1, 1, nil
}

func (su *MockStatUtils) DeviceInfo(path string) (int64, error) {
	if strings.Contains(path, "errordevicepath") {
		return 1, errors.New("error in getting device info")
	}
	return 1, nil
}

func (su *MockStatUtils) IsBlockDevice(devicePath string) (bool, error) {
	if strings.Contains(devicePath, "errorblock") {
		return false, errors.New("error in IsBlockDevice check")
	} else if strings.Contains(devicePath, "notblock") {
		return false, nil
	}
	return true, nil
}

func (su *MockStatUtils) IsDevicePathNotExist(devicePath string) bool {
	return strings.Contains(devicePath, "correctdevicepath")
}

func TestNodePublishVolume(t *testing.T) {
	// Set environment variables for fast retry in tests
	t.Setenv("UDEVADM_MAX_RETRIES", "2")
	t.Setenv("UDEVADM_RETRY_INTERVAL", "10ms")

	testCases := []struct {
		name       string
		req        *csi.NodePublishVolumeRequest
		expErrCode codes.Code
	}{
		{
			name: "Valid request",
			req: &csi.NodePublishVolumeRequest{
				VolumeId:          defaultVolumeID,
				TargetPath:        defaultTargetPath,
				StagingTargetPath: defaultStagingPath,
				Readonly:          false,
				VolumeCapability:  stdVolCap[0],
			},
			expErrCode: codes.OK,
		},
		{
			name: "Empty volume ID",
			req: &csi.NodePublishVolumeRequest{
				VolumeId:          "",
				TargetPath:        defaultTargetPath,
				StagingTargetPath: defaultStagingPath,
				Readonly:          false,
				VolumeCapability:  stdVolCap[0],
			},
			expErrCode: codes.InvalidArgument,
		},
		{
			name: "Empty staging target path",
			req: &csi.NodePublishVolumeRequest{
				VolumeId:          "testvolumeid",
				TargetPath:        defaultTargetPath,
				StagingTargetPath: "",
				Readonly:          false,
				VolumeCapability:  stdVolCap[0],
			},
			expErrCode: codes.InvalidArgument,
		},
		{
			name: "Empty target path",
			req: &csi.NodePublishVolumeRequest{
				VolumeId:          "testvolumeid",
				TargetPath:        "",
				StagingTargetPath: defaultTargetPath,
				Readonly:          false,
				VolumeCapability:  stdVolCap[0],
			},
			expErrCode: codes.InvalidArgument,
		},
		{
			name: "Empty volume capabilities",
			req: &csi.NodePublishVolumeRequest{
				VolumeId:          "testvolumeid",
				TargetPath:        defaultTargetPath,
				StagingTargetPath: defaultStagingPath,
				Readonly:          false,
				VolumeCapability:  nil,
			},
			expErrCode: codes.InvalidArgument,
		},
		{
			name: "Not supported volume capabilities",
			req: &csi.NodePublishVolumeRequest{
				VolumeId:          "testvolumeid",
				TargetPath:        defaultTargetPath,
				StagingTargetPath: defaultStagingPath,
				Readonly:          false,
				VolumeCapability:  stdVolCapNotSupported[0],
			},
			expErrCode: codes.InvalidArgument,
		},
		{
			name: "Raw block request with device path not found",
			req: &csi.NodePublishVolumeRequest{
				VolumeId:          defaultVolumeID,
				TargetPath:        defaultTargetPath,
				StagingTargetPath: defaultStagingPath,
				PublishContext:    map[string]string{PublishInfoDevicePath: "/dev/sda"},
				Readonly:          false,
				VolumeCapability:  stdBlockVolCap[0],
			},
			expErrCode: codes.Internal,
		},
		{
			name: "Raw block request with invaliddevice",
			req: &csi.NodePublishVolumeRequest{
				VolumeId:          defaultVolumeID,
				TargetPath:        defaultTargetPath,
				StagingTargetPath: defaultStagingPath,
				PublishContext:    map[string]string{PublishInfoDevicePath: ""},
				Readonly:          false,
				VolumeCapability:  stdBlockVolCap[0],
			},
			expErrCode: codes.InvalidArgument,
		},
		{
			name: "Raw block request with invalidTarget",
			req: &csi.NodePublishVolumeRequest{
				VolumeId:          defaultVolumeID,
				TargetPath:        "",
				StagingTargetPath: defaultStagingPath,
				PublishContext:    map[string]string{PublishInfoDevicePath: "/dev/sda"},
				Readonly:          false,
				VolumeCapability:  stdBlockVolCap[0],
			},
			expErrCode: codes.InvalidArgument,
		},
	}

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
		_, err := icDriver.ns.NodePublishVolume(context.Background(), tc.req)
		if err != nil {
			serverError, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Could not get error status code from err: %v", err)
			}
			if serverError.Code() != tc.expErrCode {
				t.Fatalf("Expected error code: %v, got: %v. err : %v", tc.expErrCode, serverError.Code(), err)
			}
			continue
		}
		if tc.expErrCode != codes.OK {
			t.Fatalf("Expected error: %v, got no error", tc.expErrCode)
		}
	}
}

func TestNodeUnpublishVolume(t *testing.T) {
	testCases := []struct {
		name       string
		req        *csi.NodeUnpublishVolumeRequest
		expErrCode codes.Code
	}{
		{
			name: "Valid request",
			req: &csi.NodeUnpublishVolumeRequest{
				VolumeId:   defaultVolumeID,
				TargetPath: defaultTargetPath,
			},
			expErrCode: codes.OK,
		},
		{
			name: "Empty volume ID",
			req: &csi.NodeUnpublishVolumeRequest{
				VolumeId:   "",
				TargetPath: defaultTargetPath,
			},
			expErrCode: codes.InvalidArgument,
		},
		{
			name: "Empty target path",
			req: &csi.NodeUnpublishVolumeRequest{
				VolumeId:   defaultVolumeID,
				TargetPath: "",
			},
			expErrCode: codes.InvalidArgument,
		},
	}

	icDriver := initIBMCSIDriver(t)

	for _, tc := range testCases {
		t.Logf("Test case: %s", tc.name)
		_, err := icDriver.ns.NodeUnpublishVolume(context.Background(), tc.req)
		if err != nil {
			serverError, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Could not get error status code from err: %v", err)
			}
			if serverError.Code() != tc.expErrCode {
				t.Fatalf("Expected error code: %v, got: %v. err : %v", tc.expErrCode, serverError.Code(), err)
			}
			continue
		}
		if tc.expErrCode != codes.OK {
			t.Fatalf("Expected error: %v, got no error", tc.expErrCode)
		}
	}
}

func TestNodeStageVolume(t *testing.T) {
	// Set environment variable to skip sleep in tests
	t.Setenv("UDEVADM_SLEEP_DURATION", "0s")

	volumeID := "newstagevolumeID"
	testCases := []struct {
		name       string
		req        *csi.NodeStageVolumeRequest
		expErrCode codes.Code
	}{
		{
			name: "Device path not found",
			req: &csi.NodeStageVolumeRequest{
				VolumeId:          volumeID,
				StagingTargetPath: defaultStagingPath,
				VolumeCapability:  stdVolCap[0],
				PublishContext:    map[string]string{PublishInfoDevicePath: "/dev/nonexistent"},
			},
			expErrCode: codes.Internal,
		},
		{
			name: "Empty volume ID",
			req: &csi.NodeStageVolumeRequest{
				VolumeId:          "",
				StagingTargetPath: defaultStagingPath,
				VolumeCapability:  stdVolCap[0],
				PublishContext:    map[string]string{PublishInfoDevicePath: "/dev"},
			},
			expErrCode: codes.InvalidArgument,
		},
		{
			name: "Empty target path",
			req: &csi.NodeStageVolumeRequest{
				VolumeId:          volumeID,
				StagingTargetPath: "",
				VolumeCapability:  stdVolCap[0],
				PublishContext:    map[string]string{PublishInfoDevicePath: "/dev"},
			},
			expErrCode: codes.InvalidArgument,
		},
		{
			name: "Empty volume capabilities",
			req: &csi.NodeStageVolumeRequest{
				VolumeId:          volumeID,
				StagingTargetPath: defaultTargetPath,
				VolumeCapability:  nil,
				PublishContext:    map[string]string{PublishInfoDevicePath: "/dev"},
			},
			expErrCode: codes.InvalidArgument,
		},
		{
			name: "Not supported volume capabilities",
			req: &csi.NodeStageVolumeRequest{
				VolumeId:          volumeID,
				StagingTargetPath: defaultTargetPath,
				VolumeCapability:  stdVolCapNotSupported[0],
				PublishContext:    map[string]string{PublishInfoDevicePath: "/dev"},
			},
			expErrCode: codes.InvalidArgument,
		},
		{
			name: "Empty device path in the context",
			req: &csi.NodeStageVolumeRequest{
				VolumeId:          volumeID,
				StagingTargetPath: defaultTargetPath,
				VolumeCapability:  stdVolCap[0],
				PublishContext:    map[string]string{PublishInfoDevicePath: ""},
			},
			expErrCode: codes.InvalidArgument,
		},
		{
			name: "Valid raw block StageVolume request",
			req: &csi.NodeStageVolumeRequest{
				VolumeId:          volumeID,
				StagingTargetPath: defaultStagingPath,
				VolumeCapability:  stdBlockVolCap[0],
				PublishContext:    map[string]string{PublishInfoDevicePath: "/dev/sda"},
			},
			expErrCode: codes.OK,
		},
	}

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
						return []byte("DEVNAME=/dev/sdb\nTYPE=ext4"), nil, nil
					},
				},
			},
			"blkid",
		),
		makeFakeCmd(
			&testingexec.FakeCmd{
				CombinedOutputScript: []testingexec.FakeAction{
					func() ([]byte, []byte, error) {
						return []byte("1"), nil, nil
					},
				},
			},
			"mkfs.ext2",
		),
		makeFakeCmd(
			&testingexec.FakeCmd{
				CombinedOutputScript: []testingexec.FakeAction{
					func() ([]byte, []byte, error) {
						return []byte("1"), nil, nil
					},
				},
			},
			"blockdev",
		),
		makeFakeCmd(
			&testingexec.FakeCmd{
				CombinedOutputScript: []testingexec.FakeAction{
					func() ([]byte, []byte, error) {
						return []byte("DEVNAME=/dev/sdb\nTYPE=ext4"), nil, nil
					},
				},
			},
			"blkid",
		),
		makeFakeCmd(
			&testingexec.FakeCmd{
				CombinedOutputScript: []testingexec.FakeAction{
					func() ([]byte, []byte, error) {
						return []byte("block size: 1\nblock count: 1"), nil, nil
					},
				},
			},
			"dumpe2fs",
		),
	}

	icDriver := initIBMCSIDriver(t, actionList...)
	for _, tc := range testCases {
		t.Logf("Test case: %s", tc.name)
		_, err := icDriver.ns.NodeStageVolume(context.Background(), tc.req)
		if err != nil {
			serverError, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Could not get error status code from err: %v", err)
			}
			if serverError.Code() != tc.expErrCode {
				t.Fatalf("Expected error code: %v, got: %v. err : %v", tc.expErrCode, serverError.Code(), err)
			}
			continue
		}
		if tc.expErrCode != codes.OK {
			t.Fatalf("Expected error: %v, got no error", tc.expErrCode)
		}
	}
}

func TestNodeUnstageVolume(t *testing.T) {
	testCases := []struct {
		name       string
		req        *csi.NodeUnstageVolumeRequest
		expErrCode codes.Code
	}{
		{
			name: "Valid request",
			req: &csi.NodeUnstageVolumeRequest{
				VolumeId:          defaultVolumeID,
				StagingTargetPath: defaultTargetPath,
			},
			expErrCode: codes.OK,
		},
		{
			name: "Empty volume ID",
			req: &csi.NodeUnstageVolumeRequest{
				VolumeId:          "",
				StagingTargetPath: defaultStagingPath,
			},
			expErrCode: codes.InvalidArgument,
		},
		{
			name: "Empty target path",
			req: &csi.NodeUnstageVolumeRequest{
				VolumeId:          defaultVolumeID,
				StagingTargetPath: "",
			},
			expErrCode: codes.InvalidArgument,
		},
	}

	icDriver := initIBMCSIDriver(t)
	for _, tc := range testCases {
		t.Logf("Test case: %s", tc.name)
		_, err := icDriver.ns.NodeUnstageVolume(context.Background(), tc.req)
		if err != nil {
			serverError, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Could not get error status code from err: %v", err)
			}
			if serverError.Code() != tc.expErrCode {
				t.Fatalf("Expected error code: %v, got: %v. err : %v", tc.expErrCode, serverError.Code(), err)
			}
			continue
		}
		if tc.expErrCode != codes.OK {
			t.Fatalf("Expected error: %v, got no error", tc.expErrCode)
		}
	}
}

func TestNodeGetCapabilities(t *testing.T) {
	req := &csi.NodeGetCapabilitiesRequest{}

	icDriver := initIBMCSIDriver(t)
	_, err := icDriver.ns.NodeGetCapabilities(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpedted error: %v", err)
	}
}

func TestNodeGetInfo(t *testing.T) {
	var maxVolumesPerNode int64 = DefaultVolumesPerNode

	testCases := []struct {
		name          string
		req           *csi.NodeGetInfoRequest
		resetMetadata bool
		resp          *csi.NodeGetInfoResponse
		expErrCode    codes.Code
		expError      error
	}{
		{
			name:          "Success to get node info",
			req:           &csi.NodeGetInfoRequest{},
			resetMetadata: false,
			resp: &csi.NodeGetInfoResponse{
				NodeId:            "testworker",
				MaxVolumesPerNode: maxVolumesPerNode,
				AccessibleTopology: &csi.Topology{
					Segments: map[string]string{
						utils.NodeRegionLabel: "testregion",
						utils.NodeZoneLabel:   "testzone",
					},
				},
			},
			expErrCode: codes.OK,
			expError:   nil,
		},
		{
			name:          "No node data service set",
			req:           &csi.NodeGetInfoRequest{},
			resetMetadata: true,
			resp:          nil,
			expErrCode:    codes.NotFound,
			expError:      fmt.Errorf("any error is fine because error code is getting verified"),
		},
	}

	icDriver := initIBMCSIDriver(t)
	for _, tc := range testCases {
		if tc.resetMetadata {
			icDriver.ns.Metadata = nil
		}
		response, err := icDriver.ns.NodeGetInfo(context.Background(), tc.req)
		if err != nil {
			serverError, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Could not get error status code from err: %v", err)
			}
			assert.Equal(t, tc.expErrCode, serverError.Code())
		} else {
			assert.Nil(t, err)
			assert.Equal(t, tc.resp, response)
		}
	}
}

func TestNodeGetVolumeStats(t *testing.T) {
	testCases := []struct {
		name       string
		req        *csi.NodeGetVolumeStatsRequest
		resp       *csi.NodeGetVolumeStatsResponse
		expErrCode codes.Code
		expError   string
	}{
		{
			name: "Mode is block",
			req: &csi.NodeGetVolumeStatsRequest{
				VolumeId:   defaultVolumeID,
				VolumePath: defaultVolumePath,
			},
			resp: &csi.NodeGetVolumeStatsResponse{
				Usage: []*csi.VolumeUsage{
					{
						Total: 1,
						Unit:  1,
					},
				},
			},
			expErrCode: codes.OK,
			expError:   "",
		},
		{
			name: "Empty volume ID",
			req: &csi.NodeGetVolumeStatsRequest{
				VolumeId:   "",
				VolumePath: defaultVolumePath,
			},
			resp:       nil,
			expErrCode: codes.InvalidArgument,
			expError:   "",
		},
		{
			name: "Empty volume path",
			req: &csi.NodeGetVolumeStatsRequest{
				VolumeId:   defaultVolumeID,
				VolumePath: "",
			},
			resp:       nil,
			expErrCode: codes.InvalidArgument,
			expError:   "",
		},
		{
			name: "Mode is File",
			req: &csi.NodeGetVolumeStatsRequest{
				VolumeId:   defaultVolumeID,
				VolumePath: notBlockDevice,
			},
			resp: &csi.NodeGetVolumeStatsResponse{
				Usage: []*csi.VolumeUsage{
					{
						Available: 1,
						Total:     1,
						Used:      1,
						Unit:      1,
					},
					{
						Available: 1,
						Total:     1,
						Used:      1,
						Unit:      2,
					},
				},
			},
			expErrCode: codes.OK,
			expError:   "",
		},
		{
			name: "Error in checking block device",
			req: &csi.NodeGetVolumeStatsRequest{
				VolumeId:   defaultVolumeID,
				VolumePath: errorBlockDevice,
			},
			resp:     nil,
			expError: "Failed to determine if volume is block",
		},
		{
			name: "Failed to get block size",
			req: &csi.NodeGetVolumeStatsRequest{
				VolumeId:   defaultVolumeID,
				VolumePath: errorDeviceInfo,
			},
			resp:     nil,
			expError: "Failed to get size of block volume",
		},
	}
	icDriver := initIBMCSIDriver(t)
	for _, tc := range testCases {
		t.Logf("Test case: %s", tc.name)
		fmt.Println(tc.resp)
		resp, err := icDriver.ns.NodeGetVolumeStats(context.Background(), tc.req)
		if !proto.Equal(resp, tc.resp) {
			t.Fatalf("Expected response: %v, got: %v", tc.resp, resp)
		}
		if tc.expError != "" {
			assert.NotNil(t, err)
			continue
		}
		if err != nil {
			serverError, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Could not get error status code from err: %v", err)
			}
			if serverError.Code() != tc.expErrCode {
				t.Fatalf("Expected error code: %v, got: %v. err : %v", tc.expErrCode, serverError.Code(), err)
			}
			continue
		}
		if tc.expErrCode != codes.OK {
			t.Fatalf("Expected error: %v, got no error", tc.expErrCode)
		}
	}
}

func TestNodeExpandVolume(t *testing.T) {
	// Set environment variable to skip sleep in tests
	t.Setenv("UDEVADM_SLEEP_DURATION", "0s")

	testCases := []struct {
		name       string
		req        *csi.NodeExpandVolumeRequest
		expErrCode codes.Code
	}{
		{
			name: "Empty volume Path",
			req: &csi.NodeExpandVolumeRequest{
				VolumeId:   defaultVolumeID,
				VolumePath: "",
			},
			expErrCode: codes.InvalidArgument,
		},
		{
			name: "Invalid volumePath",
			req: &csi.NodeExpandVolumeRequest{
				VolumeId:   defaultVolumeID,
				VolumePath: "/invalid-volPath_notblock",
			},
			expErrCode: codes.NotFound,
		},
		{
			name: "valid volumePath",
			req: &csi.NodeExpandVolumeRequest{
				VolumeId:   defaultVolumeID,
				VolumePath: "valid-vol-path",
				CapacityRange: &csi.CapacityRange{
					RequiredBytes: 20 * 1024 * 1024 * 1024,
				},
			},
			expErrCode: codes.OK,
		},
		{
			name: "volumePath not mounted",
			req: &csi.NodeExpandVolumeRequest{
				VolumeId:   defaultVolumeID,
				VolumePath: "fake-volPath_notblock",
			},
			expErrCode: codes.NotFound,
		},
	}

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
						return []byte("1"), nil, nil
					},
				},
			},
			"blockdev",
		),
		makeFakeCmd(
			&testingexec.FakeCmd{
				CombinedOutputScript: []testingexec.FakeAction{
					func() ([]byte, []byte, error) {
						return []byte("DEVNAME=/dev/sdb\nTYPE=ext4"), nil, nil
					},
				},
			},
			"blkid",
		),
		makeFakeCmd(
			&testingexec.FakeCmd{
				CombinedOutputScript: []testingexec.FakeAction{
					func() ([]byte, []byte, error) {
						return []byte("block size: 1\nblock count: 1"), nil, nil
					},
				},
			},
			"dumpe2fs",
		),
	}

	icDriver := initIBMCSIDriver(t, actionList...)
	_ = os.MkdirAll("valid-vol-path", os.FileMode(0755))
	_ = icDriver.ns.Mounter.Mount("valid-devicePath", "valid-vol-path", "ext4", []string{})
	for _, tc := range testCases {
		t.Logf("Test case: %s", tc.name)
		_, err := icDriver.ns.NodeExpandVolume(context.Background(), tc.req)
		if err != nil {
			serverError, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Could not get error status code from err: %v", err)
			}
			if serverError.Code() != tc.expErrCode {
				t.Fatalf("Expected error code: %v, got: %v. err : %v", tc.expErrCode, serverError.Code(), err)
			}
			continue
		}
		if tc.expErrCode != codes.OK {
			t.Fatalf("Expected error: %v, got no error", tc.expErrCode)
		}
	}
	_ = os.RemoveAll("valid-vol-path")
}

func TestIsBlockDevice(t *testing.T) {
	testCases := []struct {
		name          string
		reqDevicePath string
		yes           bool
		respError     error
	}{
		{
			name:          "Not a valid path, hence its not block device",
			reqDevicePath: "/tmp111111111111111",
			yes:           false,
			respError:     fmt.Errorf("any error is fine"),
		},
		{
			name:          "Valid path but not a block device",
			reqDevicePath: "/tmp",
			yes:           false,
			respError:     nil,
		},
	}

	statUtils := &VolumeStatUtils{}
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		response, err := statUtils.IsBlockDevice(tc.reqDevicePath)
		assert.Equal(t, tc.yes, response)
		if tc.respError != nil {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestIsDevicePathNotExist(t *testing.T) {
	testCases := []struct {
		name          string
		reqDevicePath string
		expResp       bool
	}{
		{
			name:          "Success device path not exists",
			reqDevicePath: "/tmp111111111111111",
			expResp:       true,
		},
		{
			name:          "Device path exists",
			reqDevicePath: "/tmp",
			expResp:       false,
		},
	}

	statUtils := &VolumeStatUtils{}
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		isBlock := statUtils.IsDevicePathNotExist(tc.reqDevicePath)
		assert.Equal(t, tc.expResp, isBlock)
	}
}

func TestDeviceInfo(t *testing.T) {
	testCases := []struct {
		name          string
		reqDevicePath string
		respError     error
	}{
		{
			name:          "Success device info",
			reqDevicePath: "/tmp",
			respError:     nil,
		},
		{
			name:          "Failed device info",
			reqDevicePath: "/tmp11111111111",
			respError:     fmt.Errorf("any error is fine"),
		},
	}

	statUtils := &VolumeStatUtils{}
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		_, _ = statUtils.DeviceInfo(tc.reqDevicePath)
		/*if tc.respError != nil {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}*/
	}
}

func makeFakeCmd(fakeCmd *testingexec.FakeCmd, cmd string, args ...string) testingexec.FakeCommandAction {
	c := cmd
	a := args
	return func(cmd string, args ...string) exec.Cmd {
		command := testingexec.InitFakeCmd(fakeCmd, c, a...)
		return command
	}
}

func TestCollectMountOptions(t *testing.T) {
	testCases := []struct {
		name           string
		fsType         string
		mntFlags       []string
		expectedResult []string
	}{
		{
			name:           "XFS filesystem with nouuid",
			fsType:         "xfs",
			mntFlags:       []string{"rw", "relatime"},
			expectedResult: []string{"rw", "relatime", "nouuid"},
		},
		{
			name:           "XFS filesystem with empty flags",
			fsType:         "xfs",
			mntFlags:       []string{},
			expectedResult: []string{"nouuid"},
		},
		{
			name:           "EXT4 filesystem without nouuid",
			fsType:         "ext4",
			mntFlags:       []string{"rw", "relatime"},
			expectedResult: []string{"rw", "relatime"},
		},
		{
			name:           "EXT4 filesystem with empty flags",
			fsType:         "ext4",
			mntFlags:       []string{},
			expectedResult: nil,
		},
		{
			name:           "EXT3 filesystem",
			fsType:         "ext3",
			mntFlags:       []string{"rw"},
			expectedResult: []string{"rw"},
		},
		{
			name:           "EXT2 filesystem",
			fsType:         "ext2",
			mntFlags:       []string{"ro"},
			expectedResult: []string{"ro"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := collectMountOptions(tc.fsType, tc.mntFlags)
			assert.Equal(t, tc.expectedResult, result, "Mount options should match expected result")
		})
	}
}

func TestFSInfo(t *testing.T) {
	testCases := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "Valid path - current directory",
			path:        ".",
			expectError: false,
		},
		{
			name:        "Invalid path",
			path:        "/nonexistent/path/that/does/not/exist",
			expectError: true,
		},
	}

	statUtils := &VolumeStatUtils{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			available, capacity, usage, inodes, inodesFree, inodesUsed, err := statUtils.FSInfo(tc.path)

			if tc.expectError {
				assert.NotNil(t, err, "Expected error for invalid path")
			} else {
				assert.Nil(t, err, "Should not return error for valid path")
				assert.True(t, available >= 0, "Available space should be non-negative")
				assert.True(t, capacity >= 0, "Capacity should be non-negative")
				assert.True(t, usage >= 0, "Usage should be non-negative")
				assert.True(t, inodes >= 0, "Inodes should be non-negative")
				assert.True(t, inodesFree >= 0, "Free inodes should be non-negative")
				assert.True(t, inodesUsed >= 0, "Used inodes should be non-negative")
			}
		})
	}
}

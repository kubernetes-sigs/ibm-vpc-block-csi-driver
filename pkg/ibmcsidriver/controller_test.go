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
	"net/http"
	"reflect"
	"strconv"
	"testing"

	"github.com/google/uuid"

	"github.com/IBM/ibm-csi-common/pkg/utils"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	providerError "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"

	cloudProvider "github.com/IBM/ibm-csi-common/pkg/ibmcloudprovider"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider/fake"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// Define "normal" parameters
	stdVolCap = []*csi.VolumeCapability{
		{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{FsType: "ext2"},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
	}
	stdVolCapNotSupported = []*csi.VolumeCapability{
		{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{FsType: "ext2"},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
			},
		},
	}
	stdBlockVolCap = []*csi.VolumeCapability{
		{
			AccessType: &csi.VolumeCapability_Block{
				Block: &csi.VolumeCapability_BlockVolume{},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
	}
	stdCapRange = &csi.CapacityRange{
		RequiredBytes: 20 * 1024 * 1024 * 1024,
	}
	stdParams = map[string]string{
		//"type": "ext2",
		Profile: "general-purpose",
		Zone:    "myzone",
		Region:  "myregion",
	}
	stdTopology = []*csi.Topology{
		{
			Segments: map[string]string{utils.NodeZoneLabel: "myzone", utils.NodeRegionLabel: "myregion"},
		},
	}
)

func TestCreateVolumeArguments(t *testing.T) {
	cap := 20
	volName := "test-name"
	iopsStr := ""
	// test cases
	testCases := []struct {
		name              string
		req               *csi.CreateVolumeRequest
		expVol            *csi.Volume
		expErrCode        codes.Code
		libVolumeResponse *provider.Volume
		libVolumeError    error
	}{
		{
			name: "Success default",
			req: &csi.CreateVolumeRequest{
				Name:               volName,
				CapacityRange:      stdCapRange,
				VolumeCapabilities: stdVolCap,
				Parameters:         stdParams,
			},
			expVol: &csi.Volume{
				CapacityBytes:      20 * 1024 * 1024 * 1024, // In byte
				VolumeId:           "testVolumeId",
				VolumeContext:      map[string]string{utils.NodeRegionLabel: "myregion", utils.NodeZoneLabel: "myzone", VolumeIDLabel: "testVolumeId", Tag: "", VolumeCRNLabel: "", ClusterIDLabel: ""},
				AccessibleTopology: stdTopology,
			},
			libVolumeResponse: &provider.Volume{Capacity: &cap, Name: &volName, VolumeID: "testVolumeId", Iops: &iopsStr, Az: "myzone", Region: "myregion"},
			expErrCode:        codes.OK,
			libVolumeError:    nil,
		},
		{
			name: "Empty volume name",
			req: &csi.CreateVolumeRequest{
				Name:               "",
				CapacityRange:      stdCapRange,
				VolumeCapabilities: stdVolCap,
				Parameters:         stdParams,
			},
			expVol:            nil,
			libVolumeResponse: nil,
			expErrCode:        codes.InvalidArgument,
			libVolumeError:    nil,
		},
		{
			name: "Empty volume capabilities",
			req: &csi.CreateVolumeRequest{
				Name:               volName,
				CapacityRange:      stdCapRange,
				VolumeCapabilities: nil,
				Parameters:         stdParams,
			},
			expVol:            nil,
			libVolumeResponse: nil,
			expErrCode:        codes.InvalidArgument,
			libVolumeError:    nil,
		},
		{
			name: "Not supported volume Capabilities",
			req: &csi.CreateVolumeRequest{
				Name:               volName,
				CapacityRange:      stdCapRange,
				VolumeCapabilities: stdVolCapNotSupported,
				Parameters:         stdParams,
			},
			expVol:            nil,
			libVolumeResponse: nil,
			expErrCode:        codes.InvalidArgument,
			libVolumeError:    nil,
		},
		{
			name: "ProvisioningFailed lib error form create volume",
			req: &csi.CreateVolumeRequest{
				Name:               volName,
				CapacityRange:      stdCapRange,
				VolumeCapabilities: stdVolCap,
				Parameters:         stdParams,
			},
			expErrCode:        codes.Internal,
			expVol:            nil,
			libVolumeResponse: nil,
			libVolumeError:    providerError.Message{Code: "FailedToPlaceOrder", Description: "Volume creation failed", Type: providerError.ProvisioningFailed},
		},
		{
			name: "InvalidRequest lib error form create volume",
			req: &csi.CreateVolumeRequest{
				Name:               volName,
				CapacityRange:      stdCapRange,
				VolumeCapabilities: stdVolCap,
				Parameters:         stdParams,
			},
			expErrCode:        codes.Internal,
			expVol:            nil,
			libVolumeResponse: nil,
			libVolumeError:    providerError.Message{Code: "FailedToPlaceOrder", Description: "Volume creation failed", Type: providerError.InvalidRequest},
		},
		{
			name: "Other error lib error form create volume",
			req: &csi.CreateVolumeRequest{
				Name:               volName,
				CapacityRange:      stdCapRange,
				VolumeCapabilities: stdVolCap,
				Parameters:         stdParams,
			},
			expErrCode:        codes.Internal,
			expVol:            nil,
			libVolumeResponse: nil,
			libVolumeError:    providerError.Message{Code: "FailedToPlaceOrder", Description: "Volume creation failed", Type: providerError.Unauthenticated},
		},
		{
			name: "Zone provided but region not provided as parameter",
			req: &csi.CreateVolumeRequest{
				Name:               volName,
				CapacityRange:      stdCapRange,
				VolumeCapabilities: stdVolCap,
				Parameters: map[string]string{
					//"type": "ext2",
					Profile: "general-purpose",
					Zone:    "myzone",
				},
				AccessibilityRequirements: &csi.TopologyRequirement{Preferred: []*csi.Topology{{Segments: map[string]string{
					utils.NodeRegionLabel: "myregion",
					utils.NodeZoneLabel:   "myzone",
				},
				},
				},
				},
			},
			expVol: &csi.Volume{
				CapacityBytes:      20 * 1024 * 1024 * 1024, // In byte
				VolumeId:           "testVolumeId",
				VolumeContext:      map[string]string{utils.NodeRegionLabel: "myregion", utils.NodeZoneLabel: "myzone", VolumeIDLabel: "testVolumeId", Tag: "", VolumeCRNLabel: "", ClusterIDLabel: ""},
				AccessibleTopology: stdTopology,
			},
			libVolumeResponse: &provider.Volume{Capacity: &cap, Name: &volName, VolumeID: "testVolumeId", Iops: &iopsStr, Az: "myzone", Region: "myregion"},
			expErrCode:        codes.OK,
			libVolumeError:    nil,
		},

		{
			name: "Zone and region not provided as parameter",
			req: &csi.CreateVolumeRequest{
				Name:               volName,
				CapacityRange:      stdCapRange,
				VolumeCapabilities: stdVolCap,
				Parameters: map[string]string{
					//"type": "ext2",
					Profile: "general-purpose",
				},
				AccessibilityRequirements: &csi.TopologyRequirement{Preferred: []*csi.Topology{{Segments: map[string]string{
					utils.NodeRegionLabel: "myregion",
					utils.NodeZoneLabel:   "myzone",
				},
				},
				},
				},
			},
			expVol: &csi.Volume{
				CapacityBytes:      20 * 1024 * 1024 * 1024, // In byte
				VolumeId:           "testVolumeId",
				VolumeContext:      map[string]string{utils.NodeRegionLabel: "myregion", utils.NodeZoneLabel: "myzone", VolumeIDLabel: "testVolumeId", Tag: "", VolumeCRNLabel: "", ClusterIDLabel: ""},
				AccessibleTopology: stdTopology,
			},
			libVolumeResponse: &provider.Volume{Capacity: &cap, Name: &volName, VolumeID: "testVolumeId", Iops: &iopsStr, Az: "myzone", Region: "myregion"},
			expErrCode:        codes.OK,
			libVolumeError:    nil,
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Run test cases
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		// Setup new driver each time so no interference
		icDriver := initIBMCSIDriver(t)

		// Set the response for CreateVolume
		fakeSession, err := icDriver.cs.CSIProvider.GetProviderSession(context.Background(), logger)
		assert.Nil(t, err)
		fakeStructSession, ok := fakeSession.(*fake.FakeSession)
		assert.Equal(t, true, ok)
		fakeStructSession.CreateVolumeReturns(tc.libVolumeResponse, tc.libVolumeError)
		fakeStructSession.GetVolumeByNameReturns(tc.libVolumeResponse, tc.libVolumeError)
		fakeStructSession.GetVolumeReturns(tc.libVolumeResponse, tc.libVolumeError)

		// Call CSI CreateVolume
		resp, err := icDriver.cs.CreateVolume(context.Background(), tc.req)
		if err != nil {
			//errorType := providerError.GetErrorType(err)
			serverError, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Could not get error status code from err: %v", serverError)
			}
			if serverError.Code() != tc.expErrCode {
				t.Fatalf("Expected error code-> %v, Actual error code: %v. err : %v", tc.expErrCode, serverError.Code(), err)
			}
			continue
		}
		if tc.expErrCode != codes.OK {
			t.Fatalf("Expected error-> %v, actual no error", tc.expErrCode)
		}

		// Make sure responses match
		vol := resp.GetVolume()
		if vol == nil {
			t.Fatalf("Expected volume-> %v, Actual volume is nil", tc.expVol)
		}

		// Validate output
		if !reflect.DeepEqual(vol, tc.expVol) {
			errStr := fmt.Sprintf("Expected volume-> %#v\nTopology %#v\n\n Actual volume: %#v\nTopology %#v\n\n",
				tc.expVol, tc.expVol.GetAccessibleTopology()[0], vol, vol.GetAccessibleTopology()[0])
			for i := 0; i < len(vol.GetAccessibleTopology()); i++ {
				errStr = errStr + fmt.Sprintf("Actual topology-> %#v\nExpected toplogy-> %#v\n\n", vol.GetAccessibleTopology()[i], tc.expVol.GetAccessibleTopology()[i])
			}
			t.Errorf(errStr)
		}
	}
}

func TestDeleteVolume(t *testing.T) {
	// test cases
	testCases := []struct {
		name               string
		req                *csi.DeleteVolumeRequest
		expResponse        *csi.DeleteVolumeResponse
		expErrCode         codes.Code
		libVolumeRespError error
		libVolumeResponse  *provider.Volume
	}{
		{
			name:              "Success volume delete",
			req:               &csi.DeleteVolumeRequest{VolumeId: "testVolumeId"},
			expResponse:       &csi.DeleteVolumeResponse{},
			expErrCode:        codes.OK,
			libVolumeResponse: &provider.Volume{VolumeID: "testVolumeId", Az: "myzone", Region: "myregion"},
		},
		{
			name:        "Success volume delete in case volume not found",
			req:         &csi.DeleteVolumeRequest{VolumeId: "testVolumeId"},
			expResponse: &csi.DeleteVolumeResponse{},
			expErrCode:  codes.OK,
		},
		{
			name:        "Failed volume delete with volume id empty",
			req:         &csi.DeleteVolumeRequest{VolumeId: ""},
			expResponse: nil,
			expErrCode:  codes.InvalidArgument,
		},
		{
			name:               "Failed from lib volume delete failed",
			req:                &csi.DeleteVolumeRequest{VolumeId: "testVolumeId"},
			expResponse:        nil,
			expErrCode:         codes.Internal,
			libVolumeRespError: providerError.Message{Code: "FailedToDeleteVolume", Description: "Volume deletion failed", Type: providerError.DeletionFailed},
			libVolumeResponse:  &provider.Volume{VolumeID: "testVolumeId", Az: "myzone", Region: "myregion"},
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Run test cases
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		// Setup new driver each time so no interference
		icDriver := initIBMCSIDriver(t)

		// Set the response for DeleteVolume
		fakeSession, err := icDriver.cs.CSIProvider.GetProviderSession(context.Background(), logger)
		assert.Nil(t, err)
		fakeStructSession, ok := fakeSession.(*fake.FakeSession)
		assert.Equal(t, true, ok)
		fakeStructSession.DeleteVolumeReturns(tc.libVolumeRespError)
		fakeStructSession.GetVolumeByNameReturns(tc.libVolumeResponse, nil)
		fakeStructSession.GetVolumeReturns(tc.libVolumeResponse, nil)

		// Call CSI CreateVolume
		response, err := icDriver.cs.DeleteVolume(context.Background(), tc.req)
		if tc.expErrCode != codes.OK {
			assert.NotNil(t, err)
		}
		assert.Equal(t, tc.expResponse, response)
	}
}

func isPublishVolumeresponseEqual(expected *csi.ControllerPublishVolumeResponse, actual *csi.ControllerPublishVolumeResponse) bool {
	if expected == nil && actual == nil {
		return true
	}

	if expected == nil || actual == nil {
		return false
	}

	return expected.PublishContext["volume-id"] == actual.PublishContext["volume-id"] &&
		expected.PublishContext["node-id"] == actual.PublishContext["node-id"] &&
		expected.PublishContext["device-path"] == actual.PublishContext["device-path"]
}

func TestControllerPublishVolume(t *testing.T) {
	// test cases
	testCases := []struct {
		name                   string
		req                    *csi.ControllerPublishVolumeRequest
		expResponse            *csi.ControllerPublishVolumeResponse
		expErrCode             codes.Code
		libAttachResponse      *provider.VolumeAttachmentResponse
		libAttachRespError     error
		libWaitAttachResponse  *provider.VolumeAttachmentResponse
		libWaitAttachRespError error
		libVolumeResponse      *provider.Volume
		libVolumeRespError     error
	}{
		{
			name:                   "Success attachment",
			req:                    &csi.ControllerPublishVolumeRequest{VolumeId: "vol123", NodeId: "node123", VolumeCapability: &csi.VolumeCapability{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
			expResponse:            &csi.ControllerPublishVolumeResponse{PublishContext: map[string]string{"attach-status": "", "device-path": "/tmp", "node-id": "node123", "volume-id": "vol123"}},
			expErrCode:             codes.OK,
			libAttachResponse:      &provider.VolumeAttachmentResponse{VolumeAttachmentRequest: provider.VolumeAttachmentRequest{VolumeID: "vol123", InstanceID: "node123", VPCVolumeAttachment: &provider.VolumeAttachment{DevicePath: "/tmp"}}},
			libAttachRespError:     nil,
			libWaitAttachResponse:  &provider.VolumeAttachmentResponse{VolumeAttachmentRequest: provider.VolumeAttachmentRequest{VolumeID: "vol123", InstanceID: "node123", VPCVolumeAttachment: &provider.VolumeAttachment{DevicePath: "/tmp"}}},
			libWaitAttachRespError: nil,
			libVolumeResponse:      &provider.Volume{VolumeID: "vol123"},
			libVolumeRespError:     nil,
		},
		{
			name:               "Failed AttachVolume library call for node not found",
			req:                &csi.ControllerPublishVolumeRequest{VolumeId: "vol123", NodeId: "node123", VolumeCapability: &csi.VolumeCapability{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
			expResponse:        nil,
			expErrCode:         codes.NotFound,
			libAttachResponse:  nil,
			libAttachRespError: providerError.Message{Code: "AttachFailed", Description: "Volume attach failed", Type: providerError.NodeNotFound},
			libVolumeResponse:  &provider.Volume{VolumeID: "vol123"},
			libVolumeRespError: nil,
		},
		{
			name:               "Failed AttachVolume library call AttachVolume failed with internal error",
			req:                &csi.ControllerPublishVolumeRequest{VolumeId: "vol123", NodeId: "node123", VolumeCapability: &csi.VolumeCapability{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
			expResponse:        nil,
			expErrCode:         codes.Internal,
			libAttachResponse:  nil,
			libAttachRespError: providerError.Message{Code: "AttachFailed", Description: "Volume attach failed", Type: providerError.PermissionDenied}, // any error apart from NodeNotFound
			libVolumeResponse:  &provider.Volume{VolumeID: "vol123"},
			libVolumeRespError: nil,
		},
		{
			name:               "Failed AttachVolume library call for volume not found",
			req:                &csi.ControllerPublishVolumeRequest{VolumeId: "vol123", NodeId: "node123", VolumeCapability: &csi.VolumeCapability{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
			expResponse:        nil,
			expErrCode:         codes.NotFound,
			libVolumeResponse:  nil,
			libVolumeRespError: providerError.Message{Code: "EntityNotFound", Description: "Volume not found", Type: providerError.EntityNotFound},
		},
		{
			name:               "Failed AttachVolume library call internal error for get volume call",
			req:                &csi.ControllerPublishVolumeRequest{VolumeId: "vol123", NodeId: "node123", VolumeCapability: &csi.VolumeCapability{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
			expResponse:        nil,
			expErrCode:         codes.Internal,
			libVolumeResponse:  nil,
			libVolumeRespError: providerError.Message{Description: "internal error", Type: providerError.PermissionDenied}, // any error apart from not found
		},
		{
			name:        "Failed volume id empty",
			req:         &csi.ControllerPublishVolumeRequest{VolumeId: "", NodeId: "node123", VolumeCapability: &csi.VolumeCapability{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
			expResponse: nil,
			expErrCode:  codes.InvalidArgument,
		},
		{
			name:        "Failed node id empty",
			req:         &csi.ControllerPublishVolumeRequest{VolumeId: "vol123", NodeId: "", VolumeCapability: &csi.VolumeCapability{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
			expResponse: nil,
			expErrCode:  codes.InvalidArgument,
		},
		{
			name:        "Failed nil volume capabilities",
			req:         &csi.ControllerPublishVolumeRequest{VolumeId: "vol123", NodeId: "node123"},
			expResponse: nil,
			expErrCode:  codes.InvalidArgument,
		},
		{
			name:        "Failed unsupported volume capability",
			req:         &csi.ControllerPublishVolumeRequest{VolumeId: "vol123", NodeId: "node123", VolumeCapability: &csi.VolumeCapability{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER}}},
			expResponse: nil,
			expErrCode:  codes.InvalidArgument,
		},
		{
			name:                   "Failed while waiting for attachment",
			req:                    &csi.ControllerPublishVolumeRequest{VolumeId: "vol123", NodeId: "node123", VolumeCapability: &csi.VolumeCapability{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
			expResponse:            nil,
			expErrCode:             codes.Internal,
			libAttachResponse:      &provider.VolumeAttachmentResponse{VolumeAttachmentRequest: provider.VolumeAttachmentRequest{VolumeID: "vol123", InstanceID: "node123", VPCVolumeAttachment: &provider.VolumeAttachment{DevicePath: "/tmp"}}},
			libWaitAttachRespError: providerError.Message{Description: "internal error while waiting for attachment"}, // any error code is fine
			libVolumeResponse:      &provider.Volume{VolumeID: "vol123"},
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Run test cases
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		// Setup new driver each time so no interference
		icDriver := initIBMCSIDriver(t)

		// Set the response for CreateVolume
		fakeSession, err := icDriver.cs.CSIProvider.GetProviderSession(context.Background(), logger)
		assert.Nil(t, err)
		fakeStructSession, ok := fakeSession.(*fake.FakeSession)
		assert.Equal(t, true, ok)
		fakeStructSession.AttachVolumeReturns(tc.libAttachResponse, tc.libAttachRespError)
		fakeStructSession.WaitForAttachVolumeReturns(tc.libWaitAttachResponse, tc.libWaitAttachRespError)
		fakeStructSession.GetVolumeByNameReturns(tc.libVolumeResponse, tc.libVolumeRespError)
		fakeStructSession.GetVolumeReturns(tc.libVolumeResponse, tc.libVolumeRespError)

		// Call CSI CreateVolume
		response, err := icDriver.cs.ControllerPublishVolume(context.Background(), tc.req)
		if tc.expErrCode != codes.OK {
			assert.NotNil(t, err)
		}
		// This is because csi.ControllerPublishVolumeResponse contains request ID which is always different
		// hence better to compair all fields
		assert.Equal(t, true, isPublishVolumeresponseEqual(tc.expResponse, response))
	}
}

func TestControllerUnpublishVolume(t *testing.T) {
	// test cases
	testCases := []struct {
		name                  string
		req                   *csi.ControllerUnpublishVolumeRequest
		expResponse           *csi.ControllerUnpublishVolumeResponse
		expErrCode            codes.Code
		libDetachResponse     *http.Response
		libDetachResponseErr  error
		libWaitDetachResponse error
	}{
		{
			name:                  "Success detach volume",
			req:                   &csi.ControllerUnpublishVolumeRequest{VolumeId: "volumeid", NodeId: "nodeid"},
			expResponse:           &csi.ControllerUnpublishVolumeResponse{},
			expErrCode:            codes.OK,
			libDetachResponse:     &http.Response{StatusCode: http.StatusOK},
			libDetachResponseErr:  nil,
			libWaitDetachResponse: nil,
		},
		{
			name:                  "Nil volume ID",
			req:                   &csi.ControllerUnpublishVolumeRequest{VolumeId: "", NodeId: "nodeid"},
			expResponse:           nil,
			expErrCode:            codes.InvalidArgument,
			libDetachResponse:     nil,
			libDetachResponseErr:  nil,
			libWaitDetachResponse: nil,
		},
		{
			name:                  "Nil node ID",
			req:                   &csi.ControllerUnpublishVolumeRequest{VolumeId: "volumeid", NodeId: ""},
			expResponse:           nil,
			expErrCode:            codes.InvalidArgument,
			libDetachResponse:     nil,
			libDetachResponseErr:  nil,
			libWaitDetachResponse: nil,
		},
		{
			name:              "Detach volume failed",
			req:               &csi.ControllerUnpublishVolumeRequest{VolumeId: "volumeid", NodeId: "nodeid"},
			expResponse:       nil,
			expErrCode:        codes.Internal,
			libDetachResponse: nil,
			libDetachResponseErr: providerError.Message{
				Description: "Volume detach failed",
				Type:        providerError.DetachFailed,
			},
			libWaitDetachResponse: nil,
		},
		{
			name:                 "Wait for detach volume failed",
			req:                  &csi.ControllerUnpublishVolumeRequest{VolumeId: "volumeid", NodeId: "nodeid"},
			expResponse:          nil,
			expErrCode:           codes.Internal,
			libDetachResponse:    nil,
			libDetachResponseErr: nil,
			libWaitDetachResponse: providerError.Message{
				Description: "Volume detach status failed",
				Type:        providerError.RetrivalFailed, // any error is fine as driver is checking error only
			},
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Run test cases
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		// Setup new driver each time so no interference
		icDriver := initIBMCSIDriver(t)

		// Set the response for CreateVolume
		fakeSession, err := icDriver.cs.CSIProvider.GetProviderSession(context.Background(), logger)
		assert.Nil(t, err)
		fakeStructSession, ok := fakeSession.(*fake.FakeSession)
		assert.Equal(t, true, ok)
		fakeStructSession.DetachVolumeReturns(tc.libDetachResponse, tc.libDetachResponseErr)
		fakeStructSession.WaitForDetachVolumeReturns(tc.libWaitDetachResponse)

		// Call CSI CreateVolume
		response, err := icDriver.cs.ControllerUnpublishVolume(context.Background(), tc.req)
		if tc.expErrCode != codes.OK {
			assert.NotNil(t, err)
		}
		// This is because csi.ControllerPublishVolumeResponse contains request ID which is always different
		// hence better to compair all fields
		assert.Equal(t, tc.expResponse, response)
	}
}

func TestValidateVolumeCapabilities(t *testing.T) {
	// test cases
	testCases := []struct {
		name              string
		req               *csi.ValidateVolumeCapabilitiesRequest
		expResponse       *csi.ValidateVolumeCapabilitiesResponse
		expErrCode        codes.Code
		libGetVolumeError error
	}{
		{
			name: "Success validate volume capabilities",
			req: &csi.ValidateVolumeCapabilitiesRequest{VolumeId: "volumeid",
				VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
			},
			expResponse: &csi.ValidateVolumeCapabilitiesResponse{
				Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
					VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
				},
			},
			expErrCode:        codes.OK,
			libGetVolumeError: nil,
		},
		{
			name: "Passing nil volume capabilities",
			req: &csi.ValidateVolumeCapabilitiesRequest{VolumeId: "volumeid",
				VolumeCapabilities: nil,
			},
			expResponse:       nil,
			expErrCode:        codes.InvalidArgument,
			libGetVolumeError: nil,
		},
		{
			name: "Passing nil volume ID",
			req: &csi.ValidateVolumeCapabilitiesRequest{VolumeId: "",
				VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
			},
			expResponse:       nil,
			expErrCode:        codes.InvalidArgument,
			libGetVolumeError: nil,
		},
		{
			name: "Get volume failed",
			req: &csi.ValidateVolumeCapabilitiesRequest{VolumeId: "volume-not-found-ID",
				VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
			},
			expResponse: nil,
			expErrCode:  codes.NotFound,
			libGetVolumeError: providerError.Message{
				Code:        "StorageFindFailedWithVolumeName",
				Description: "Volume not found by volume ID",
				Type:        providerError.RetrivalFailed,
			},
		},
		{
			name: "Internal error while getting volume details",
			req: &csi.ValidateVolumeCapabilitiesRequest{VolumeId: "volumeid",
				VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}}},
			},
			expResponse: nil,
			expErrCode:  codes.Internal,
			libGetVolumeError: providerError.Message{
				Code:        "StorageFindFailed",
				Description: "Internal error",
				Type:        providerError.PermissionDenied, // any error apartfrom providerError.RetrivalFailed
			},
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Run test cases
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		// Setup new driver each time so no interference
		icDriver := initIBMCSIDriver(t)

		// Set the response for GetVolume
		fakeSession, err := icDriver.cs.CSIProvider.GetProviderSession(context.Background(), logger)
		assert.Nil(t, err)
		fakeStructSession, ok := fakeSession.(*fake.FakeSession)
		assert.Equal(t, true, ok)
		fakeStructSession.GetVolumeReturns(nil, tc.libGetVolumeError)

		// Call CSI CreateVolume
		response, err := icDriver.cs.ValidateVolumeCapabilities(context.Background(), tc.req)
		if tc.expErrCode != codes.OK {
			t.Logf("Error code")
			assert.NotNil(t, err)
		}
		// This is because csi.ControllerPublishVolumeResponse contains request ID which is always different
		// hence better to compair all fields
		assert.Equal(t, tc.expResponse, response)
	}
}

func TestListVolumes(t *testing.T) {
	limit := 100
	testCases := []struct {
		name            string
		maxEntries      int32
		expectedEntries int
		expectedErr     bool
		expErrCode      codes.Code
		libVolumeError  error
	}{
		{
			name:            "normal",
			expectedEntries: 50,
			expectedErr:     false,
			expErrCode:      codes.OK,
			libVolumeError:  nil,
		},
		{
			name:            "fine amount of entries",
			maxEntries:      40,
			expectedEntries: 40,
			expectedErr:     false,
			expErrCode:      codes.OK,
			libVolumeError:  nil,
		},
		{
			name:            "too many entries, but defaults to 100",
			maxEntries:      101,
			expectedEntries: 100,
			expectedErr:     false,
			expErrCode:      codes.OK,
			libVolumeError:  nil,
		},
		{
			name:           "negative entries",
			maxEntries:     -1,
			expectedErr:    true,
			expErrCode:     codes.InvalidArgument,
			libVolumeError: providerError.Message{Code: "InvalidListVolumesLimit", Description: "The value '-1' specified in the limit parameter of the list volume call is not valid.", Type: providerError.InvalidRequest},
		},
		{
			name:           "Invalid start volume ID",
			maxEntries:     10,
			expectedErr:    true,
			expErrCode:     codes.Aborted,
			libVolumeError: providerError.Message{Code: "StartVolumeIDNotFound", Description: "The volume ID specified in the start parameter of the list volume call could not be found.", Type: providerError.InvalidRequest},
		},
		{
			name:           "internal error",
			maxEntries:     10,
			expectedErr:    true,
			expErrCode:     codes.Internal,
			libVolumeError: providerError.Message{Code: "ListVolumesFailed", Description: "Unable to fetch list of volumes.", Type: providerError.RetrivalFailed},
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		// Setup new driver each time so no interference
		icDriver := initIBMCSIDriver(t)

		// Set the response for CreateVolume
		fakeSession, err := icDriver.cs.CSIProvider.GetProviderSession(context.Background(), logger)
		assert.Nil(t, err)
		fakeStructSession, ok := fakeSession.(*fake.FakeSession)
		assert.Equal(t, true, ok)

		maxEntries := int(tc.maxEntries)
		if maxEntries == 0 {
			maxEntries = 50
		} else if maxEntries > limit {
			maxEntries = limit
		}

		volList := &provider.VolumeList{}
		if !tc.expectedErr {
			volList = createVolume(maxEntries)
		}
		fakeStructSession.ListVolumesReturns(volList, tc.libVolumeError)

		lvr := &csi.ListVolumesRequest{
			MaxEntries: tc.maxEntries,
		}
		resp, err := icDriver.cs.ListVolumes(context.TODO(), lvr)
		if tc.expErrCode != codes.OK {
			assert.NotNil(t, err)
		}
		if tc.expectedErr && err == nil {
			t.Fatalf("Got no error when expecting an error")
		}
		if err != nil {
			if !tc.expectedErr {
				t.Fatalf("Got error '%v', expecting none", err)
			}
		} else {
			if len(resp.Entries) != tc.expectedEntries {
				t.Fatalf("Got '%v' entries, expected '%v'", len(resp.Entries), tc.expectedEntries)
			}
			if resp.NextToken != volList.Next {
				t.Fatalf("Got '%v' next_token, expected '%v'", resp.NextToken, volList.Next)
			}
		}
	}
}

func TestGetCapacity(t *testing.T) {
	// test cases
	testCases := []struct {
		name        string
		req         *csi.GetCapacityRequest
		expResponse *csi.GetCapacityResponse
		expErrCode  codes.Code
	}{
		{
			name:        "Success get capacity",
			req:         &csi.GetCapacityRequest{},
			expResponse: nil,
			expErrCode:  codes.OK,
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Run test cases
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		// Setup new driver each time so no interference
		icDriver := initIBMCSIDriver(t)

		fakeSession, err := icDriver.cs.CSIProvider.GetProviderSession(context.Background(), logger)
		assert.Nil(t, err)
		/*fakeStructSession*/ _, ok := fakeSession.(*fake.FakeSession)
		assert.Equal(t, true, ok)

		// Call CSI CreateVolume
		response, err := icDriver.cs.GetCapacity(context.Background(), tc.req)
		if tc.expErrCode != codes.OK {
			t.Logf("Error code")
			assert.NotNil(t, err)
		}
		assert.Equal(t, tc.expResponse, response)
	}
}

func TestControllerGetCapabilities(t *testing.T) {
	// test cases
	testCases := []struct {
		name        string
		req         *csi.ControllerGetCapabilitiesRequest
		expResponse *csi.ControllerGetCapabilitiesResponse
		expErrCode  codes.Code
	}{
		{
			name: "Success controller get capabilities",
			req:  &csi.ControllerGetCapabilitiesRequest{},
			expResponse: &csi.ControllerGetCapabilitiesResponse{
				Capabilities: []*csi.ControllerServiceCapability{
					{Type: &csi.ControllerServiceCapability_Rpc{Rpc: &csi.ControllerServiceCapability_RPC{Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME}}},
					{Type: &csi.ControllerServiceCapability_Rpc{Rpc: &csi.ControllerServiceCapability_RPC{Type: csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME}}},
					{Type: &csi.ControllerServiceCapability_Rpc{Rpc: &csi.ControllerServiceCapability_RPC{Type: csi.ControllerServiceCapability_RPC_LIST_VOLUMES}}},
					{Type: &csi.ControllerServiceCapability_Rpc{Rpc: &csi.ControllerServiceCapability_RPC{Type: csi.ControllerServiceCapability_RPC_EXPAND_VOLUME}}},
					// &csi.ControllerServiceCapability{Type: &csi.ControllerServiceCapability_Rpc{Rpc: &csi.ControllerServiceCapability_RPC{Type: csi.ControllerServiceCapability_RPC_GET_CAPACITY}}},
					// &csi.ControllerServiceCapability{Type: &csi.ControllerServiceCapability_Rpc{Rpc: &csi.ControllerServiceCapability_RPC{Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT}}},
					// &csi.ControllerServiceCapability{Type: &csi.ControllerServiceCapability_Rpc{Rpc: &csi.ControllerServiceCapability_RPC{Type: csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS}}},
					// &csi.ControllerServiceCapability{Type: &csi.ControllerServiceCapability_Rpc{Rpc: &csi.ControllerServiceCapability_RPC{Type: csi.ControllerServiceCapability_RPC_PUBLISH_READONLY}}},
				},
			},
			expErrCode: codes.OK,
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Run test cases
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		// Setup new driver each time so no interference
		icDriver := initIBMCSIDriver(t)

		fakeSession, err := icDriver.cs.CSIProvider.GetProviderSession(context.Background(), logger)
		assert.Nil(t, err)
		/*fakeStructSession*/ _, ok := fakeSession.(*fake.FakeSession)
		assert.Equal(t, true, ok)

		// Call CSI CreateVolume
		response, err := icDriver.cs.ControllerGetCapabilities(context.Background(), tc.req)
		if tc.expErrCode != codes.OK {
			t.Logf("Error code")
			assert.NotNil(t, err)
		}

		if !reflect.DeepEqual(response, tc.expResponse) {
			assert.Equal(t, tc.expResponse, response)
		}
	}
}

func TestCreateSnapshot(t *testing.T) {
	// test cases
	testCases := []struct {
		name        string
		req         *csi.CreateSnapshotRequest
		expResponse *csi.CreateSnapshotResponse
		expErrCode  codes.Code
	}{
		{
			name:        "Success create snapshot",
			req:         &csi.CreateSnapshotRequest{},
			expResponse: nil,
			expErrCode:  codes.OK,
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Run test cases
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		// Setup new driver each time so no interference
		icDriver := initIBMCSIDriver(t)

		fakeSession, err := icDriver.cs.CSIProvider.GetProviderSession(context.Background(), logger)
		assert.Nil(t, err)
		/*fakeStructSession*/ _, ok := fakeSession.(*fake.FakeSession)
		assert.Equal(t, true, ok)

		// Call CSI CreateVolume
		response, err := icDriver.cs.CreateSnapshot(context.Background(), tc.req)
		if tc.expErrCode != codes.OK {
			t.Logf("Error code")
			assert.NotNil(t, err)
		}
		assert.Equal(t, tc.expResponse, response)
	}
}

func TestDeleteSnapshot(t *testing.T) {
	// test cases
	testCases := []struct {
		name        string
		req         *csi.DeleteSnapshotRequest
		expResponse *csi.DeleteSnapshotResponse
		expErrCode  codes.Code
	}{
		{
			name:        "Success delete snapshot",
			req:         &csi.DeleteSnapshotRequest{},
			expResponse: nil,
			expErrCode:  codes.OK,
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Run test cases
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		// Setup new driver each time so no interference
		icDriver := initIBMCSIDriver(t)

		fakeSession, err := icDriver.cs.CSIProvider.GetProviderSession(context.Background(), logger)
		assert.Nil(t, err)
		/*fakeStructSession*/ _, ok := fakeSession.(*fake.FakeSession)
		assert.Equal(t, true, ok)

		// Call CSI CreateVolume
		response, err := icDriver.cs.DeleteSnapshot(context.Background(), tc.req)
		if tc.expErrCode != codes.OK {
			t.Logf("Error code")
			assert.NotNil(t, err)
		}
		assert.Equal(t, tc.expResponse, response)
	}
}

func TestListSnapshots(t *testing.T) {
	// test cases
	testCases := []struct {
		name        string
		req         *csi.ListSnapshotsRequest
		expResponse *csi.ListSnapshotsResponse
		expErrCode  codes.Code
	}{
		{
			name:        "Success list snapshots",
			req:         &csi.ListSnapshotsRequest{},
			expResponse: nil,
			expErrCode:  codes.OK,
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Run test cases
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		// Setup new driver each time so no interference
		icDriver := initIBMCSIDriver(t)

		fakeSession, err := icDriver.cs.CSIProvider.GetProviderSession(context.Background(), logger)
		assert.Nil(t, err)
		/*fakeStructSession*/ _, ok := fakeSession.(*fake.FakeSession)
		assert.Equal(t, true, ok)

		// Call CSI CreateVolume
		response, err := icDriver.cs.ListSnapshots(context.Background(), tc.req)
		if tc.expErrCode != codes.OK {
			t.Logf("Error code")
			assert.NotNil(t, err)
		}
		assert.Equal(t, tc.expResponse, response)
	}
}

func TestGetSnapshots(t *testing.T) {
	// test cases
	testCases := []struct {
		name        string
		req         *csi.ListSnapshotsRequest
		expResponse *csi.ListSnapshotsResponse
		expErrCode  codes.Code
	}{
		{
			name:        "Success get snapshots",
			req:         &csi.ListSnapshotsRequest{},
			expResponse: nil,
			expErrCode:  codes.OK,
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Run test cases
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		// Setup new driver each time so no interference
		icDriver := initIBMCSIDriver(t)

		fakeSession, err := icDriver.cs.CSIProvider.GetProviderSession(context.Background(), logger)
		assert.Nil(t, err)
		/*fakeStructSession*/ _, ok := fakeSession.(*fake.FakeSession)
		assert.Equal(t, true, ok)

		// Call CSI CreateVolume
		response, err := icDriver.cs.getSnapshots(context.Background(), tc.req)
		if tc.expErrCode != codes.OK {
			t.Logf("Error code")
			assert.NotNil(t, err)
		}
		assert.Equal(t, tc.expResponse, response)
	}
}

func TestGetSnapshotByID(t *testing.T) {
	// test cases
	testCases := []struct {
		name        string
		req         string
		expResponse *csi.ListSnapshotsResponse
		expErrCode  codes.Code
	}{
		{
			name:        "Success get snapshotByID",
			req:         "snapshotID",
			expResponse: nil,
			expErrCode:  codes.OK,
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Run test cases
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		// Setup new driver each time so no interference
		icDriver := initIBMCSIDriver(t)

		fakeSession, err := icDriver.cs.CSIProvider.GetProviderSession(context.Background(), logger)
		assert.Nil(t, err)
		/*fakeStructSession*/ _, ok := fakeSession.(*fake.FakeSession)
		assert.Equal(t, true, ok)

		// Call CSI CreateVolume
		response, err := icDriver.cs.getSnapshotByID(context.Background(), tc.req)
		if tc.expErrCode != codes.OK {
			t.Logf("Error code")
			assert.NotNil(t, err)
		}
		assert.Equal(t, tc.expResponse, response)
	}
}

func TestControllerExpandVolume(t *testing.T) {
	cap := 20
	volName := "test-name"
	iopsStr := ""
	// test cases
	testCases := []struct {
		name                 string
		req                  *csi.ControllerExpandVolumeRequest
		expResponse          *csi.ControllerExpandVolumeResponse
		expErrCode           codes.Code
		libExpandResponse    *http.Response
		libVolumeResponse    *provider.Volume
		libExpandResponseErr error
		libVolumeError       error
	}{
		{
			name:                 "Success controller expand volume",
			req:                  &csi.ControllerExpandVolumeRequest{VolumeId: "volumeid", CapacityRange: stdCapRange},
			expResponse:          &csi.ControllerExpandVolumeResponse{CapacityBytes: stdCapRange.RequiredBytes, NodeExpansionRequired: true},
			expErrCode:           codes.OK,
			libExpandResponse:    &http.Response{StatusCode: http.StatusOK},
			libVolumeResponse:    &provider.Volume{Capacity: &cap, Name: &volName, VolumeID: "volumeid", Iops: &iopsStr, Az: "myzone", Region: "myregion"},
			libExpandResponseErr: nil,
			libVolumeError:       nil,
		},
		{
			name:                 "Nil capacity",
			req:                  &csi.ControllerExpandVolumeRequest{VolumeId: "volumeid", CapacityRange: nil},
			expResponse:          nil,
			expErrCode:           codes.InvalidArgument,
			libExpandResponse:    nil,
			libVolumeResponse:    nil,
			libExpandResponseErr: nil,
			libVolumeError:       nil,
		},
		{
			name:                 "Nil volume ID",
			req:                  &csi.ControllerExpandVolumeRequest{VolumeId: "", CapacityRange: stdCapRange},
			expResponse:          nil,
			expErrCode:           codes.InvalidArgument,
			libExpandResponse:    nil,
			libVolumeResponse:    nil,
			libExpandResponseErr: nil,
			libVolumeError:       nil,
		},
		{
			name:              "Expand volume failed",
			req:               &csi.ControllerExpandVolumeRequest{VolumeId: "volumeid", CapacityRange: stdCapRange},
			expResponse:       nil,
			expErrCode:        codes.Internal,
			libExpandResponse: nil,
			libVolumeResponse: &provider.Volume{Capacity: &cap, Name: &volName, VolumeID: "volumeid", Iops: &iopsStr, Az: "myzone", Region: "myregion"},
			libExpandResponseErr: providerError.Message{
				Code: "FailedToPlaceOrder",
			},
			libVolumeError: providerError.Message{Code: "FailedToPlaceOrder", Description: "Volume expansion failed", Type: providerError.Unauthenticated},
		},
	}

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	// Run test cases
	for _, tc := range testCases {
		t.Logf("test case: %s", tc.name)
		// Setup new driver each time so no interference
		icDriver := initIBMCSIDriver(t)

		// Set the response for CreateVolume
		fakeSession, err := icDriver.cs.CSIProvider.GetProviderSession(context.Background(), logger)
		assert.Nil(t, err)
		fakeStructSession, ok := fakeSession.(*fake.FakeSession)
		assert.Equal(t, true, ok)
		if tc.req.CapacityRange != nil {
			fakeStructSession.ExpandVolumeReturns(tc.req.CapacityRange.RequiredBytes, tc.libVolumeError)
		}
		fakeStructSession.GetVolumeByNameReturns(tc.libVolumeResponse, tc.libVolumeError)
		fakeStructSession.GetVolumeReturns(tc.libVolumeResponse, tc.libVolumeError)

		// Call CSI CreateVolume
		response, err := icDriver.cs.ControllerExpandVolume(context.Background(), tc.req)
		if tc.expErrCode != codes.OK {
			t.Logf("Error code")
			assert.NotNil(t, err)
		}
		assert.Equal(t, tc.expResponse, response)
	}
}

func createVolume(maxEntries int) *provider.VolumeList {
	volList := &provider.VolumeList{}
	cap := 10
	for i := 0; i <= maxEntries; i++ {
		volName := "unit-test-volume" + strconv.Itoa(i)
		vol := &provider.Volume{
			VolumeID: fmt.Sprintf("vol-uuid-test-vol-%s", uuid.New().String()[:10]),
			Name:     &volName,
			Region:   "my-region",
			Capacity: &cap,
		}
		if i == maxEntries {
			volList.Next = vol.VolumeID
		} else {
			volList.Volumes = append(volList.Volumes, vol)
		}
	}
	return volList
}

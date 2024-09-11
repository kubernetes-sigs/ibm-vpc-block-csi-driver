/*
Copyright 2024 The Kubernetes Authors.

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
	"os"
	"strings"
	"time"

	cloudProvider "github.com/IBM/ibm-csi-common/pkg/ibmcloudprovider"
	commonError "github.com/IBM/ibm-csi-common/pkg/messages"
	"github.com/IBM/ibm-csi-common/pkg/metrics"
	"github.com/IBM/ibm-csi-common/pkg/utils"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	providerError "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	utilReasonCode "github.com/IBM/ibmcloud-volume-interface/lib/utils/reasoncode"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	csi "github.com/container-storage-interface/spec/lib/go/csi"

	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// CSIControllerServer ...
type CSIControllerServer struct {
	Driver      *IBMCSIDriver
	CSIProvider cloudProvider.CloudProviderInterface
	mutex       utils.LockStore
}

const (
	// PublishInfoVolumeID ...
	PublishInfoVolumeID = "volume-id"

	// PublishInfoNodeID ...
	PublishInfoNodeID = "node-id"

	// PublishInfoStatus ...
	PublishInfoStatus = "attach-status"

	// PublishInfoDevicePath ...
	PublishInfoDevicePath = "device-path"

	// PublishInfoRequestID ...
	PublishInfoRequestID = "request-id"
)

var _ csi.ControllerServer = &CSIControllerServer{}

// CreateVolume ...
func (csiCS *CSIControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	// populate requestID in the context
	ctx = context.WithValue(ctx, provider.RequestID, requestID)
	ctxLogger.Info("CSIControllerServer-CreateVolume... ", zap.Reflect("Request", *req))
	defer metrics.UpdateDurationFromStart(ctxLogger, "CreateVolume", time.Now())

	// Check basic parameters validations i.e PVC name given
	name := req.GetName()
	if len(name) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.MissingVolumeName, requestID, nil)
	}

	// check volume capabilities
	volumeCapabilities := req.GetVolumeCapabilities()
	if len(volumeCapabilities) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.NoVolumeCapabilities, requestID, nil)
	}

	// Validate volume capabilities, are all capabilities supported by driver or not
	if !areVolumeCapabilitiesSupported(req.GetVolumeCapabilities(), csiCS.Driver.vcap) {
		return nil, commonError.GetCSIError(ctxLogger, commonError.VolumeCapabilitiesNotSupported, requestID, nil)
	}

	// Get volume input Parameters
	requestedVolume, err := getVolumeParameters(ctxLogger, req, csiCS.CSIProvider.GetConfig())
	if requestedVolume != nil {
		// For logging mask VolumeEncryptionKey
		// Create copy of the requestedVolume
		tempReqVol := (*requestedVolume)
		// Mask VolumeEncryptionKey
		tempReqVol.VPCVolume.VolumeEncryptionKey = &provider.VolumeEncryptionKey{CRN: "********"}
		ctxLogger.Info("Volume request after masking encryption key", zap.Reflect("Volume", tempReqVol))
	}

	if err != nil {
		ctxLogger.Error("Unable to extract parameters", zap.Error(err))
		return nil, commonError.GetCSIError(ctxLogger, commonError.InvalidParameters, requestID, err)
	}

	// TODO: Determine Zones and Region for the disk

	// Validate if volume Already Exists
	session, err := csiCS.CSIProvider.GetProviderSession(ctx, ctxLogger)
	if err != nil {
		if userError.GetUserErrorCode(err) == string(utilReasonCode.EndpointNotReachable) {
			return nil, commonError.GetCSIError(ctxLogger, commonError.EndpointNotReachable, requestID, err)
		}
		if userError.GetUserErrorCode(err) == string(utilReasonCode.Timeout) {
			return nil, commonError.GetCSIError(ctxLogger, commonError.Timeout, requestID, err)
		}
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}

	volumeSource := req.GetVolumeContentSource()
	if volumeSource != nil {
		if _, ok := volumeSource.GetType().(*csi.VolumeContentSource_Snapshot); !ok {
			return nil, commonError.GetCSIError(ctxLogger, commonError.UnsupportedVolumeContentSource, requestID, nil)
		}
		sourceSnapshot := volumeSource.GetSnapshot()
		if sourceSnapshot == nil {
			return nil, commonError.GetCSIError(ctxLogger, commonError.VolumeInvalidArguments, requestID, nil)
		}
		snapshotIdentifier := sourceSnapshot.GetSnapshotId()
		// Remove all whitespaces and search crn: string at 0th position
		// to finalise that user provided crn or not
		if strings.Index(strings.ReplaceAll(snapshotIdentifier, " ", ""), "crn:") == 0 {
			requestedVolume.SnapshotCRN = snapshotIdentifier
		} else {
			requestedVolume.SnapshotID = snapshotIdentifier
		}
	}

	existingVol, err := checkIfVolumeExists(session, *requestedVolume, ctxLogger)
	if existingVol != nil && err == nil {
		ctxLogger.Info("Volume already exists", zap.Reflect("ExistingVolume", existingVol))
		if existingVol.Capacity != nil && requestedVolume.Capacity != nil && *existingVol.Capacity == *requestedVolume.Capacity {
			return createCSIVolumeResponse(*existingVol, int64(*(existingVol.Capacity)*utils.GB), nil, csiCS.CSIProvider.GetClusterID(), csiCS.Driver.region), nil
		}
		return nil, commonError.GetCSIError(ctxLogger, commonError.VolumeAlreadyExists, requestID, err, name, *requestedVolume.Capacity)
	}

	// Create volume
	volumeObj, err := session.CreateVolume(*requestedVolume)
	if err != nil {
		if providerError.RetrivalFailed == providerError.GetErrorType(err) {
			return nil, commonError.GetCSIError(ctxLogger, commonError.ObjectNotFound, requestID, err, "creation")
		}
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err, "creation")
	}

	// return csi volume object
	return createCSIVolumeResponse(*volumeObj, int64(*(requestedVolume.Capacity)*utils.GB), nil, csiCS.CSIProvider.GetClusterID(), csiCS.Driver.region), nil
}

// DeleteVolume ...
func (csiCS *CSIControllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	// populate requestID in the context
	ctx = context.WithValue(ctx, provider.RequestID, requestID)
	defer metrics.UpdateDurationFromStart(ctxLogger, "DeleteVolume", time.Now())
	ctxLogger.Info("CSIControllerServer-DeleteVolume... ", zap.Reflect("Request", *req))

	// Validate arguments
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyVolumeID, requestID, nil)
	}

	// TODO:~ Following could be enhancement although currect way is working fine
	// Get the volume name by using volume ID
	// and delete volume by name

	// get the session
	session, err := csiCS.CSIProvider.GetProviderSession(ctx, ctxLogger)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.FailedPrecondition, requestID, err)
	}
	volume := &provider.Volume{}
	volume.VolumeID = volumeID

	existingVol, err := checkIfVolumeExists(session, *volume, ctxLogger)
	if existingVol == nil && err == nil {
		ctxLogger.Info("Volume not found. Returning success without deletion...")
		return &csi.DeleteVolumeResponse{}, nil
	}

	err = session.DeleteVolume(volume)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}
	return &csi.DeleteVolumeResponse{}, nil
}

// ControllerPublishVolume ...
func (csiCS *CSIControllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	// populate requestID in the context
	ctx = context.WithValue(ctx, provider.RequestID, requestID)
	ctxLogger.Info("CSIControllerServer-ControllerPublishVolume...", zap.Reflect("Request", *req))
	defer metrics.UpdateDurationFromStart(ctxLogger, metrics.FunctionLabel("ControllerPublishVolume"), time.Now())

	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyVolumeID, requestID, nil)
	}
	nodeID := req.GetNodeId()
	if len(nodeID) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyNodeID, requestID, nil)
	}

	volumeCapability := req.GetVolumeCapability()
	if volumeCapability == nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.NoVolumeCapabilities, requestID, nil)
	}

	//Allow only one active attach/detach operation for an instance at anytime
	lockWaitStart := time.Now()
	csiCS.mutex.Lock(nodeID)
	defer csiCS.mutex.Unlock(nodeID)
	defer metrics.UpdateDurationFromStart(ctxLogger, metrics.FunctionLabel("ControllerPublishVolume.Lock"), lockWaitStart)

	volumeCapabilities := []*csi.VolumeCapability{volumeCapability}
	// Validate volume capabilities, are all capabilities supported by driver or not
	if !areVolumeCapabilitiesSupported(volumeCapabilities, csiCS.Driver.vcap) {
		return nil, commonError.GetCSIError(ctxLogger, commonError.VolumeCapabilitiesNotSupported, requestID, nil)
	}

	sess, err := csiCS.CSIProvider.GetProviderSession(ctx, ctxLogger)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}

	// Validate the node instance that the volume will be attached to actually exists
	// Todo Need and API to check existence of an instance being attached to via Lib
	requestedVolume := &provider.Volume{}
	requestedVolume.VolumeID = volumeID
	volDetail, err := checkIfVolumeExists(sess, *requestedVolume, ctxLogger)
	// Volume not found
	if volDetail == nil && err == nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.ObjectNotFound, requestID, nil, volumeID)
	} else if err != nil { // In case of other errors apart from volume not  found
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}

	clusterID := csiCS.CSIProvider.GetClusterID()
	volumeAttachmentReq := provider.VolumeAttachmentRequest{
		VolumeID:   volumeID,
		InstanceID: nodeID,
		IKSVolumeAttachment: &provider.IKSVolumeAttachment{
			ClusterID: &clusterID,
		},
	}
	response, err := sess.AttachVolume(volumeAttachmentReq)
	if err != nil {
		// Node should be present if not return the error code
		if providerError.GetErrorType(err) == providerError.NodeNotFound {
			return nil, commonError.GetCSIError(ctxLogger, commonError.ObjectNotFound, requestID, err)
		}
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}

	//Pass in the VPCVolumeAttachment ID for efficient retrival in WaitForAttachVolume()
	volumeAttachmentReq.VPCVolumeAttachment = &provider.VolumeAttachment{
		ID: response.VPCVolumeAttachment.ID,
	}

	response, err = sess.WaitForAttachVolume(volumeAttachmentReq)
	if err != nil {
		//retry gap is constant in the common lib i.e 10 seconds and number of retries are 4*Retry configure in the driver
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}

	ctxLogger.Info("Attachment response", zap.Reflect("Response", response))
	controllerPublishVolumeResponse := createControllerPublishVolumeResponse(*response, map[string]string{PublishInfoRequestID: requestID})
	return controllerPublishVolumeResponse, nil
}

// ControllerUnpublishVolume ...
func (csiCS *CSIControllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	// populate requestID in the context
	ctx = context.WithValue(ctx, provider.RequestID, requestID)
	defer metrics.UpdateDurationFromStart(ctxLogger, metrics.FunctionLabel("ControllerUnpublishVolume"), time.Now())
	ctxLogger.Info("CSIControllerServer-ControllerUnpublishVolume... ", zap.Reflect("Request", *req))

	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyVolumeID, requestID, nil)
	}

	nodeID := req.GetNodeId()
	if len(nodeID) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyNodeID, requestID, nil)
	}

	//Allow only one active attach/detach operation for an instance at anytime
	csiCS.mutex.Lock(nodeID)
	defer csiCS.mutex.Unlock(nodeID)

	clusterID := csiCS.CSIProvider.GetClusterID()
	volumeAttachmentReq := provider.VolumeAttachmentRequest{
		VolumeID:   volumeID,
		InstanceID: nodeID,
		IKSVolumeAttachment: &provider.IKSVolumeAttachment{
			ClusterID: &clusterID,
		},
	}
	sess, err := csiCS.CSIProvider.GetProviderSession(ctx, ctxLogger)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}
	response, err := sess.DetachVolume(volumeAttachmentReq)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}
	err = sess.WaitForDetachVolume(volumeAttachmentReq)
	if err != nil {
		//retry gap is constant in the common lib i.e 10 seconds and number of retries are 4*Retry configure in the driver
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}
	ctxLogger.Info("Detach response", zap.Reflect("response", response))
	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

// ValidateVolumeCapabilities ...
func (csiCS *CSIControllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	// populate requestID in the context
	ctx = context.WithValue(ctx, provider.RequestID, requestID)
	ctxLogger.Info("CSIControllerServer-ValidateVolumeCapabilities", zap.Reflect("Request", *req))

	// Validate Arguments
	if req.GetVolumeCapabilities() == nil || len(req.GetVolumeCapabilities()) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.NoVolumeCapabilities, requestID, nil)
	}
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyVolumeID, requestID, nil)
	}

	// Check if Requested Volume exists
	session, err := csiCS.CSIProvider.GetProviderSession(ctx, ctxLogger)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}

	// Get volume details by using volume ID, it should exists with provider
	_, err = session.GetVolume(volumeID)
	if err != nil {
		if providerError.RetrivalFailed == providerError.GetErrorType(err) {
			return nil, commonError.GetCSIError(ctxLogger, commonError.ObjectNotFound, requestID, err, volumeID)
		}
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}

	// Setup Response
	var confirmed *csi.ValidateVolumeCapabilitiesResponse_Confirmed
	// Check if Volume Capabilities supported by the Driver Match
	if areVolumeCapabilitiesSupported(req.GetVolumeCapabilities(), csiCS.Driver.vcap) {
		confirmed = &csi.ValidateVolumeCapabilitiesResponse_Confirmed{VolumeCapabilities: req.GetVolumeCapabilities()}
	}

	// Return Response
	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: confirmed,
	}, nil
}

// ListVolumes ...
func (csiCS *CSIControllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	// populate requestID in the context
	ctx = context.WithValue(ctx, provider.RequestID, requestID)
	ctxLogger.Info("CSIControllerServer-ListVolumes...", zap.Reflect("Request", *req))
	defer metrics.UpdateDurationFromStart(ctxLogger, metrics.FunctionLabel("ListVolumes"), time.Now())

	session, err := csiCS.CSIProvider.GetProviderSession(ctx, ctxLogger)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}

	maxEntries := int(req.MaxEntries)
	tags := map[string]string{}
	volumeList, err := session.ListVolumes(maxEntries, req.StartingToken, tags)
	if err != nil {
		errCode := err.(providerError.Message).Code
		if strings.Contains(errCode, "InvalidListVolumesLimit") {
			return nil, commonError.GetCSIError(ctxLogger, commonError.InvalidParameters, requestID, err)
		} else if strings.Contains(errCode, "StartVolumeIDNotFound") {
			return nil, commonError.GetCSIError(ctxLogger, commonError.StartVolumeIDNotFound, requestID, err, req.StartingToken)
		}
		return nil, commonError.GetCSIError(ctxLogger, commonError.ListVolumesFailed, requestID, err)
	}

	entries := []*csi.ListVolumesResponse_Entry{}
	for _, vol := range volumeList.Volumes {
		if vol.Capacity != nil {
			entries = append(entries, &csi.ListVolumesResponse_Entry{
				Volume: &csi.Volume{
					VolumeId:      vol.VolumeID,
					CapacityBytes: int64(*vol.Capacity * utils.GiB),
				},
			})
		}
	}

	return &csi.ListVolumesResponse{
		Entries:   entries,
		NextToken: volumeList.Next,
	}, nil
}

// GetCapacity ...
func (csiCS *CSIControllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	// populate requestID in the context
	_ = context.WithValue(ctx, provider.RequestID, requestID)

	ctxLogger.Info("CSIControllerServer-GetCapacity", zap.Reflect("Request", *req))
	return nil, commonError.GetCSIError(ctxLogger, commonError.MethodUnimplemented, requestID, nil, "GetCapacity")
}

// ControllerGetCapabilities implements the default GRPC callout.
func (csiCS *CSIControllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	// populate requestID in the context
	_ = context.WithValue(ctx, provider.RequestID, requestID)

	ctxLogger.Info("CSIControllerServer-GetCapacity", zap.Reflect("Request", *req))
	// Return the capabilities as per provider volume capabilities
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: csiCS.Driver.cscap,
	}, nil
}

// CreateSnapshot ...
func (csiCS *CSIControllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	// populate requestID in the context
	ctx = context.WithValue(ctx, provider.RequestID, requestID)
	ctxLogger.Info("CSIControllerServer-CreateSnapshot... ", zap.Reflect("Request", *req))
	defer metrics.UpdateDurationFromStart(ctxLogger, "CreateSnapshot", time.Now())

	//Feature flag to enable/disable CreateSnapshot feature.
	if strings.ToLower(os.Getenv("IS_SNAPSHOT_ENABLED")) == "false" {
		ctxLogger.Warn("CreateSnapshot functionality is disabled.")
		time.Sleep(10 * time.Minute) //To avoid multiple retries from kubernetes to CSI Driver
		return nil, commonError.GetCSIError(ctxLogger, commonError.MethodUnimplemented, requestID, nil, "CreateSnapshot functionality is disabled.")
	}

	snapshotName := req.GetName()
	if len(snapshotName) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.MissingSnapshotName, requestID, nil)
	}

	sourceVolumeID := req.GetSourceVolumeId()
	if len(sourceVolumeID) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.MissingSourceVolumeID, requestID, nil)
	}

	// Validate if volume Already Exists
	session, err := csiCS.CSIProvider.GetProviderSession(ctx, ctxLogger)
	if err != nil {
		if userError.GetUserErrorCode(err) == string(utilReasonCode.EndpointNotReachable) {
			return nil, commonError.GetCSIError(ctxLogger, commonError.EndpointNotReachable, requestID, err)
		}
		if userError.GetUserErrorCode(err) == string(utilReasonCode.Timeout) {
			return nil, commonError.GetCSIError(ctxLogger, commonError.Timeout, requestID, err)
		}
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}

	snapshot, err := session.GetSnapshotByName(snapshotName)
	if snapshot != nil {
		if snapshot.VolumeID != sourceVolumeID {
			return nil, commonError.GetCSIError(ctxLogger, commonError.SnapshotAlreadyExists, requestID, err, snapshotName, sourceVolumeID)
		}
		ctxLogger.Info("Snapshot with name already exist for volume", zap.Reflect("SnapshotName", snapshotName), zap.Reflect("VolumeID", sourceVolumeID))
		return createCSISnapshotResponse(*snapshot), nil
	}
	snapshotParameters := provider.SnapshotParameters{}
	snapshotParameters.Name = snapshotName
	snapshotTags := map[string]string{
		"name": snapshotName,
	}
	snapshotParameters.SnapshotTags = snapshotTags

	snapshot, err = session.CreateSnapshot(sourceVolumeID, snapshotParameters)

	if err != nil {
		time.Sleep(time.Duration(getMaxDelaySnapshotCreate(ctxLogger)) * time.Second) //To avoid multiple retries from kubernetes to CSI Driver
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err, "creation")
	}
	return createCSISnapshotResponse(*snapshot), nil
}

// DeleteSnapshot ...
func (csiCS *CSIControllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	// populate requestID in the context
	ctx = context.WithValue(ctx, provider.RequestID, requestID)
	defer metrics.UpdateDurationFromStart(ctxLogger, "DeleteSnapshot", time.Now())
	ctxLogger.Info("CSIControllerServer-DeleteSnapshot... ", zap.Reflect("Request", *req))

	// Validate arguments
	snapshotID := req.GetSnapshotId()
	if len(snapshotID) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptySnapshotID, requestID, nil)
	}

	// get the session
	session, err := csiCS.CSIProvider.GetProviderSession(ctx, ctxLogger)
	if err != nil {
		if userError.GetUserErrorCode(err) == string(utilReasonCode.EndpointNotReachable) {
			return nil, commonError.GetCSIError(ctxLogger, commonError.EndpointNotReachable, requestID, err)
		}
		if userError.GetUserErrorCode(err) == string(utilReasonCode.Timeout) {
			return nil, commonError.GetCSIError(ctxLogger, commonError.Timeout, requestID, err)
		}
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}

	snapshot := &provider.Snapshot{}
	snapshot.SnapshotID, _ = getSnapshotAndAccountIDsFromCRN(snapshotID)

	err = session.DeleteSnapshot(snapshot)
	if err != nil {
		if providerError.RetrivalFailed == providerError.GetErrorType(err) {
			ctxLogger.Info("Snapshot not found. Returning success without deletion...")
			return &csi.DeleteSnapshotResponse{}, nil
		}
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}
	return &csi.DeleteSnapshotResponse{}, nil
}

// ListSnapshots ...
func (csiCS *CSIControllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	// populate requestID in the context
	ctx = context.WithValue(ctx, provider.RequestID, requestID)
	ctxLogger.Info("CSIControllerServer-ListSnapshots...", zap.Reflect("Request", *req))
	defer metrics.UpdateDurationFromStart(ctxLogger, metrics.FunctionLabel("ListSnapshots"), time.Now())

	session, err := csiCS.CSIProvider.GetProviderSession(ctx, ctxLogger)
	if err != nil {
		if userError.GetUserErrorCode(err) == string(utilReasonCode.EndpointNotReachable) {
			return nil, commonError.GetCSIError(ctxLogger, commonError.EndpointNotReachable, requestID, err)
		}
		if userError.GetUserErrorCode(err) == string(utilReasonCode.Timeout) {
			return nil, commonError.GetCSIError(ctxLogger, commonError.Timeout, requestID, err)
		}
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}

	entries := []*csi.ListSnapshotsResponse_Entry{}
	snapshotID := req.GetSnapshotId()
	snapID, snapshotAccountID := getSnapshotAndAccountIDsFromCRN(snapshotID)

	if len(snapID) != 0 && csiCS.Driver.accountID == snapshotAccountID { // in case snapshotID's account and cluster account ID is same
		snapshot, err := session.GetSnapshot(snapID)
		if snapshot == nil {
			return &csi.ListSnapshotsResponse{}, nil
		}
		if providerError.RetrivalFailed == providerError.GetErrorType(err) {
			ctxLogger.Info("Snapshot not found. Returning success ...")
			return &csi.ListSnapshotsResponse{}, nil
		}
		return &csi.ListSnapshotsResponse{
			Entries: append(entries, &csi.ListSnapshotsResponse_Entry{
				Snapshot: createCSISnapshotResponse(*snapshot).Snapshot,
			}),
			NextToken: "",
		}, nil
	} else { // in case of cross account snapshot restore
		return &csi.ListSnapshotsResponse{
			Entries: append(entries, &csi.ListSnapshotsResponse_Entry{
				Snapshot: &csi.Snapshot{
					SnapshotId:     snapshotID,
					SourceVolumeId: "",
					ReadyToUse:     true,
				},
			}),
			NextToken: "",
		}, nil
	}

	maxEntries := int(req.MaxEntries) //nolint
	tags := map[string]string{}
	sourceVolumeID := req.GetSourceVolumeId()
	if len(sourceVolumeID) != 0 {
		tags["source_volume.id"] = sourceVolumeID
	}
	snapshotList, err := session.ListSnapshots(maxEntries, req.StartingToken, tags)
	if err != nil {
		errCode := err.(providerError.Message).Code
		if strings.Contains(errCode, "InvalidListSnapshotLimit") {
			return nil, commonError.GetCSIError(ctxLogger, commonError.InvalidParameters, requestID, err)
		} else if strings.Contains(errCode, "StartSnapshotIDNotFound") {
			return nil, commonError.GetCSIError(ctxLogger, commonError.StartSnapshotIDNotFound, requestID, err, req.StartingToken)
		}
		return nil, commonError.GetCSIError(ctxLogger, commonError.ListSnapshotsFailed, requestID, err)
	}

	for _, snap := range snapshotList.Snapshots {
		snapObj := createCSISnapshotResponse(*snap)
		entries = append(entries, &csi.ListSnapshotsResponse_Entry{
			Snapshot: snapObj.Snapshot,
		})
	}

	return &csi.ListSnapshotsResponse{
		Entries:   entries,
		NextToken: snapshotList.Next,
	}, nil
}

// getSnapshots ...
func (csiCS *CSIControllerServer) getSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	// populate requestID in the context
	_ = context.WithValue(ctx, provider.RequestID, requestID)

	ctxLogger.Info("CSIControllerServer-getSnapshots", zap.Reflect("Request", *req))
	return nil, commonError.GetCSIError(ctxLogger, commonError.MethodUnimplemented, requestID, nil, "getSnapshots")
}

// getSnapshotById ...
func (csiCS *CSIControllerServer) getSnapshotByID(ctx context.Context, snapshotID string) (*csi.ListSnapshotsResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	// populate requestID in the context
	_ = context.WithValue(ctx, provider.RequestID, requestID)

	ctxLogger.Info("CSIControllerServer-getSnapshotByID", zap.Reflect("Request", snapshotID))
	return nil, commonError.GetCSIError(ctxLogger, commonError.MethodUnimplemented, requestID, nil, "getSnapshotByID")
}

// ControllerExpandVolume ...
func (csiCS *CSIControllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	// populate requestID in the context
	_ = context.WithValue(ctx, provider.RequestID, requestID)

	ctxLogger.Info("CSIControllerServer-ControllerExpandVolume", zap.Reflect("Request", *req))
	volumeID := req.GetVolumeId()
	capacity := req.GetCapacityRange().GetRequiredBytes()
	if len(volumeID) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyVolumeID, requestID, nil)
	}

	// get the session
	session, err := csiCS.CSIProvider.GetProviderSession(ctx, ctxLogger)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.FailedPrecondition, requestID, err)
	}
	requestedVolume := &provider.Volume{}
	requestedVolume.VolumeID = volumeID
	volDetail, err := checkIfVolumeExists(session, *requestedVolume, ctxLogger)
	// Volume not found
	if volDetail == nil && err == nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.ObjectNotFound, requestID, nil, volumeID)
	} else if err != nil { // In case of other errors apart from volume not  found
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}

	volumeExpansionReq := provider.ExpandVolumeRequest{
		VolumeID: volumeID,
		Capacity: capacity,
	}
	_, err = session.ExpandVolume(volumeExpansionReq)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.InternalError, requestID, err)
	}
	return &csi.ControllerExpandVolumeResponse{CapacityBytes: capacity, NodeExpansionRequired: true}, nil
}

// ControllerGetVolume ...
func (csiCS *CSIControllerServer) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	return nil, commonError.GetCSIError(ctxLogger, commonError.MethodUnimplemented, requestID, nil, "ControllerGetVolume")
}

// ControllerModifyVolume ...
func (csiCS *CSIControllerServer) ControllerModifyVolume(ctx context.Context, req *csi.ControllerModifyVolumeRequest) (*csi.ControllerModifyVolumeResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	return nil, commonError.GetCSIError(ctxLogger, commonError.MethodUnimplemented, requestID, nil, "ControllerModifyVolume")
}

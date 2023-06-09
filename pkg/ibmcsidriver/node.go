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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"

	"os/exec"
	"time"

	commonError "github.com/IBM/ibm-csi-common/pkg/messages"
	nodeMetadata "github.com/IBM/ibm-csi-common/pkg/metadata"
	"github.com/IBM/ibm-csi-common/pkg/metrics"
	"github.com/IBM/ibm-csi-common/pkg/mountmanager"
	"github.com/IBM/ibm-csi-common/pkg/utils"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
	"k8s.io/kubernetes/pkg/volume/util/fs"
	mount "k8s.io/mount-utils"
)

// CSINodeServer ...
type CSINodeServer struct {
	Driver   *IBMCSIDriver
	Mounter  mountmanager.Mounter
	Metadata nodeMetadata.NodeMetadata
	Stats    StatsUtils
	// TODO: Only lock mutually exclusive calls and make locking more fine grained
	mux sync.Mutex
}

// StatsUtils ...
type StatsUtils interface {
	FSInfo(path string) (int64, int64, int64, int64, int64, int64, error)
	IsBlockDevice(devicePath string) (bool, error)
	DeviceInfo(devicePath string) (int64, error)
	IsDevicePathNotExist(devicePath string) bool
}

// VolumeStatUtils ...
type VolumeStatUtils struct {
}

// VolumeMountUtils ...
type VolumeMountUtils struct {
}

// FSInfo ...
func (su *VolumeStatUtils) FSInfo(path string) (int64, int64, int64, int64, int64, int64, error) {
	return fs.Info(path)
}

const (
	// DefaultVolumesPerNode is the default number of volumes attachable to a node
	DefaultVolumesPerNode = 4

	// MaxVolumesPerNode is the maximum number of volumes attachable to a node
	MaxVolumesPerNode = 12

	// MinimumCoresWithMaximumAttachableVolumes is the minimum cores required to have maximum number of attachable volumes, currently 4 as per the docs.
	MinimumCoresWithMaximumAttachableVolumes = 4

	// FSTypeExt2 represents the ext2 filesystem type
	FSTypeExt2 = "ext2"

	// FSTypeExt3 represents the ext3 filesystem type
	FSTypeExt3 = "ext3"

	// FSTypeExt4 represents the ext4 filesystem type
	FSTypeExt4 = "ext4"

	// FSTypeXfs represents te xfs filesystem type
	FSTypeXfs = "xfs"

	// default file system type to be used when it is not provided
	defaultFsType = FSTypeExt4
)

var _ csi.NodeServer = &CSINodeServer{}

// NodePublishVolume ...
func (csiNS *CSINodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	publishContext := req.GetPublishContext()
	controlleRequestID := publishContext[PublishInfoRequestID]
	ctxLogger, requestID := utils.GetContextLoggerWithRequestID(ctx, false, &controlleRequestID)
	ctxLogger.Info("CSINodeServer-NodePublishVolume...", zap.Reflect("Request", *req))
	metrics.UpdateDurationFromStart(ctxLogger, "NodePublishVolume", time.Now())
	csiNS.mux.Lock()
	defer csiNS.mux.Unlock()

	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyVolumeID, requestID, nil)
	}

	source := req.GetStagingTargetPath()
	if len(source) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.NoStagingTargetPath, requestID, nil)
	}

	target := req.GetTargetPath()
	if len(target) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.NoTargetPath, requestID, nil)
	}

	volumeCapability := req.GetVolumeCapability()
	if volumeCapability == nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.NoVolumeCapabilities, requestID, nil)
	}

	volumeCapabilities := []*csi.VolumeCapability{volumeCapability}
	// Validate volume capabilities, are all capabilities supported by driver or not
	if !areVolumeCapabilitiesSupported(volumeCapabilities, csiNS.Driver.vcap) {
		return nil, commonError.GetCSIError(ctxLogger, commonError.VolumeCapabilitiesNotSupported, requestID, nil)
	}

	// Check if targetPath is already mounted. If it already moounted return OK
	notMounted, err := csiNS.Mounter.IsLikelyNotMountPoint(target)
	if err != nil && !os.IsNotExist(err) {
		//Error other than PathNotExists
		ctxLogger.Error(fmt.Sprintf("Can not validate target mount point: %s %v", target, err))
		return nil, commonError.GetCSIError(ctxLogger, commonError.MountPointValidateError, requestID, err, target)
	}
	// Its OK if IsLikelyNotMountPoint returns PathNotExists error
	if !notMounted {
		// The  target Path is already mounted, Retrun OK
		/* TODO
		1) Target Path MUST be the vol referenced by vol ID
		2) Check volume capability matches for ALREADY_EXISTS
		3) Readonly MUST match
		*/
		return &csi.NodePublishVolumeResponse{}, nil
	}
	// Perform a bind mount to the full path to allow duplicate mounts of the same PD.
	options := []string{"bind"}
	readOnly := req.GetReadonly()
	if readOnly {
		options = append(options, "ro")
	}
	fsType := "" // Let the fsType be derived from global mount(NodeStageVolume)

	var nodePublishResponse *csi.NodePublishVolumeResponse
	var mountErr error

	switch volumeCapability.GetAccessType().(type) {
	case *csi.VolumeCapability_Block:
		nodePublishResponse, mountErr = csiNS.processMountForBlock(ctxLogger, requestID, publishContext[PublishInfoDevicePath], target, volumeID, options)

	case *csi.VolumeCapability_Mount:
		nodePublishResponse, mountErr = csiNS.processMount(ctxLogger, requestID, source, target, fsType, options)
	}

	ctxLogger.Info("CSINodeServer-NodePublishVolume response...", zap.Reflect("Response", nodePublishResponse), zap.Error(mountErr))
	return nodePublishResponse, mountErr
}

// NodeUnpublishVolume ...
func (csiNS *CSINodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	ctxLogger.Info("CSINodeServer-NodeUnpublishVolume...", zap.Reflect("Request", *req))
	metrics.UpdateDurationFromStart(ctxLogger, "NodeUnpublishVolume", time.Now())
	csiNS.mux.Lock()
	defer csiNS.mux.Unlock()
	// Validate Arguments
	targetPath := req.GetTargetPath()
	volID := req.GetVolumeId()
	if len(volID) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyVolumeID, requestID, nil)
	}
	if len(targetPath) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.NoTargetPath, requestID, nil)
	}

	ctxLogger.Info("Unmounting  target path", zap.String("targetPath", targetPath))
	err := mount.CleanupMountPoint(targetPath, csiNS.Mounter, false /* bind mount */)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.UnmountFailed, requestID, err, targetPath)
	}

	nodeUnpublishVolumeResponse := &csi.NodeUnpublishVolumeResponse{}
	ctxLogger.Info("Successfully unmounted  target path", zap.String("targetPath", targetPath), zap.Error(err))
	return nodeUnpublishVolumeResponse, err
}

// NodeStageVolume ...
func (csiNS *CSINodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	publishContext := req.GetPublishContext()
	controlleRequestID := publishContext[PublishInfoRequestID]
	ctxLogger, requestID := utils.GetContextLoggerWithRequestID(ctx, false, &controlleRequestID)
	ctxLogger.Info("CSINodeServer-NodeStageVolume...", zap.Reflect("Request", *req))
	metrics.UpdateDurationFromStart(ctxLogger, "NodeStageVolume", time.Now())

	csiNS.mux.Lock()
	defer csiNS.mux.Unlock()

	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyVolumeID, requestID, nil)
	}
	stagingTargetPath := req.GetStagingTargetPath()
	if len(stagingTargetPath) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.NoStagingTargetPath, requestID, nil)
	}
	volumeCapability := req.GetVolumeCapability()
	if volumeCapability == nil || volumeCapability.AccessMode.GetMode() == csi.VolumeCapability_AccessMode_UNKNOWN {
		return nil, commonError.GetCSIError(ctxLogger, commonError.NoVolumeCapabilities, requestID, nil)
	}

	volumeCapabilities := []*csi.VolumeCapability{volumeCapability}
	// Validate volume capabilities, are all capabilities supported by driver or not
	if !areVolumeCapabilitiesSupported(volumeCapabilities, csiNS.Driver.vcap) {
		return nil, commonError.GetCSIError(ctxLogger, commonError.VolumeCapabilitiesNotSupported, requestID, nil)
	}

	// If the access type is block, do nothing for stage.
	if volumeCapability != nil {
		if blk := volumeCapability.GetBlock(); blk != nil {
			klog.V(4).InfoS("NodeStageVolume: called. Since it is a block device, ignoring...", "volumeID", volumeID)
			return &csi.NodeStageVolumeResponse{}, nil
		}
	}

	// Check devicePath is available in the publish context
	devicePath := publishContext[PublishInfoDevicePath]
	if len(devicePath) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyDevicePath, requestID, nil)
	}
	// Check source Path
	source, err := csiNS.findDevicePathSource(ctxLogger, devicePath, volumeID)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.DevicePathFindFailed, requestID, nil, devicePath)
	}
	ctxLogger.Info("Found device path ", zap.String("devicePath", devicePath), zap.String("source", source))

	// Check target path
	exists, err := csiNS.Mounter.PathExists(stagingTargetPath)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.TargetPathCheckFailed, requestID, err, stagingTargetPath)
	}

	// Create the target directory if does not exist.
	if !exists {
		// If target path does not exist we need to create the directory where volume will be staged
		ctxLogger.Info("Creating target directory ", zap.String("stagingTargetPath", stagingTargetPath))
		if err = csiNS.Mounter.MakeDir(stagingTargetPath); err != nil {
			return nil, commonError.GetCSIError(ctxLogger, commonError.TargetPathCreateFailed, requestID, err, stagingTargetPath)
		}
	}
	// Check if a device is mounted in target directory
	device, _, err := mount.GetDeviceNameFromMount(csiNS.Mounter, stagingTargetPath)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.VolumeMountCheckFailed, requestID, err, stagingTargetPath)
	}

	// This operation (NodeStageVolume) MUST be idempotent.
	// If the volume corresponding to the volume_id is already staged to the staging_target_path,
	// and is identical to the specified volume_capability the Plugin MUST reply 0 OK.
	target, err := filepath.EvalSymlinks(source)
	if err == nil && device == target {
		ctxLogger.Info("volume already staged", zap.String("volumeID", volumeID))
		return &csi.NodeStageVolumeResponse{}, nil
	}

	mnt := volumeCapability.GetMount()
	options := mnt.MountFlags
	// find  FS type
	fsType := defaultFsType
	if mnt.FsType != "" {
		fsType = mnt.FsType
	}

	// FormatAndMount will format only if needed
	ctxLogger.Info("Formating and mounting ", zap.String("source", source), zap.String("stagingTargetPath", stagingTargetPath), zap.String("fsType", fsType), zap.Reflect("options", options))
	err = csiNS.Mounter.GetSafeFormatAndMount().FormatAndMount(source, stagingTargetPath, fsType, options)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.FormatAndMountFailed, requestID, err, source, stagingTargetPath)
	}

	if _, err := csiNS.Mounter.Resize(devicePath, stagingTargetPath); err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.FileSystemResizeFailed, requestID, err)
	}

	nodeStageVolumeResponse := &csi.NodeStageVolumeResponse{}
	return nodeStageVolumeResponse, err
}

// NodeUnstageVolume ...
func (csiNS *CSINodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	ctxLogger.Info("CSINodeServer-NodeUnstageVolume ... ", zap.Reflect("Request", *req))
	metrics.UpdateDurationFromStart(ctxLogger, "NodeUnstageVolume", time.Now())
	csiNS.mux.Lock()
	defer csiNS.mux.Unlock()

	// Validate arguments
	volumeID := req.GetVolumeId()
	stagingTargetPath := req.GetStagingTargetPath()
	if len(volumeID) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyVolumeID, requestID, nil)
	}
	if len(stagingTargetPath) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.NoStagingTargetPath, requestID, nil)
	}

	ctxLogger.Info("Unmounting staging target path", zap.String("stagingTargetPath", stagingTargetPath))
	err := mount.CleanupMountPoint(stagingTargetPath, csiNS.Mounter, false /* bind mount */)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.UnmountFailed, requestID, err, stagingTargetPath)
	}

	ctxLogger.Info("Successfully Unmounted staging target path", zap.String("stagingTargetPath", stagingTargetPath))
	nodeUnstageVolumeResponse := &csi.NodeUnstageVolumeResponse{}
	return nodeUnstageVolumeResponse, err
}

// NodeGetCapabilities ...
func (csiNS *CSINodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	ctxLogger, _ := utils.GetContextLogger(ctx, false)
	ctxLogger.Info("CSINodeServer-NodeGetCapabilities... ", zap.Reflect("Request", *req))

	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: csiNS.Driver.nscap,
	}, nil
}

// NodeGetInfo ...
func (csiNS *CSINodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	ctxLogger.Info("CSINodeServer-NodeGetInfo... ", zap.Reflect("Request", *req))

	nodeName := os.Getenv("KUBE_NODE_NAME")

	nodeInfo := nodeMetadata.NodeInfoManager{
		NodeName: nodeName,
	}

	// Check if node metadata service initialized properly
	if csiNS.Metadata == nil {
		metadata, err := nodeInfo.NewNodeMetadata(ctxLogger)
		if err != nil {
			ctxLogger.Error("Failed to initialize node metadata", zap.Error(err))
			return nil, commonError.GetCSIError(ctxLogger, commonError.NodeMetadataInitFailed, requestID, err)
		}
		csiNS.Metadata = metadata
	}

	top := &csi.Topology{
		Segments: map[string]string{
			utils.NodeRegionLabel: csiNS.Metadata.GetRegion(),
			utils.NodeZoneLabel:   csiNS.Metadata.GetZone(),
		},
	}

	resp := &csi.NodeGetInfoResponse{
		NodeId:             csiNS.Metadata.GetWorkerID(),
		AccessibleTopology: top,
	}
	ctxLogger.Info("NodeGetInfoResponse", zap.Reflect("NodeGetInfoResponse", resp))
	return resp, nil
}

// NodeGetVolumeStats ...
func (csiNS *CSINodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	var resp *csi.NodeGetVolumeStatsResponse
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	ctxLogger.Info("CSINodeServer-NodeGetVolumeStats... ", zap.Reflect("Request", *req)) //nolint:staticcheck
	metrics.UpdateDurationFromStart(ctxLogger, "NodeGetVolumeStats", time.Now())
	if req == nil || req.VolumeId == "" { //nolint:staticcheck
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyVolumeID, requestID, nil)
	}

	if req.VolumePath == "" {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyVolumePath, requestID, nil)
	}

	volumePath := req.VolumePath
	// Return if path does not exist
	if csiNS.Stats.IsDevicePathNotExist(volumePath) {
		return nil, commonError.GetCSIError(ctxLogger, commonError.DevicePathNotExists, requestID, nil, volumePath, req.VolumeId)
	}

	// check if volume mode is raw volume mode
	isBlock, err := csiNS.Stats.IsBlockDevice(volumePath)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.BlockDeviceCheckFailed, requestID, err, req.VolumeId)
	}
	// if block device, get deviceStats
	if isBlock {
		capacity, err := csiNS.Stats.DeviceInfo(volumePath)
		if err != nil {
			return nil, commonError.GetCSIError(ctxLogger, commonError.GetDeviceInfoFailed, requestID, err)
		}

		resp = &csi.NodeGetVolumeStatsResponse{
			Usage: []*csi.VolumeUsage{
				{
					Total: capacity,
					Unit:  csi.VolumeUsage_BYTES,
				},
			},
		}

		ctxLogger.Info("Response for Volume stats", zap.Reflect("Response", resp))
		return resp, nil
	}

	// else get the file system stats
	available, capacity, usage, inodes, inodesFree, inodesUsed, err := csiNS.Stats.FSInfo(volumePath)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.GetFSInfoFailed, requestID, err)
	}
	resp = &csi.NodeGetVolumeStatsResponse{
		Usage: []*csi.VolumeUsage{
			{
				Available: available,
				Total:     capacity,
				Used:      usage,
				Unit:      csi.VolumeUsage_BYTES,
			},
			{
				Available: inodesFree,
				Total:     inodes,
				Used:      inodesUsed,
				Unit:      csi.VolumeUsage_INODES,
			},
		},
	}

	ctxLogger.Info("Response for Volume stats", zap.Reflect("Response", resp))
	return resp, nil
}

// NodeExpandVolume ...
func (csiNS *CSINodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	ctxLogger, requestID := utils.GetContextLogger(ctx, false)
	ctxLogger.Info("CSINodeServer-NodeExpandVolume", zap.Reflect("Request", *req))
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyVolumeID, requestID, nil)
	}
	volumePath := req.GetVolumePath()
	if len(volumePath) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyVolumePath, requestID, nil)
	}

	volumeCapability := req.GetVolumeCapability()
	isBlock := false
	// VolumeCapability is optional, if specified, use that as source of truth.
	if volumeCapability != nil {
		volumeCapabilities := []*csi.VolumeCapability{volumeCapability}
		if !areVolumeCapabilitiesSupported(volumeCapabilities, csiNS.Driver.vcap) {
			return nil, commonError.GetCSIError(ctxLogger, commonError.VolumeCapabilitiesNotSupported, requestID, nil)
		}
		isBlock = volumeCapability.GetBlock() != nil
	} else {
		// VolumeCapability is nil, check if volumePath points to a block device.
		var err error
		isBlock, err = csiNS.Stats.IsBlockDevice(volumePath)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to determine if volumePath [%v] is a block device: %v", volumePath, err)
		}
	}
	// Noop for block NodeExpandVolume.
	if isBlock {
		capacity, err := csiNS.Stats.DeviceInfo(volumePath)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get block capacity on path %s: %v", volumePath, err)
		}
		klog.V(4).InfoS("NodeExpandVolume: called, since given volumePath is a block device, ignoring...", "volumeID", volumeID, "volumePath", volumePath)
		return &csi.NodeExpandVolumeResponse{CapacityBytes: capacity}, nil
	}

	notMounted, err := csiNS.Mounter.IsLikelyNotMountPoint(volumePath)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.ObjectNotFound, requestID, err, volumePath)
	}

	if notMounted {
		return nil, commonError.GetCSIError(ctxLogger, commonError.VolumePathNotMounted, requestID, nil, volumePath)
	}

	devicePath, _, err := mount.GetDeviceNameFromMount(csiNS.Mounter, volumePath)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.GetDeviceInfoFailed, requestID, err, volumePath)
	}

	if devicePath == "" {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyDevicePath, requestID, err)
	}

	if _, err := csiNS.Mounter.Resize(devicePath, volumePath); err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.FileSystemResizeFailed, requestID, err)
	}
	return &csi.NodeExpandVolumeResponse{CapacityBytes: req.CapacityRange.RequiredBytes}, nil
}

// IsBlockDevice ...
func (su *VolumeStatUtils) IsBlockDevice(devicePath string) (bool, error) {
	var stat unix.Stat_t
	err := unix.Stat(devicePath, &stat)
	if err != nil {
		return false, err
	}

	return (stat.Mode & unix.S_IFMT) == unix.S_IFBLK, nil
}

// DeviceInfo ...
func (su *VolumeStatUtils) DeviceInfo(devicePath string) (int64, error) {
	// See http://man7.org/linux/man-pages/man8/blockdev.8.html for details
	output, err := exec.Command("blockdev", "--getsize64", devicePath).CombinedOutput() // #nosec G204: The blockdev is command which allows on to call block device ioctls so we must pass in a dynamic value here.
	if err != nil {
		return 0, fmt.Errorf("failed to get size of block volume at path %s: output: %s, err: %v", devicePath, string(output), err)
	}
	strOut := strings.TrimSpace(string(output))
	gotSizeBytes, err := strconv.ParseInt(strOut, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse size '%s' into int", strOut)
	}

	return gotSizeBytes, nil
}

// IsDevicePathNotExist ...
func (su *VolumeStatUtils) IsDevicePathNotExist(devicePath string) bool {
	var stat unix.Stat_t
	err := unix.Stat(devicePath, &stat)
	if err != nil {
		if os.IsNotExist(err) {
			return true
		}
	}
	return false
}

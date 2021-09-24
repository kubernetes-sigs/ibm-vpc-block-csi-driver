/**
 * Copyright 2021 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package messages ...
package messages

import (
	"google.golang.org/grpc/codes"
)

// messagesEn ...
var messagesEn = map[string]Message{
	MethodUnimplemented: {
		Code:        MethodUnimplemented,
		Description: "'%s' CSI interface method not yet implemented",
		Type:        codes.Unimplemented,
		Action:      "Please do not use this method as its not implemented yet",
	},
	MissingVolumeName: {
		Code:        MissingVolumeName,
		Description: "Volume name not provided",
		Type:        codes.InvalidArgument,
		Action:      "Please provide volume name while creating volume",
	},
	NoVolumeCapabilities: {
		Code:        NoVolumeCapabilities,
		Description: "Volume capabilities must be provided",
		Type:        codes.InvalidArgument,
		Action:      "Please provide volume capabilities in the storage class before creating volume",
	},
	VolumeCapabilitiesNotSupported: {
		Code:        VolumeCapabilitiesNotSupported,
		Description: "Volume capabilities not supported",
		Type:        codes.InvalidArgument,
		Action:      "Please provide valid volume capabilities while creating volume",
	},
	InvalidParameters: {
		Code:        InvalidParameters,
		Description: "Failed to extract parameters",
		Type:        codes.InvalidArgument,
		Action:      "Please provide valid parameters",
	},
	ObjectNotFound: {
		Code:        ObjectNotFound,
		Description: "Object not found",
		Type:        codes.NotFound,
		Action:      "Please check 'BackendError' tag for more details",
	},
	InternalError: {
		Code:        InternalError,
		Description: "Internal error occurred",
		Type:        codes.Internal,
		Action:      "Please check 'BackendError' tag for more details",
	},
	VolumeAlreadyExists: {
		Code:        VolumeAlreadyExists,
		Description: "Volume with name '%s' already exists with same name and it is incompatible size '%s'",
		Type:        codes.AlreadyExists,
		Action:      "Please provide different name or have same size of existing volume",
	},
	VolumeInvalidArguments: {
		Code:        VolumeInvalidArguments,
		Description: "Invalid arguments for create volume",
		Type:        codes.InvalidArgument,
		Action:      "Please provide valid arguments while creating volume",
	},
	VolumeCreationFailed: {
		Code:        VolumeCreationFailed,
		Description: "Failed to create volume",
		Type:        codes.Internal,
		Action:      "Please check the error which return in BackendError tag",
	},
	EmptyVolumeID: {
		Code:        EmptyVolumeID,
		Description: "VolumeID must be provided",
		Type:        codes.InvalidArgument,
		Action:      "Please provide volume ID for attach/detach or delete it",
	},
	EmptyNodeID: {
		Code:        EmptyNodeID,
		Description: "NodeID is empty",
		Type:        codes.InvalidArgument,
		Action:      "Please check all node's labels by using kubectl command",
	},
	FailedPrecondition: {
		Code:        FailedPrecondition,
		Description: "Provider is not ready to responde",
		Type:        codes.FailedPrecondition,
		Action:      "Please retry after some time, if problem persist then report issue to IKS storage team",
	},
	NoStagingTargetPath: {
		Code:        NoStagingTargetPath,
		Description: "Staging target path not provided",
		Type:        codes.InvalidArgument,
		Action:      "Please check if there is any error in POD describe related with volume attach",
	},
	NoTargetPath: {
		Code:        NoTargetPath,
		Description: "Target path must be provided",
		Type:        codes.InvalidArgument,
		Action:      "Please check if there is any error in POD describe related with volume attach",
	},
	MountPointValidateError: {
		Code:        MountPointValidateError,
		Description: "Failed to check whether target path '%s' is a mount point",
		Type:        codes.FailedPrecondition,
		Action:      "Please check if there is any error in POD describe related with volume attach",
	},
	UnmountFailed: {
		Code:        UnmountFailed,
		Description: "Unmount failed for '%s' target path",
		Type:        codes.Internal,
		Action:      "Please check if there is any error in POD describe related with volume detach",
	},
	MountFailed: {
		Code:        MountFailed,
		Description: "Failed to mount '%q' at '%q'",
		Type:        codes.Internal,
		Action:      "Please check if there is any error in POD describe related with volume attach",
	},
	EmptyDevicePath: {
		Code:        EmptyDevicePath,
		Description: "Staging device path must be provided",
		Type:        codes.InvalidArgument,
		Action:      "Please check if there is any error in POD describe related with volume attach",
	},
	DevicePathFindFailed: {
		Code:        DevicePathFindFailed,
		Description: "Failed to find '%s' device path",
		Type:        codes.Internal,
		Action:      "Please check if there is any error in POD describe related with volume attach",
	},
	TargetPathCheckFailed: {
		Code:        TargetPathCheckFailed,
		Description: "Failed to check if staging target path '%s' exists",
		Type:        codes.Internal,
		Action:      "Please check if there is any error in POD describe related with volume attach",
	},
	TargetPathCreateFailed: {
		Code:        TargetPathCreateFailed,
		Description: "Failed to create target path '%s'",
		Type:        codes.Internal,
		Action:      "Please check if there is any error in POD describe related with volume attach",
	},
	VolumeMountCheckFailed: {
		Code:        VolumeMountCheckFailed,
		Description: "Failed to check if volume is already mounted on '%s'",
		Type:        codes.Internal,
		Action:      "Please check if there is any error in POD describe related with volume attach",
	},
	FormatAndMountFailed: {
		Code:        FormatAndMountFailed,
		Description: "Failed to format '%s' and mount it at '%s'",
		Type:        codes.Internal,
		Action:      "Please check if there is any error in POD describe related with volume attach",
	},
	NodeMetadataInitFailed: {
		Code:        NodeMetadataInitFailed,
		Description: "Failed to initialize node metadata",
		Type:        codes.NotFound, //i.e correct no need to change to other code
		Action:      "Please check the node labels as per BackendError, accordingly you may add the labels manually",
	},
	EmptyVolumePath: {
		Code:        EmptyVolumePath,
		Description: "Volume path can not be empty",
		Type:        codes.InvalidArgument,
		Action:      "Please check if volume is used by POD properly",
	},
	DevicePathNotExists: {
		Code:        DevicePathNotExists,
		Description: "Device path '%s' does not exist for volume ID '%s'",
		Type:        codes.NotFound,
		Action:      "Please check if volume is used by POD properly",
	},
	BlockDeviceCheckFailed: {
		Code:        BlockDeviceCheckFailed,
		Description: "Failed to determine if volume '%s' is block device or not",
		Type:        codes.Internal,
		Action:      "Please check if volume is used by POD properly",
	},
	GetDeviceInfoFailed: {
		Code:        GetDeviceInfoFailed,
		Description: "Failed to get device info",
		Type:        codes.Internal,
		Action:      "Please check if volume is used by POD properly",
	},
	GetFSInfoFailed: {
		Code:        GetFSInfoFailed,
		Description: "Failed to get FS info",
		Type:        codes.Internal,
		Action:      "Please check if volume is used by POD properly",
	},
	DriverNotConfigured: {
		Code:        DriverNotConfigured,
		Description: "Driver name not configured",
		Type:        codes.Unavailable,
		Action:      "Developer need to set the driver name",
	},
	RemoveMountTargetFailed: {
		Code:        RemoveMountTargetFailed,
		Description: "Failed to remove '%q' mount target",
		Type:        codes.Internal,
		Action:      "Please check if volume is used by POD properly",
	},
	CreateMountTargetFailed: {
		Code:        CreateMountTargetFailed,
		Description: "Failed to create '%q' mount target",
		Type:        codes.Internal,
		Action:      "Please check if volume is used by POD properly",
	},
	ListVolumesFailed: {
		Code:        ListVolumesFailed,
		Description: "Failed to list volumes",
		Type:        codes.Internal,
		Action:      "Please check 'BackendError' tag for more details",
	},
	StartVolumeIDNotFound: {
		Code:        StartVolumeIDNotFound,
		Description: "The volume ID '%s' specified in the start parameter of the list volume call could not be found",
		Type:        codes.Aborted,
		Action:      "Please verify that the start volume ID is correct and whether you have access to the volume ID",
	},
	FileSystemResizeFailed: {
		Code:        FileSystemResizeFailed,
		Description: "Failed to resize the file system",
		Type:        codes.Internal,
		Action:      "Please check if there is any error in PVC describe related with volume resize",
	},
	VolumePathNotMounted: {
		Code:        VolumePathNotMounted,
		Description: "VolumePath '%s' is not mounted",
		Type:        codes.FailedPrecondition,
		Action:      "Please check if there is any error in POD describe related with volume attach",
	},
}

// InitMessages ...
func InitMessages() map[string]Message {
	return messagesEn
}

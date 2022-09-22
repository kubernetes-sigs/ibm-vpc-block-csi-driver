/**
 * Copyright 2020 IBM Corp.
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
	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
)

// messagesEn ...
var messagesEn = map[string]util.Message{
	"AuthenticationFailed": {
		Code:        AuthenticationFailed,
		Description: "Failed to authenticate the user.",
		Type:        util.Unauthenticated,
		RC:          400,
		Action:      "Verify that you entered the correct IBM Cloud user name and password. If the error persists, the authentication service might be unavailable. Wait a few minutes and try again. ",
	},
	"EndpointNotReachable": {
		Code:        "EndpointNotReachable",
		Description: "IAM TOKEN exchange request failed.",
		Type:        util.FailedAccessToken,
		RC:          500,
		Action:      "Verify that iks_token_exchange_endpoint_private_url is reachable from the cluster. You can find this url by running 'kubectl get secret storage-secret-storage -n kube-system'.",
	},
	"Timeout": {
		Code:        "Timeout",
		Description: "IAM Token exchange endpoint is not reachable.",
		Type:        util.FailedAccessToken,
		RC:          503,
		Action:      "Wait for a few mninutes and try again. If the error persists user can open a container network issue.",
	},
	"ErrorRequiredFieldMissing": {
		Code:        "ErrorRequiredFieldMissing",
		Description: "[%s] is required to complete the operation.",
		Type:        util.InvalidRequest,
		RC:          400,
		Action:      "Review the error that is returned. Provide the missing information in your request and try again. ",
	},
	"FailedToPlaceOrder": {
		Code:        "FailedToPlaceOrder",
		Description: "Failed to create volume with the storage provider",
		Type:        util.ProvisioningFailed,
		RC:          500,
		Action:      "Review the error that is returned. If the volume creation service is currently unavailable, try to manually create the volume with the 'ibmcloud is volume-create' command.",
	},
	"FailedToDeleteVolume": {
		Code:        "FailedToDeleteVolume",
		Description: "The volume ID '%d' could not be deleted from your VPC.",
		Type:        util.DeletionFailed,
		RC:          500,
		Action:      "Verify that the volume ID exists. Run 'ibmcloud is volumes' to list available volumes in your account. If the ID is correct, try to delete the volume with the 'ibmcloud is volume-delete' command. ",
	},
	"FailedToExpandVolume": {
		Code:        "FailedToExpandVolume",
		Description: "The volume ID '%s' could not be expanded from VPC.",
		Type:        util.ExpansionFailed,
		RC:          500,
		Action:      "Verify that the volume ID exists and attached to an instance. Run 'ibmcloud is volumes' to list available volumes in your account. If the ID is correct, check that expected capacity is valid and supported",
	},
	"FailedToUpdateVolume": {
		Code:        "FailedToUpdateVolume",
		Description: "The volume ID '%d' could not be updated",
		Type:        util.UpdateFailed,
		RC:          500,
		Action:      "Verify that the volume ID exists. Run 'ibmcloud is volumes' to list available volumes in your account.",
	},
	"FailedToDeleteSnapshot": {
		Code:        "FailedToDeleteSnapshot",
		Description: "Failed to delete '%d' snapshot ID",
		Type:        util.DeletionFailed,
		RC:          500,
		Action:      "Check whether the snapshot ID exists. You may need to verify by using 'ibmcloud is' cli",
	},
	"StorageFindFailedWithVolumeId": {
		Code:        "StorageFindFailedWithVolumeId",
		Description: "A volume with the specified volume ID '%s' could not be found.",
		Type:        util.RetrivalFailed,
		RC:          404,
		Action:      "Verify that the volume ID exists. Run 'ibmcloud is volumes' to list available volumes in your account.",
	},
	"StorageFindFailedWithVolumeName": {
		Code:        "StorageFindFailedWithVolumeName",
		Description: "A volume with the specified volume name '%s' does not exist.",
		Type:        util.RetrivalFailed,
		RC:          404,
		Action:      "Verify that the specified volume exists. Run 'ibmcloud is volumes' to list available volumes in your account.",
	},
	"SnapshotIDNotFound": {
		Code:        "SnapshotIDNotFound",
		Description: "A snapshot with the specified snapshot ID '%s' could not be found.",
		Type:        util.RetrivalFailed,
		RC:          404,
		Action:      "Please check the snapshot ID once, You many need to verify by using 'ibmcloud is' cli.",
	},
	"StorageFindFailedWithSnapshotName": {
		Code:        "StorageFindFailedWithSnapshotName",
		Description: "A snapshot with the specified snapshot name '%s' could not be found.",
		Type:        util.RetrivalFailed,
		RC:          404,
		Action:      "Please check the snapshot name once, You many need to verify by using 'ibmcloud is' cli.",
	},
	"VolumeAttachFindFailed": {
		Code:        VolumeAttachFindFailed,
		Description: "No volume attachment could be found for the specified volume ID '%s' and instance ID '%s'.",
		Type:        util.VolumeAttachFindFailed,
		RC:          400,
		Action:      "Verify that a volume attachment for your instance exists. Run 'ibmcloud is in-vols INSTANCE_ID' to list active volume attachments for your instance ID. ",
	},
	"VolumeAttachFailed": {
		Code:        VolumeAttachFailed,
		Description: "The volume ID '%s' could not be attached to the instance ID '%s'.",
		Type:        util.AttachFailed,
		RC:          500,
		Action:      "Verify that the volume ID and instance ID exist. Run 'ibmcloud is volumes' to list available volumes, and 'ibmcloud is instances' to list available instances in your account. ",
	},
	"VolumeAttachTimedOut": {
		Code:        VolumeAttachTimedOut,
		Description: "The volume ID '%s' could not be attached to the instance ID '%s'",
		Type:        util.AttachFailed,
		RC:          500,
		Action:      "Verify that the volume ID and instance ID exist. Run 'ibmcloud is volumes' to list available volumes, and 'ibmcloud is instances' to list available instances in your account.",
	},
	"VolumeDetachFailed": {
		Code:        VolumeDetachFailed,
		Description: "The volumd ID '%s' could not be detached from the instance ID '%s'.",
		Type:        util.DetachFailed,
		RC:          500,
		Action:      "Verify that the specified instance ID has active volume attachments. Run 'ibmcloud is in-vols INSTANCE_ID' to list active volume attachments for your instance ID. ",
	},
	"VolumeDetachTimedOut": {
		Code:        VolumeDetachTimedOut,
		Description: "The volume ID '%s' could not be detached from the instance ID '%s'",
		Type:        util.DetachFailed,
		RC:          500,
		Action:      "Verify that the specified instance ID has active volume attachments. Run 'ibmcloud is in-vols INSTANCE_ID' to list active volume attachments for your instance ID.",
	},
	"InvalidVolumeID": {
		Code:        "InvalidVolumeID",
		Description: "The specified volume ID '%s' is not valid.",
		Type:        util.InvalidRequest,
		RC:          400,
		Action:      "Verify that the volume ID exists. Run 'ibmcloud is volumes' to list available volumes in your account.",
	},
	"InvalidVolumeName": {
		Code:        "InvalidVolumeName",
		Description: "The specified volume name '%s' is not valid. ",
		Type:        util.InvalidRequest,
		RC:          400,
		Action:      "Verify that the volume name exists. Run 'ibmcloud is volumes' to list available volumes in your account.",
	},
	"VolumeCapacityInvalid": {
		Code:        "VolumeCapacityInvalid",
		Description: "The specified volume capacity '%d' is not valid. ",
		Type:        util.InvalidRequest,
		RC:          400,
		Action:      "Verify the specified volume capacity. The volume capacity must be a positive number between 10 GB and 2000 GB. ",
	},
	"IopsInvalid": {
		Code:        "IopsInvalid",
		Description: "The specified volume IOPS '%s' is not valid for the selected volume profile. ",
		Type:        util.InvalidRequest,
		RC:          400,
		Action:      "Review available volume profiles and IOPS in the IBM Cloud Block Storage for VPC documentation https://cloud.ibm.com/docs/vpc-on-classic-block-storage?topic=vpc-on-classic-block-storage-block-storage-profiles.",
	},
	"VolumeProfileIopsInvalid": {
		Code:        "VolumeProfileIopsInvalid",
		Description: "The specified IOPS value is not valid for the selected volume profile. ",
		Type:        util.InvalidRequest,
		RC:          400,
		Action:      "Review available volume profiles and IOPS in the IBM Cloud Block Storage for VPC documentation https://cloud.ibm.com/docs/vpc-on-classic-block-storage?topic=vpc-on-classic-block-storage-block-storage-profiles.",
	},
	"VolumeProfileEmpty": {
		Code:        "VolumeProfileEmpty",
		Description: "Volume profile is empty, you need to pass valid profile name.",
		Type:        util.InvalidRequest,
		RC:          400,
		Action:      "Review storage class used to create volume and add valid profile parameter.",
	},
	"EmptyResourceGroup": {
		Code:        "EmptyResourceGroup",
		Description: "Resource group information could not be found.",
		Type:        util.InvalidRequest,
		RC:          400,
		Action:      "Provide the name or ID of the resource group that you want to use for your volume. Run 'ibmcloud resource groups' to list the resource groups that you have access to. ",
	},
	"EmptyResourceGroupIDandName": {
		Code:        "EmptyResourceGroupIDandName",
		Description: "Resource group ID or name could not be found.",
		Type:        util.InvalidRequest,
		RC:          400,
		Action:      "Provide the name or ID of the resource group that you want to use for your volume. Run 'ibmcloud resource groups' to list the resource groups that you have access to.",
	},
	"SnapshotSpaceOrderFailed": {
		Code:        "SnapshotSpaceOrderFailed",
		Description: "Snapshot space order failed for the given volume ID",
		Type:        util.ProvisioningFailed,
		RC:          500,
		Action:      "Please check your input",
	},
	"VolumeNotInValidState": {
		Code:        "VolumeNotInValidState",
		Description: "Volume %s did not get valid (available) status within timeout period.",
		Type:        util.ProvisioningFailed,
		RC:          500,
		Action:      "Please check your input",
	},
	"VolumeDeletionInProgress": {
		Code:        "VolumeDeletionInProgress",
		Description: "Volume %s deletion in progress.",
		Type:        util.ProvisioningFailed,
		RC:          500,
		Action:      "Wait for volume deletion",
	},
	"ListSnapshotsFailed": {
		Code:        "ListSnapshotsFailed",
		Description: "Unable to fetch list of volumes.",
		Type:        util.RetrivalFailed,
		RC:          404,
		Action:      "Run 'ibmcloud is snapshots' to list available snapshots in your account.",
	},
	"ListVolumesFailed": {
		Code:        "ListVolumesFailed",
		Description: "Unable to fetch list of volumes.",
		Type:        util.RetrivalFailed,
		RC:          404,
		Action:      "Run 'ibmcloud is volumes' to list available volumes in your account.",
	},
	"InvalidListVolumesLimit": {
		Code:        "InvalidListVolumesLimit",
		Description: "The value '%v' specified in the limit parameter of the list volume call is not valid.",
		Type:        util.InvalidRequest,
		RC:          400,
		Action:      "Verify the limit parameter's value. The limit must be a positive number between 0 and 100.",
	},
	"InvalidListSnapshotLimit": {
		Code:        "InvalidListSnapshotLimit",
		Description: "The value '%v' specified in the limit parameter of the list snapshot call is not valid.",
		Type:        util.InvalidRequest,
		RC:          400,
		Action:      "Verify the limit parameter's value. The limit must be a positive number between 0 and 100.",
	},
	"StartVolumeIDNotFound": {
		Code:        "StartVolumeIDNotFound",
		Description: "The volume ID '%s' specified in the start parameter of the list volume call could not be found.",
		Type:        util.InvalidRequest,
		RC:          400,
		Action:      "Please verify that the start volume ID is correct and whether you have access to the volume ID.",
	},
	"StartSnapshotIDNotFound": {
		Code:        "StartSnapshotIDNotFound",
		Description: "The snapshot ID '%s' specified in the start parameter of the list volume call could not be found.",
		Type:        util.InvalidRequest,
		RC:          400,
		Action:      "Please verify that the start snapshot ID is correct and whether you have access to the snapshot ID.",
	},
	"InvalidServiceSession": {
		Code:        "InvalidServiceSession",
		Description: "The Service Session was not found due to error while generating IAM token.",
		Type:        util.RetrivalFailed,
		RC:          500,
		Action:      "Please retry again after some time.",
	},
}

// InitMessages ...
func InitMessages() map[string]util.Message {
	return messagesEn
}

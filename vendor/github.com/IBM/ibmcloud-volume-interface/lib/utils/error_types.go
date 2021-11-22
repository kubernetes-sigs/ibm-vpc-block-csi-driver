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

// Package util ...
package util

// These are the error types which all provider should categorize their errors
const (
	// FailedAccessToken ...
	FailedAccessToken = "FailedAccessToken"

	// ProvisioningFailed volume or snapshot provisioning failed
	ProvisioningFailed = "ProvisioningFailed"

	// DeletionFailed ...
	DeletionFailed = "DeletionFailed"

	// UpdateFailed ...
	UpdateFailed = "UpdateFailed"

	// RetrivalFailed ...
	RetrivalFailed = "RetrivalFailed"

	// InvalidRequest ...
	InvalidRequest = "InvalidRequest"

	// EntityNotFound ...
	EntityNotFound = "EntityNotFound"

	// PermissionDenied ...
	PermissionDenied = "PermissionDenied"

	// Unauthenticated ...
	Unauthenticated = "Unauthenticated"

	// ErrorTypeFailed ...
	ErrorTypeFailed = "ErrorTypeConversionFailed"

	// VolumeAttachFindFailed ...
	VolumeAttachFindFailed = "VolumeAttachFindFailed"

	// AttachFailed ...
	AttachFailed = "AttachFailed"

	// VolumeAccessPointFindFailed ...
	VolumeAccessPointFindFailed = "VolumeAccessPointFindFailed"

	// CreateVolumeAccessPointFailed ...
	CreateVolumeAccessPointFailed = "CreateVolumeAccessPointFailed"

	// DeleteVolumeAccessPointFailed ...
	DeleteVolumeAccessPointFailed = "DeleteVolumeAccessPointFailed"

	// InstanceNotFound ...
	NodeNotFound = "NodeNotFound"

	// DetachFailed ...
	DetachFailed = "DetachFailed"

	// ExpansionFailed ...
	ExpansionFailed = "ExpansionFailed"
)

// GetErrorType return the user error type provided by volume provider
func GetErrorType(err error) string {
	providerError, ok := err.(Message)
	if ok {
		return providerError.Type
	}
	return ErrorTypeFailed
}

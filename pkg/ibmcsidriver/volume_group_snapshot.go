/*
Copyright 2026 The Kubernetes Authors.

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

package ibmcsidriver

import (
	"fmt"
	"strings"

	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	providerError "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// groupSnapshotSourceVolumeIDs returns complete source-volume membership.
// The boolean is false when backend member snapshot details are not yet available.
func groupSnapshotSourceVolumeIDs(groupSnapshot *provider.GroupSnapshot) ([]string, bool) {
	if groupSnapshot == nil {
		return nil, false
	}

	existingVolumeIDs := make([]string, 0, len(groupSnapshot.Snapshots))
	for _, snapshot := range groupSnapshot.Snapshots {
		if snapshot == nil || snapshot.VolumeID == "" {
			return nil, false
		}
		existingVolumeIDs = append(existingVolumeIDs, snapshot.VolumeID)
	}
	if len(existingVolumeIDs) == 0 {
		return nil, false
	}
	return existingVolumeIDs, true
}

// groupSnapshotMemberIDs returns the CSI-facing member identifiers, preferring
// CRNs because those are exposed as SnapshotId in group snapshot responses.
func groupSnapshotMemberIDs(groupSnapshot *provider.GroupSnapshot) ([]string, bool) {
	if groupSnapshot == nil {
		return nil, false
	}

	snapshotIDs := make([]string, 0, len(groupSnapshot.Snapshots))
	for _, snapshot := range groupSnapshot.Snapshots {
		if snapshot == nil {
			return nil, false
		}
		snapshotID := snapshot.SnapshotCRN
		if snapshotID == "" {
			snapshotID = snapshot.SnapshotID
		}
		if snapshotID == "" {
			return nil, false
		}
		snapshotIDs = append(snapshotIDs, snapshotID)
	}
	if len(snapshotIDs) == 0 {
		return nil, false
	}
	return snapshotIDs, true
}

// equalStringSets compares unordered CSI identifier lists while preserving
// duplicate counts.
func equalStringSets(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	counts := make(map[string]int, len(left))
	for _, value := range left {
		counts[value]++
	}
	for _, value := range right {
		if counts[value] == 0 {
			return false
		}
		counts[value]--
	}
	return true
}

// isVolumeGroupSnapshotNotFoundError identifies the VPC not-found response even
// when volume-vpc wraps it as a group snapshot deletion failure.
func isVolumeGroupSnapshotNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	errorText := strings.ToLower(strings.ReplaceAll(err.Error(), " ", ""))
	return strings.Contains(errorText, "snapshot_consistency_groups_not_found") ||
		strings.Contains(errorText, "snapshots_not_found") ||
		providerError.GetErrorType(err) == providerError.EntityNotFound
}

// volumeGroupSnapshotStatusError builds consistent VGS errors and includes the
// request ID so an error can be correlated with controller logs.
func volumeGroupSnapshotStatusError(code codes.Code, requestID, message string, args ...interface{}) error {
	description := fmt.Sprintf(message, args...)
	return status.Errorf(code, "%s (request ID: %s)", description, requestID)
}

// volumeGroupSnapshotCSIError converts known VPC failures into actionable CSI
// status codes and preserves the request ID in the returned error.
func volumeGroupSnapshotCSIError(logger *zap.Logger, requestID, operation string, err error) error {
	code := volumeGroupSnapshotErrorCode(err)
	logger.Error("Volume group snapshot backend request failed",
		zap.String("operation", operation),
		zap.String("grpcCode", code.String()),
		zap.Error(err))
	return status.Errorf(code, "failed to %s volume group snapshot (request ID: %s): %v", operation, requestID, err)
}

// volumeGroupSnapshotErrorCode maps backend reason codes first, then falls back
// to the generic provider error category.
func volumeGroupSnapshotErrorCode(err error) codes.Code {
	if err == nil {
		return codes.OK
	}

	errorText := strings.ToLower(strings.ReplaceAll(err.Error(), " ", ""))
	switch {
	case strings.Contains(errorText, "snapshots_source_volume_not_found"):
		return codes.NotFound
	case strings.Contains(errorText, "snapshot_consistency_groups_not_found"),
		strings.Contains(errorText, "snapshots_not_found"):
		return codes.NotFound
	case strings.Contains(errorText, "snapshots_source_volume_not_attached"):
		return codes.FailedPrecondition
	case strings.Contains(errorText, "snapshots_source_volume_busy"):
		return codes.Aborted
	case strings.Contains(errorText, "snapshots_service_unavailable"):
		return codes.Unavailable
	}
	switch {
	case strings.Contains(errorText, "rc:400"):
		return codes.InvalidArgument
	case strings.Contains(errorText, "rc:401"):
		return codes.Unauthenticated
	case strings.Contains(errorText, "rc:403"):
		return codes.PermissionDenied
	case strings.Contains(errorText, "rc:404"):
		return codes.NotFound
	case strings.Contains(errorText, "rc:409"):
		return codes.FailedPrecondition
	case strings.Contains(errorText, "rc:429"):
		return codes.ResourceExhausted
	case strings.Contains(errorText, "rc:5"):
		return codes.Unavailable
	}

	switch providerError.GetErrorType(err) {
	case providerError.EntityNotFound:
		return codes.NotFound
	case providerError.InvalidRequest:
		return codes.InvalidArgument
	case providerError.PermissionDenied:
		return codes.PermissionDenied
	case providerError.Unauthenticated, providerError.FailedAccessToken:
		return codes.Unauthenticated
	}

	return codes.Internal
}

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

	commonError "github.com/IBM/ibm-csi-common/pkg/messages"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	providerError "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	utilReasonCode "github.com/IBM/ibmcloud-volume-interface/lib/utils/reasoncode"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// validateGroupSnapshotSourceVolumes checks if members from req and resp match
func validateGroupSnapshotSourceVolumes(snapshotName, requestID string, snapshots []*provider.Snapshot, sourceVolumeIDs []string, mismatchCode codes.Code) error {
	memberVolumeIDs := make([]string, 0, len(snapshots))
	for _, snapshot := range snapshots {
		memberVolumeIDs = append(memberVolumeIDs, snapshot.VolumeID)
	}

	if matchIDs(memberVolumeIDs, sourceVolumeIDs) {
		return nil
	}

	if mismatchCode == codes.AlreadyExists {
		return volumeGroupSnapshotStatusError(codes.AlreadyExists, requestID, "volume group snapshot %q already exists with a different set of source volume IDs (requested %v, existing %v); create a new VolumeGroupSnapshot or delete and recreate the requested VolumeGroupSnapshot to generate a new snapshot name", snapshotName, sourceVolumeIDs, memberVolumeIDs)
	}

	// comes here after a VGS gets created, but VPC returned snapshots of different volumes
	return volumeGroupSnapshotStatusError(codes.Internal, requestID, "backend returned unexpected source volumes for newly created volume group snapshot %q (requested %v, returned %v)", snapshotName, sourceVolumeIDs, memberVolumeIDs)
}

// resolveGroupSnapshotMemberCRNs returns the CRN for each member snapshot. The boolean is false when any member has no CRN.
func resolveGroupSnapshotMemberCRNs(snapshots []*provider.Snapshot) ([]string, bool) {
	if len(snapshots) == 0 {
		return nil, false
	}

	crns := make([]string, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot == nil || snapshot.SnapshotCRN == "" {
			return nil, false
		}
		crns = append(crns, snapshot.SnapshotCRN)
	}

	return crns, true
}

// matchIDs reports whether left and right contain the same IDs regardless of order, preserving duplicate counts.
func matchIDs(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	counts := make(map[string]int, len(left))
	for _, id := range left {
		counts[id]++
	}
	for _, id := range right {
		if counts[id] == 0 {
			return false
		}
		counts[id]--
	}
	return true
}

// isGroupSnapshotNotFound reports whether the error indicates the consistency group does not exist on the VPC backend.
func isGroupSnapshotNotFound(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "snapshot_consistency_groups_not_found") ||
		providerError.GetErrorType(err) == providerError.EntityNotFound
}

// volumeGroupSnapshotStatusError builds consistent VGS errors and includes the
// request ID so an error can be correlated with controller logs.
func volumeGroupSnapshotStatusError(code codes.Code, requestID, message string, args ...interface{}) error {
	description := fmt.Sprintf(message, args...)
	return status.Errorf(code, "%s (request ID: %s)", description, requestID)
}

// providerSessionCSIError maps a GetProviderSession failure to the CSI error:
// unreachable endpoint, timeout, or internal error for everything else.
func providerSessionCSIError(logger *zap.Logger, requestID string, err error) error {
	switch userError.GetUserErrorCode(err) {
	case string(utilReasonCode.EndpointNotReachable):
		return commonError.GetCSIError(logger, commonError.EndpointNotReachable, requestID, err)
	case string(utilReasonCode.Timeout):
		return commonError.GetCSIError(logger, commonError.Timeout, requestID, err)
	}
	return commonError.GetCSIError(logger, commonError.InternalError, requestID, err)
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
	if userError.GetUserErrorCode(err) == "GroupSnapshotMemberDetailsUnavailable" {
		return codes.Aborted
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

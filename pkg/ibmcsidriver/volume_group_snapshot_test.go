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
	"errors"
	"strings"
	"testing"

	commonError "github.com/IBM/ibm-csi-common/pkg/messages"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	providerError "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	utilReasonCode "github.com/IBM/ibmcloud-volume-interface/lib/utils/reasoncode"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestVolumeGroupSnapshotStatusErrorIncludesMessageAndRequestID(t *testing.T) {
	err := volumeGroupSnapshotStatusError(codes.InvalidArgument, "request-id", "volume group snapshot %q was not found", "group-id")

	message := err.Error()
	assert.True(t, strings.Contains(message, `volume group snapshot "group-id" was not found`))
	assert.True(t, strings.Contains(message, "request ID: request-id"))
}

// Member resolution uses CRNs only and never accepts a partial member list.
func TestResolveGroupSnapshotMemberCRNs(t *testing.T) {
	testCases := []struct {
		name         string
		snapshots    []*provider.Snapshot
		expectedCRNs []string
		expectedOK   bool
	}{
		{name: "nil members"},
		{name: "empty members", snapshots: []*provider.Snapshot{}},
		{
			name:      "nil member after a valid member",
			snapshots: []*provider.Snapshot{{SnapshotCRN: "snapshot-crn-1"}, nil},
		},
		{
			name:      "missing CRN after a valid member",
			snapshots: []*provider.Snapshot{{SnapshotCRN: "snapshot-crn-1"}, {VolumeID: "volume-2"}},
		},
		{
			name:      "short snapshot ID without CRN",
			snapshots: []*provider.Snapshot{{SnapshotID: "snapshot-id-1", VolumeID: "volume-1"}},
		},
		{
			name:         "complete CRNs",
			snapshots:    []*provider.Snapshot{{SnapshotCRN: "snapshot-crn-2"}, {SnapshotCRN: "snapshot-crn-1"}},
			expectedCRNs: []string{"snapshot-crn-2", "snapshot-crn-1"},
			expectedOK:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			crns, ok := resolveGroupSnapshotMemberCRNs(tc.snapshots)
			assert.Equal(t, tc.expectedOK, ok)
			assert.Equal(t, tc.expectedCRNs, crns)
		})
	}
}

// A failed GetProviderSession call is mapped by its provider reason code.
// Anything that is not a known reason code, including a plain error, becomes Internal.
func TestProviderSessionCSIError(t *testing.T) {
	// GetCSIError looks messages up here; without it the status code would be OK and no error is returned.
	commonError.MessagesEn = commonError.InitMessages()

	testCases := []struct {
		name         string
		err          error
		expectedCode codes.Code
		expectedText string // CSI reason code carried in the message
	}{
		{
			name:         "endpoint not reachable",
			err:          providerError.Message{Code: string(utilReasonCode.EndpointNotReachable)},
			expectedCode: codes.Unavailable,
			expectedText: commonError.EndpointNotReachable,
		},
		{
			name:         "timeout",
			err:          providerError.Message{Code: string(utilReasonCode.Timeout)},
			expectedCode: codes.DeadlineExceeded,
			expectedText: commonError.Timeout,
		},
		{
			name:         "unknown reason code",
			err:          providerError.Message{Code: "SomethingElse"},
			expectedCode: codes.Internal,
			expectedText: commonError.InternalError,
		},
		{
			name:         "provider error without a reason code",
			err:          providerError.Message{},
			expectedCode: codes.Internal,
			expectedText: commonError.InternalError,
		},
		{
			name:         "plain error",
			err:          errors.New("connection refused"),
			expectedCode: codes.Internal,
			expectedText: commonError.InternalError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := providerSessionCSIError(zap.NewNop(), "request-id", tc.err)

			st, ok := status.FromError(err)
			assert.True(t, ok, "must be a gRPC status error")
			assert.Equal(t, tc.expectedCode, st.Code())
			assert.Contains(t, st.Message(), "Code: "+tc.expectedText)
			assert.Contains(t, st.Message(), "request-id")
			assert.Contains(t, st.Message(), tc.err.Error(), "the original error must be kept in the message")
		})
	}
}

func TestIsGroupSnapshotNotFound(t *testing.T) {
	groupNotFoundErr := providerError.Message{
		Type:         providerError.DeletionFailed,
		BackendError: "Code:snapshot_consistency_groups_not_found, RC:404",
	}
	membersNotFoundErr := providerError.Message{
		Type:         providerError.DeletionFailed,
		BackendError: "Code:snapshots_not_found, RC:404",
	}

	assert.False(t, isGroupSnapshotNotFound(nil))
	assert.True(t, isGroupSnapshotNotFound(groupNotFoundErr))
	assert.True(t, isGroupSnapshotNotFound(providerError.Message{Type: providerError.EntityNotFound}))
	assert.False(t, isGroupSnapshotNotFound(membersNotFoundErr))
	assert.False(t, isGroupSnapshotNotFound(providerError.Message{Type: providerError.DeletionFailed}))
}

// Source validation must distinguish a name conflict from an invalid new backend result.
func TestValidateGroupSnapshotSourceVolumes(t *testing.T) {
	testCases := []struct {
		name         string
		snapshots    []*provider.Snapshot
		mismatchCode codes.Code
		wantCode     codes.Code
	}{
		{
			name:         "matching sources in a different order",
			snapshots:    []*provider.Snapshot{{VolumeID: "volume-b"}, {VolumeID: "volume-a"}},
			mismatchCode: codes.AlreadyExists, wantCode: codes.OK,
		},
		{
			name:         "existing group contains different sources",
			snapshots:    []*provider.Snapshot{{VolumeID: "volume-a"}, {VolumeID: "volume-c"}},
			mismatchCode: codes.AlreadyExists, wantCode: codes.AlreadyExists,
		},
		{
			name:         "new group contains unexpected sources",
			snapshots:    []*provider.Snapshot{{VolumeID: "volume-a"}, {VolumeID: "volume-c"}},
			mismatchCode: codes.Internal, wantCode: codes.Internal,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateGroupSnapshotSourceVolumes("group-name", "request-id", tc.snapshots, []string{"volume-a", "volume-b"}, tc.mismatchCode)
			assert.Equal(t, tc.wantCode, status.Code(err))
			if err != nil {
				assert.Contains(t, err.Error(), "group-name")
				assert.Contains(t, err.Error(), "request-id")
			}
			if tc.wantCode == codes.AlreadyExists {
				assert.Contains(t, err.Error(), "delete and recreate the requested VolumeGroupSnapshot")
			}
		})
	}
}

func TestVolumeGroupSnapshotErrorCode(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected codes.Code
	}{
		{
			name: "provider member source-volume details are incomplete",
			err: providerError.Message{
				Code: "GroupSnapshotMemberDetailsUnavailable",
				Type: providerError.RetrivalFailed,
				RC:   409,
			},
			expected: codes.Aborted,
		},
		{
			name: "member lookup permission denied",
			err: providerError.Message{
				Code:         "GroupSnapshotMemberLookupFailed",
				Type:         providerError.RetrivalFailed,
				RC:           500,
				BackendError: "Trace Code:member-lookup-trace, Code:snapshots_not_authorized, RC:403 Forbidden",
			},
			expected: codes.PermissionDenied,
		},
		{
			name: "member lookup authentication failed",
			err: providerError.Message{
				Code:         "GroupSnapshotMemberLookupFailed",
				Type:         providerError.RetrivalFailed,
				RC:           500,
				BackendError: "Trace Code:member-lookup-trace, Code:token_invalid, RC:401 Unauthorized",
			},
			expected: codes.Unauthenticated,
		},
		{
			name: "member lookup rate limited",
			err: providerError.Message{
				Code:         "GroupSnapshotMemberLookupFailed",
				Type:         providerError.RetrivalFailed,
				RC:           500,
				BackendError: "Trace Code:member-lookup-trace, Code:snapshots_too_many_requests, RC:429 Too Many Requests",
			},
			expected: codes.ResourceExhausted,
		},
		{
			name: "member lookup internal backend error",
			err: providerError.Message{
				Code:         "GroupSnapshotMemberLookupFailed",
				Type:         providerError.RetrivalFailed,
				RC:           500,
				BackendError: "Trace Code:member-lookup-trace, Code:internal_error, RC:500 Internal Server Error",
			},
			expected: codes.Unavailable,
		},
		{
			name: "source volume is not attached",
			err: providerError.Message{
				Type:         providerError.ProvisioningFailed,
				BackendError: "Code:snapshots_source_volume_not_attached, RC:409",
			},
			expected: codes.FailedPrecondition,
		},
		{
			name: "source volume not found",
			err: providerError.Message{
				Type:         providerError.ProvisioningFailed,
				BackendError: "Code:snapshots_source_volume_not_found, RC:404",
			},
			expected: codes.NotFound,
		},
		{
			name: "source volume busy",
			err: providerError.Message{
				Type:         providerError.ProvisioningFailed,
				BackendError: "Code:snapshots_source_volume_busy, RC:409",
			},
			expected: codes.Aborted,
		},
		{
			name: "snapshot service unavailable",
			err: providerError.Message{
				Type:         providerError.ProvisioningFailed,
				BackendError: "Code:snapshots_service_unavailable, RC:503",
			},
			expected: codes.Unavailable,
		},
		{
			name: "generic backend conflict",
			err: providerError.Message{
				Type:         providerError.DeletionFailed,
				BackendError: "Code:invalid_state, RC:409",
			},
			expected: codes.FailedPrecondition,
		},
		{
			name: "backend capacity exhausted",
			err: providerError.Message{
				Type:         providerError.ProvisioningFailed,
				BackendError: "Code:limit_reached, RC:429",
			},
			expected: codes.ResourceExhausted,
		},
		{
			name:     "invalid backend request",
			err:      providerError.Message{Type: providerError.InvalidRequest},
			expected: codes.InvalidArgument,
		},
		{
			name:     "permission denied",
			err:      providerError.Message{Type: providerError.PermissionDenied},
			expected: codes.PermissionDenied,
		},
		{
			name:     "unknown provider error",
			err:      providerError.Message{Type: providerError.ProvisioningFailed},
			expected: codes.Internal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, volumeGroupSnapshotErrorCode(tc.err))
		})
	}
}

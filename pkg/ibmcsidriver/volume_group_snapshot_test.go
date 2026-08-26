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
	"strings"
	"testing"

	providerError "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestVolumeGroupSnapshotStatusErrorIncludesMessageAndRequestID(t *testing.T) {
	err := volumeGroupSnapshotStatusError(codes.InvalidArgument, "request-id", "volume group snapshot %q was not found", "group-id")

	message := err.Error()
	assert.True(t, strings.Contains(message, `volume group snapshot "group-id" was not found`))
	assert.True(t, strings.Contains(message, "request ID: request-id"))
}

func TestIsVolumeGroupSnapshotNotFoundError(t *testing.T) {
	groupNotFoundErr := providerError.Message{
		Type:         providerError.DeletionFailed,
		BackendError: "Code:snapshot_consistency_groups_not_found, RC:404",
	}
	membersNotFoundErr := providerError.Message{
		Type:         providerError.DeletionFailed,
		BackendError: "Code:snapshots_not_found, RC:404",
	}

	assert.True(t, isVolumeGroupSnapshotNotFoundError(groupNotFoundErr))
	assert.True(t, isVolumeGroupSnapshotNotFoundError(membersNotFoundErr))
	assert.False(t, isVolumeGroupSnapshotNotFoundError(providerError.Message{Type: providerError.DeletionFailed}))
}

func TestVolumeGroupSnapshotErrorCode(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected codes.Code
	}{
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

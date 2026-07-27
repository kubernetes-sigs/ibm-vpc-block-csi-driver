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

package sanity

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	providerError "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/google/uuid"
	sanityUtils "github.com/kubernetes-csi/csi-test/v4/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type fakeGroupSnapshot struct {
	*provider.GroupSnapshot
	name            string
	resourceGroupID string
}

func TestVolumeGroupSnapshotSanity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping volume group snapshot sanity testing...")
	}

	client := newGroupControllerSanityClient(t)

	t.Run("advertises group snapshot capability", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		response, err := client.GroupControllerGetCapabilities(ctx, &csi.GroupControllerGetCapabilitiesRequest{})
		if err != nil {
			t.Fatalf("GroupControllerGetCapabilities failed: %v", err)
		}
		if len(response.Capabilities) != 1 {
			t.Fatalf("expected one group controller capability, got %d", len(response.Capabilities))
		}
		got := response.Capabilities[0].GetRpc().GetType()
		want := csi.GroupControllerServiceCapability_RPC_CREATE_DELETE_GET_VOLUME_GROUP_SNAPSHOT
		if got != want {
			t.Fatalf("expected capability %s, got %s", want, got)
		}
	})

	t.Run("rejects missing required fields", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := client.CreateVolumeGroupSnapshot(ctx, &csi.CreateVolumeGroupSnapshotRequest{
			SourceVolumeIds: []string{"volume-1"},
		})
		requireRPCCode(t, err, codes.InvalidArgument)

		_, err = client.CreateVolumeGroupSnapshot(ctx, &csi.CreateVolumeGroupSnapshotRequest{
			Name: "group-without-volumes",
		})
		requireRPCCode(t, err, codes.InvalidArgument)

		_, err = client.GetVolumeGroupSnapshot(ctx, &csi.GetVolumeGroupSnapshotRequest{})
		requireRPCCode(t, err, codes.InvalidArgument)

		_, err = client.DeleteVolumeGroupSnapshot(ctx, &csi.DeleteVolumeGroupSnapshotRequest{})
		requireRPCCode(t, err, codes.InvalidArgument)

		_, err = client.DeleteVolumeGroupSnapshot(ctx, &csi.DeleteVolumeGroupSnapshotRequest{
			GroupSnapshotId: "missing-group",
		})
		requireRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("creates gets and deletes a group snapshot", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		request := &csi.CreateVolumeGroupSnapshotRequest{
			Name:            "sanity-group-snapshot",
			SourceVolumeIds: []string{"volume-1", "volume-2"},
		}
		created, err := client.CreateVolumeGroupSnapshot(ctx, request)
		if err != nil {
			t.Fatalf("CreateVolumeGroupSnapshot failed: %v", err)
		}
		group := created.GetGroupSnapshot()
		if group == nil {
			t.Fatal("CreateVolumeGroupSnapshot returned no group snapshot")
		}
		if group.GroupSnapshotId == "" {
			t.Fatal("CreateVolumeGroupSnapshot returned an empty group snapshot ID")
		}
		if !group.ReadyToUse {
			t.Fatal("expected created group snapshot to be ready to use")
		}
		if len(group.Snapshots) != len(request.SourceVolumeIds) {
			t.Fatalf("expected %d member snapshots, got %d", len(request.SourceVolumeIds), len(group.Snapshots))
		}

		snapshotIDs := make([]string, 0, len(group.Snapshots))
		for i, snapshot := range group.Snapshots {
			if snapshot.SnapshotId == "" {
				t.Fatalf("member snapshot %d has an empty snapshot ID", i)
			}
			if snapshot.SourceVolumeId != request.SourceVolumeIds[i] {
				t.Fatalf("member snapshot %d has source volume %q, want %q", i, snapshot.SourceVolumeId, request.SourceVolumeIds[i])
			}
			if snapshot.GroupSnapshotId != group.GroupSnapshotId {
				t.Fatalf("member snapshot %d has group ID %q, want %q", i, snapshot.GroupSnapshotId, group.GroupSnapshotId)
			}
			snapshotIDs = append(snapshotIDs, snapshot.SnapshotId)
		}

		repeated, err := client.CreateVolumeGroupSnapshot(ctx, request)
		if err != nil {
			t.Fatalf("repeated CreateVolumeGroupSnapshot failed: %v", err)
		}
		if repeated.GetGroupSnapshot().GetGroupSnapshotId() != group.GroupSnapshotId {
			t.Fatalf("repeated create returned group ID %q, want %q", repeated.GetGroupSnapshot().GetGroupSnapshotId(), group.GroupSnapshotId)
		}

		fetched, err := client.GetVolumeGroupSnapshot(ctx, &csi.GetVolumeGroupSnapshotRequest{
			GroupSnapshotId: group.GroupSnapshotId,
		})
		if err != nil {
			t.Fatalf("GetVolumeGroupSnapshot failed: %v", err)
		}
		if fetched.GetGroupSnapshot().GetGroupSnapshotId() != group.GroupSnapshotId {
			t.Fatalf("get returned group ID %q, want %q", fetched.GetGroupSnapshot().GetGroupSnapshotId(), group.GroupSnapshotId)
		}
		if len(fetched.GetGroupSnapshot().GetSnapshots()) != len(snapshotIDs) {
			t.Fatalf("get returned %d member snapshots, want %d", len(fetched.GetGroupSnapshot().GetSnapshots()), len(snapshotIDs))
		}

		_, err = client.DeleteVolumeGroupSnapshot(ctx, &csi.DeleteVolumeGroupSnapshotRequest{
			GroupSnapshotId: group.GroupSnapshotId,
		})
		requireRPCCode(t, err, codes.InvalidArgument)

		_, err = client.DeleteVolumeGroupSnapshot(ctx, &csi.DeleteVolumeGroupSnapshotRequest{
			GroupSnapshotId: group.GroupSnapshotId,
			SnapshotIds:     []string{"wrong-member-snapshot"},
		})
		requireRPCCode(t, err, codes.InvalidArgument)

		_, err = client.DeleteVolumeGroupSnapshot(ctx, &csi.DeleteVolumeGroupSnapshotRequest{
			GroupSnapshotId: group.GroupSnapshotId,
			SnapshotIds:     snapshotIDs,
		})
		if err != nil {
			t.Fatalf("DeleteVolumeGroupSnapshot failed: %v", err)
		}

		_, err = client.GetVolumeGroupSnapshot(ctx, &csi.GetVolumeGroupSnapshotRequest{
			GroupSnapshotId: group.GroupSnapshotId,
		})
		requireRPCCode(t, err, codes.NotFound)

		_, err = client.DeleteVolumeGroupSnapshot(ctx, &csi.DeleteVolumeGroupSnapshotRequest{
			GroupSnapshotId: group.GroupSnapshotId,
			SnapshotIds:     snapshotIDs,
		})
		if err != nil {
			t.Fatalf("idempotent DeleteVolumeGroupSnapshot failed: %v", err)
		}
	})
}

func newGroupControllerSanityClient(t *testing.T) csi.GroupControllerClient {
	t.Helper()

	driver, _ := initCSIDriverAndProviderForSanity(t)
	tempDir, err := os.MkdirTemp("/tmp", "vgs-sanity-")
	if err != nil {
		t.Fatalf("Failed to create sanity temp directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove sanity temp directory: %v", err)
		}
	})

	endpoint := fmt.Sprintf("unix:%s", path.Join(tempDir, "csi.sock"))
	go driver.Run(endpoint)

	connection, err := sanityUtils.Connect(
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect to CSI driver: %v", err)
	}
	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("Failed to close CSI connection: %v", err)
		}
	})

	return csi.NewGroupControllerClient(connection)
}

func requireRPCCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if status.Code(err) != want {
		t.Fatalf("expected gRPC code %s, got %s: %v", want, status.Code(err), err)
	}
}

func (c *fakeProviderSession) CreateGroupSnapshot(sourceVolumeIDs []string, parameters provider.GroupSnapshotParameters) (*provider.GroupSnapshot, error) {
	groupSnapshotID := fmt.Sprintf("group-snapshot-%s", uuid.New().String())
	creationTime := time.Now()
	memberSnapshots := make([]*provider.Snapshot, 0, len(sourceVolumeIDs))

	for _, sourceVolumeID := range sourceVolumeIDs {
		snapshotID := fmt.Sprintf("snapshot-%s", uuid.New().String())
		snapshot := &provider.Snapshot{
			VolumeID:             sourceVolumeID,
			SnapshotID:           snapshotID,
			SnapshotCRN:          snapshotID,
			SnapshotSize:         1,
			SnapshotCreationTime: creationTime,
			ReadyToUse:           true,
		}
		memberSnapshots = append(memberSnapshots, snapshot)
		c.snapshots[snapshotID] = &fakeSnapshot{Snapshot: snapshot}
	}

	groupSnapshot := &provider.GroupSnapshot{
		GroupSnapshotID:           groupSnapshotID,
		GroupSnapshotCRN:          groupSnapshotID,
		GroupSnapshotCreationTime: creationTime,
		Snapshots:                 memberSnapshots,
		ReadyToUse:                true,
	}
	c.groupSnapshots[groupSnapshotID] = &fakeGroupSnapshot{
		GroupSnapshot:   groupSnapshot,
		name:            parameters.Name,
		resourceGroupID: parameters.ResourceGroup,
	}

	return groupSnapshot, nil
}

func (c *fakeProviderSession) DeleteGroupSnapshot(groupSnapshotID string, snapshotIDs []string) error {
	groupSnapshot, exists := c.groupSnapshots[groupSnapshotID]
	if !exists {
		return providerError.Message{
			Code:        "GroupSnapshotNotFound",
			Description: "Group snapshot not found",
			Type:        providerError.RetrivalFailed,
		}
	}

	expectedSnapshotIDs := make(map[string]struct{}, len(groupSnapshot.Snapshots))
	for _, snapshot := range groupSnapshot.Snapshots {
		expectedSnapshotIDs[snapshot.SnapshotCRN] = struct{}{}
	}
	seenSnapshotIDs := make(map[string]struct{}, len(snapshotIDs))
	for _, snapshotID := range snapshotIDs {
		if _, exists := expectedSnapshotIDs[snapshotID]; !exists {
			return groupSnapshotMemberMismatchError()
		}
		seenSnapshotIDs[snapshotID] = struct{}{}
	}
	if len(seenSnapshotIDs) != len(expectedSnapshotIDs) {
		return groupSnapshotMemberMismatchError()
	}

	for _, snapshot := range groupSnapshot.Snapshots {
		delete(c.snapshots, snapshot.SnapshotCRN)
	}
	delete(c.groupSnapshots, groupSnapshotID)
	return nil
}

func (c *fakeProviderSession) GetGroupSnapshot(groupSnapshotID string) (*provider.GroupSnapshot, error) {
	groupSnapshot, exists := c.groupSnapshots[groupSnapshotID]
	if !exists {
		return nil, nil
	}
	return groupSnapshot.GroupSnapshot, nil
}

func (c *fakeProviderSession) GetGroupSnapshotByName(groupSnapshotName string, resourceGroupID string) (*provider.GroupSnapshot, error) {
	for _, groupSnapshot := range c.groupSnapshots {
		if groupSnapshot.name == groupSnapshotName && groupSnapshot.resourceGroupID == resourceGroupID {
			return groupSnapshot.GroupSnapshot, nil
		}
	}
	return nil, nil
}

func groupSnapshotMemberMismatchError() error {
	return providerError.Message{
		Code:        "GroupSnapshotMembersMismatch",
		Description: "Group snapshot member IDs do not match",
		Type:        providerError.InvalidRequest,
	}
}

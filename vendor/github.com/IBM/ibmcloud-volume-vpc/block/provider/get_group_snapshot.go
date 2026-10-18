/**
 * Copyright 2026 IBM Corp.
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

package provider

import (
	"errors"
	"fmt"

	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// GetGroupSnapshot gets a snapshot consistency group by ID
func (vpcs *VPCSession) GetGroupSnapshot(groupSnapshotID string) (*provider.GroupSnapshot, error) {
	vpcs.Logger.Info("Entry GetGroupSnapshot", zap.Reflect("groupSnapshotID", groupSnapshotID))
	defer vpcs.Logger.Info("Exit GetGroupSnapshot", zap.Reflect("groupSnapshotID", groupSnapshotID))

	var (
		result *models.SnapshotConsistencyGroupResponse
		err    error
	)
	err = retry(vpcs.Logger, func() error {
		result, err = vpcs.Apiclient.SnapshotConsistencyGroupService().GetSnapshotConsistencyGroup(groupSnapshotID, vpcs.Logger)
		return err
	})
	if err != nil {
		return nil, userError.GetUserError("GroupSnapshotRetrievalFailed", err, groupSnapshotID)
	}

	vpcs.Logger.Info("Successfully retrieved snapshot consistency group details", zap.Reflect("groupSnapshotDetails", result))

	return vpcs.getGroupSnapshotWithMembers(result)
}

// GetGroupSnapshotByName gets a snapshot consistency group by name and resourceGroupID
func (vpcs *VPCSession) GetGroupSnapshotByName(name string, resourceGroupID string) (*provider.GroupSnapshot, error) {
	vpcs.Logger.Debug("Entry of GetGroupSnapshotByName method...")
	defer vpcs.Logger.Debug("Exit from GetGroupSnapshotByName method...")

	vpcs.Logger.Info("Retrieving snapshot consistency group by name from VPC", zap.Reflect("GroupSnapshotName", name))

	if len(name) == 0 {
		return nil, userError.GetUserError("InvalidGroupSnapshotName", nil)
	}

	var (
		result *models.SnapshotConsistencyGroupResponse
		err    error
	)
	err = retry(vpcs.Logger, func() error {
		result, err = vpcs.Apiclient.SnapshotConsistencyGroupService().GetSnapshotConsistencyGroupByName(name, resourceGroupID, vpcs.Logger)
		return err
	})
	if err != nil {
		return nil, userError.GetUserError("GroupSnapshotNameLookupFailed", err, name)
	}

	if result == nil {
		return nil, nil
	}

	vpcs.Logger.Info("Successfully retrieved snapshot consistency group details", zap.Reflect("groupSnapshotDetails", result))

	return vpcs.getGroupSnapshotWithMembers(result)
}

// getGroupSnapshotWithMembers fetches full member details and converts the group.
// If any member's source volumeID is missing, return an error so CSI sidecar can retry.
func (vpcs *VPCSession) getGroupSnapshotWithMembers(group *models.SnapshotConsistencyGroupResponse) (*provider.GroupSnapshot, error) {
	if group == nil {
		return nil, errors.New("VPC returned an empty snapshot consistency group response")
	}

	// Reject an empty groupID
	if group.ID == "" {
		return nil, errors.New("VPC returned a snapshot consistency group without an ID")
	}
	snapshotDetails, err := vpcs.listGroupMemberSnapshots(group.ID)
	if err != nil {
		vpcs.Logger.Error("Failed to retrieve individual member snapshot details",
			zap.String("groupSnapshotID", group.ID), zap.Error(err))
		return nil, userError.GetUserError("GroupSnapshotMemberLookupFailed", err, group.ID)
	}

	// Check if any member snapshot is missing its source volumeID.
	response := FromProviderToLibGroupSnapshot(group, snapshotDetails, vpcs.Logger)
	if len(response.Snapshots) == 0 {
		return nil, userError.GetUserError("GroupSnapshotMemberDetailsUnavailable", nil, group.ID)
	}
	for _, member := range response.Snapshots {
		if member == nil || member.VolumeID == "" {
			return nil, userError.GetUserError("GroupSnapshotMemberDetailsUnavailable", nil, group.ID)
		}
	}
	return response, nil
}

// listGroupMemberSnapshots reads every page of the group's member snapshots.
// If any page fails, return error without retrying/returning partial results. CSI can then retry the req from the start.
func (vpcs *VPCSession) listGroupMemberSnapshots(groupID string) ([]*models.Snapshot, error) {
	filters := &models.LisSnapshotFilters{SnapshotConsistencyGroupID: groupID}
	seenTokens := map[string]bool{}
	var members []*models.Snapshot
	start := ""
	// Pagination loop: the group filter and limit stay the same; only the start token changes.
	for {
		page, err := vpcs.Apiclient.SnapshotService().ListSnapshots(maxLimit, start, filters, vpcs.Logger)
		if err != nil {
			return nil, err
		}
		if page == nil {
			return nil, errors.New("VPC returned an empty snapshot page")
		}

		members = append(members, page.Snapshots...)

		start, err = nextSnapshotStart(page)
		if err != nil {
			return nil, err
		}

		if start == "" {
			return members, nil
		}

		// Stop if a page token repeats, so we do not keep fetching the same pages.
		if seenTokens[start] {
			return nil, fmt.Errorf("snapshot list repeated page token %q", start)
		}
		seenTokens[start] = true
	}
}

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
	"net/url"
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/metrics"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// GetGroupSnapshot gets a volume group snapshot by ID
func (vpcs *VPCSession) GetGroupSnapshot(groupSnapshotID string) (*provider.GroupSnapshot, error) {
	vpcs.Logger.Info("Entry GetGroupSnapshot", zap.Reflect("groupSnapshotID", groupSnapshotID))
	defer vpcs.Logger.Info("Exit GetGroupSnapshot", zap.Reflect("groupSnapshotID", groupSnapshotID))
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "GetGroupSnapshot", time.Now())

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

	vpcs.Logger.Info("Successfully retrieved volume group snapshot details", zap.Reflect("groupSnapshotDetails", result))

	return vpcs.getGroupSnapshotWithMembers(result)
}

// GetGroupSnapshotByName gets a volume group snapshot by name and resourceGroupID
func (vpcs *VPCSession) GetGroupSnapshotByName(name string, resourceGroupID string) (*provider.GroupSnapshot, error) {
	vpcs.Logger.Info("Entry GetGroupSnapshotByName", zap.String("groupSnapshotName", name), zap.String("resourceGroupID", resourceGroupID))
	defer vpcs.Logger.Info("Exit GetGroupSnapshotByName", zap.String("groupSnapshotName", name), zap.String("resourceGroupID", resourceGroupID))
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "GetGroupSnapshotByName", time.Now())

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

	vpcs.Logger.Info("Successfully retrieved volume group snapshot details", zap.Reflect("groupSnapshotDetails", result))

	return vpcs.getGroupSnapshotWithMembers(result)
}

// The group API lists snapshot IDs but doesn't say which volume each came from.
func (vpcs *VPCSession) getGroupSnapshotWithMembers(group *models.SnapshotConsistencyGroupResponse) (*provider.GroupSnapshot, error) {
	if group == nil {
		return nil, userError.GetUserError("InvalidGroupSnapshotResponse", nil, "the response is empty")
	}
	// An empty group ID would list snapshots for the entire account.
	if group.ID == "" {
		return nil, userError.GetUserError("InvalidGroupSnapshotResponse", nil, "the group ID is missing")
	}
	if len(group.Snapshots) == 0 {
		return nil, userError.GetUserError("GroupSnapshotMemberDetailsUnavailable", nil, group.ID)
	}
	if group.CreatedAt == nil || group.CreatedAt.IsZero() || group.LifecycleState == "" {
		return nil, userError.GetUserError("InvalidGroupSnapshotResponse", nil, fmt.Sprintf("group %q is missing its creation time or lifecycle state", group.ID))
	}

	members, err := vpcs.listGroupMemberSnapshots(group.ID)
	if err != nil {
		return nil, err
	}

	return fromProviderToLibGroupSnapshot(group, members, vpcs.Logger)
}

// Read every page with the same group filter. If a page fails, don't return a partial list.
func (vpcs *VPCSession) listGroupMemberSnapshots(groupID string) (map[string]*models.Snapshot, error) {
	filters := &models.LisSnapshotFilters{SnapshotConsistencyGroupID: groupID}

	members := make(map[string]*models.Snapshot)
	seenTokens := make(map[string]bool)

	start := ""
	for {
		page, err := vpcs.Apiclient.SnapshotService().ListSnapshots(maxLimit, start, filters, vpcs.Logger)
		if err != nil {
			return nil, userError.GetUserError("GroupSnapshotMemberLookupFailed", err, groupID)
		}

		if page == nil {
			return nil, userError.GetUserError("InvalidGroupSnapshotResponse", nil, fmt.Sprintf("empty member snapshot page for group %q", groupID))
		}

		for _, snapshot := range page.Snapshots {
			if snapshot == nil || snapshot.ID == "" {
				continue
			}

			if _, exists := members[snapshot.ID]; exists {
				return nil, userError.GetUserError("GroupSnapshotMemberLookupFailed", fmt.Errorf("VPC listed snapshot %q more than once", snapshot.ID), groupID)
			}
			members[snapshot.ID] = snapshot
		}

		start, err = nextSnapshotStart(page.Next)
		if err != nil {
			return nil, userError.GetUserError("GroupSnapshotMemberLookupFailed", err, groupID)
		}

		if start == "" {
			return members, nil
		}
		// A token we already used would send us back to a page we've read.
		if seenTokens[start] {
			return nil, userError.GetUserError("GroupSnapshotMemberLookupFailed", errors.New("VPC repeated a snapshot page token"), groupID)
		}
		seenTokens[start] = true
	}
}

// Read the next page token. No link means there are no more pages.
func nextSnapshotStart(nextLink *models.HReference) (string, error) {
	if nextLink == nil {
		return "", nil
	}

	link, err := url.Parse(nextLink.Href)
	if err != nil {
		return "", fmt.Errorf("invalid snapshot next-page link: %w", err)
	}

	query, err := url.ParseQuery(link.RawQuery)
	if err != nil {
		return "", fmt.Errorf("invalid snapshot next-page query: %w", err)
	}

	tokens := query["start"]
	if len(tokens) != 1 || tokens[0] == "" {
		return "", errors.New("snapshot next-page link must contain one non-empty start token")
	}

	return tokens[0], nil
}

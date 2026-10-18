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

package models

import "time"

// SnapshotConsistencyGroupResponse is the response received from the VPC
type SnapshotConsistencyGroupResponse struct {
	Href                    string                     `json:"href"`
	ID                      string                     `json:"id"`
	Name                    string                     `json:"name"`
	CRN                     string                     `json:"crn"`
	ResourceGroup           *ResourceGroup             `json:"resource_group"`
	CreatedAt               *time.Time                 `json:"created_at"`
	Status                  string                     `json:"status"`
	LifecycleState          string                     `json:"lifecycle_state"`
	ResourceType            string                     `json:"resource_type"`
	Snapshots               []SnapshotReference        `json:"snapshots"`
	ServiceTags             []string                   `json:"service_tags"`
	UserTags                []string                   `json:"user_tags"`
	DeleteSnapshotsOnDelete bool                       `json:"delete_snapshots_on_delete"`
	BackupPolicyPlan        *BackupPolicyPlanReference `json:"backup_policy_plan"`
}

type BackupPolicyPlanReference struct {
	ID           string                  `json:"id"`
	Href         string                  `json:"href"`
	Name         string                  `json:"name"`
	Deleted      *Deleted                `json:"deleted"`
	ResourceType string                  `json:"resource_type"`
	Remote       *BackupPolicyPlanRemote `json:"remote"`
}

type BackupPolicyPlanRemote struct {
	Region *RegionReference `json:"region"`
}

type RegionReference struct {
	Href string `json:"href"`
	Name string `json:"name"`
}

// SnapshotReference is a reference to a snapshot within a consistency group
type SnapshotReference struct {
	Href         string   `json:"href"`
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	CRN          string   `json:"crn"`
	ResourceType string   `json:"resource_type"`
	Deleted      *Deleted `json:"deleted"`
}

// SnapshotConsistencyGroupList ...
type SnapshotConsistencyGroupList struct {
	First                     *HReference                         `json:"first,omitempty"`
	Limit                     int                                 `json:"limit,omitempty"`
	SnapshotConsistencyGroups []*SnapshotConsistencyGroupResponse `json:"snapshot_consistency_groups"`
	Next                      *HReference                         `json:"next,omitempty"`
	TotalCount                int                                 `json:"total_count,omitempty"`
}

// SnapshotConsistencyGroupRequest is the request body for creating a consistency group
type SnapshotConsistencyGroupRequest struct {
	Name                    string                  `json:"name,omitempty"`
	ResourceGroup           *ResourceGroup          `json:"resource_group,omitempty"`
	Snapshots               []GroupSnapshotTemplate `json:"snapshots"`
	DeleteSnapshotsOnDelete bool                    `json:"delete_snapshots_on_delete"`
}

// GroupSnapshotTemplate is a snapshot definition within a consistency group create request
type GroupSnapshotTemplate struct {
	Name         string        `json:"name,omitempty"`
	SourceVolume *SourceVolume `json:"source_volume"`
	UserTags     []string      `json:"user_tags,omitempty"`
}

// ListSnapshotConsistencyGroupFilters ...
type ListSnapshotConsistencyGroupFilters struct {
	ResourceGroupID string `json:"resource_group.id,omitempty"`
	Name            string `json:"name,omitempty"`
}

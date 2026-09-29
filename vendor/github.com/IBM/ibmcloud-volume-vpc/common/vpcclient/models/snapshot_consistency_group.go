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

type SnapshotConsistencyGroup struct {
	Href                    string                     `json:"href,omitempty"`
	ID                      string                     `json:"id,omitempty"`
	Name                    string                     `json:"name,omitempty"`
	CRN                     string                     `json:"crn,omitempty"`
	ResourceGroup           *ResourceGroup             `json:"resource_group,omitempty"`
	CreatedAt               *time.Time                 `json:"created_at,omitempty"`
	Status                  string                     `json:"status,omitempty"`
	LifecycleState          string                     `json:"lifecycle_state,omitempty"`
	ResourceType            string                     `json:"resource_type,omitempty"`
	Snapshots               []SnapshotReference        `json:"snapshots,omitempty"`
	ServiceTags             []string                   `json:"service_tags,omitempty"`
	UserTags                []string                   `json:"user_tags,omitempty"`
	DeleteSnapshotsOnDelete bool                       `json:"delete_snapshots_on_delete,omitempty"`
	BackupPolicyPlan        *BackupPolicyPlanReference `json:"backup_policy_plan,omitempty"`
}

type BackupPolicyPlanReference struct {
	ID           string                  `json:"id,omitempty"`
	Href         string                  `json:"href,omitempty"`
	Name         string                  `json:"name,omitempty"`
	Deleted      *Deleted                `json:"deleted,omitempty"`
	ResourceType string                  `json:"resource_type,omitempty"`
	Remote       *BackupPolicyPlanRemote `json:"remote,omitempty"`
}

type BackupPolicyPlanRemote struct {
	Region *RegionReference `json:"region,omitempty"`
}

type RegionReference struct {
	Href string `json:"href,omitempty"`
	Name string `json:"name,omitempty"`
}

// SnapshotReference is a reference to a snapshot within a consistency group
type SnapshotReference struct {
	Href         string   `json:"href,omitempty"`
	ID           string   `json:"id,omitempty"`
	Name         string   `json:"name,omitempty"`
	CRN          string   `json:"crn,omitempty"`
	ResourceType string   `json:"resource_type,omitempty"`
	Deleted      *Deleted `json:"deleted,omitempty"`
}

// SnapshotConsistencyGroupList ...
type SnapshotConsistencyGroupList struct {
	First                     *HReference                 `json:"first,omitempty"`
	Limit                     int                         `json:"limit,omitempty"`
	SnapshotConsistencyGroups []*SnapshotConsistencyGroup `json:"snapshot_consistency_groups"`
	Next                      *HReference                 `json:"next,omitempty"`
	TotalCount                int                         `json:"total_count,omitempty"`
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

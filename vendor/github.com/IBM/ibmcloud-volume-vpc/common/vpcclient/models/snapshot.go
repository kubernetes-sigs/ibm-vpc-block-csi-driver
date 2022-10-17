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

// Package models ...
package models

import "time"

// Snapshot ...
// type Snapshot struct {
// 	Href          string         `json:"href,omitempty"`
// 	ID            string         `json:"id,omitempty"`
// 	Name          string         `json:"name,omitempty"`
// 	ResourceGroup *ResourceGroup `json:"resource_group,omitempty"`
// 	CRN           string         `json:"crn,omitempty"`
// 	CreatedAt     *time.Time     `json:"created_at,omitempty"`
// 	Status        StatusType     `json:"status,omitempty"`
// 	Tags          []string       `json:"tags,omitempty"`
// }

// SnapshotList ...
type SnapshotList struct {
	First      *HReference `json:"first,omitempty"`
	Next       *HReference `json:"next,omitempty"`
	Snapshots  []*Snapshot `json:"snapshots"`
	Limit      int         `json:"limit,omitempty"`
	TotalCount int         `json:"total_count,omitempty"`
}

// LisSnapshotFilters ...
type LisSnapshotFilters struct {
	ResourceGroupID string `json:"resource_group.id,omitempty"`
	Name            string `json:"name,omitempty"`
	SourceVolumeID  string `json:"source_volume.id,omitempty"`
}

// Snapshot ...
type Snapshot struct {
	Href            string           `json:"href,omitempty"`
	ID              string           `json:"id,omitempty"`
	Name            string           `json:"name,omitempty"`
	MinimumCapacity int64            `json:"minimum_capacity,omitempty"`
	ResourceGroup   *ResourceGroup   `json:"resource_group,omitempty"`
	OperatingSystem *OperatingSystem `json:"operating_system,omitempty"`
	CreatedAt       *time.Time       `json:"created_at,omitempty"`
	Status          string           `json:"status,omitempty"`
	Encryption      string           `json:"encryption,omitempty"`
	ResourceType    string           `json:"resource_type,omitempty"`
	Size            int64            `json:"size,omitempty"`
	Bootable        bool             `json:"bootable,omitempty"`
	LifecycleState  string           `json:"lifecycle_state,omitempty"`
	SourceVolume    *SourceVolume    `json:"source_volume,omitempty"`
	UserTags        []string         `json:"user_tags,omitempty"`
	ServiceTags     []string         `json:"service_tags,omitempty"`
	CapturedAt      *time.Time       `json:"captured_at,omitempty"`

	SourceImage      *SourceImage         `json:"source_image,omitempty"`
	CRN              string               `json:"crn,omitempty"`
	Clones           *[]Clone             `json:"clones,omitempty"`
	BackupPolicyPlan *BackupPolicyPlan    `json:"backup_policy_plan,omitempty"`
	EncryptionKey    *VolumeEncryptionKey `json:"encryption_key,omitempty"`
}

// BackupPolicyPlan ...
type BackupPolicyPlan struct {
	ID           string   `json:"id,omitempty"`
	Href         string   `json:"href,omitempty"`
	Name         string   `json:"name,omitempty"`
	Deleted      *Deleted `json:"deleted,omitempty"`
	ResourceType string   `json:"resource_type,omitempty"`
}

// OperatingSystem ...
type OperatingSystem struct {
	Href              string `json:"href,omitempty"`
	Version           string `json:"version,omitempty"`
	Vendor            string `json:"vendor,omitempty"`
	Name              string `json:"name,omitempty"`
	Family            string `json:"family,omitempty"`
	DisplayName       string `json:"display_name,omitempty"`
	Architecture      string `json:"architecture,omitempty"`
	DedicatedHostOnly bool   `json:"dedicated_host_only,omitempty"`
}

// SourceImage ...
type SourceImage struct {
	ID      string   `json:"id,omitempty"`
	Href    string   `json:"href,omitempty"`
	Name    string   `json:"name,omitempty"`
	CRN     string   `json:"crn,omitempty"`
	Deleted *Deleted `json:"deleted,omitempty"`
}

// SourceVolume ...
type SourceVolume struct {
	ID      string   `json:"id,omitempty"`
	Href    string   `json:"href,omitempty"`
	Name    string   `json:"name,omitempty"`
	CRN     string   `json:"crn,omitempty"`
	Deleted *Deleted `json:"deleted,omitempty"`
}

// Deleted ...
type Deleted struct {
	MoreInfo string `json:"more_info,omitempty"`
}

// Clone ...
type Clone struct {
	Available string     `json:"available,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	Zone      *Zone      `json:"zone,omitempty"`
}

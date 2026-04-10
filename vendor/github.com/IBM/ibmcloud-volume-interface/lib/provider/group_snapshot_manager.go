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

// Package provider ...
package provider

// GroupSnapshotManager ...
type GroupSnapshotManager interface {
	// CreateGroupSnapshot creates a group snapshot for the given source volume IDs
	CreateGroupSnapshot(sourceVolumeIDs []string, groupSnapshotParameters GroupSnapshotParameters) (*GroupSnapshot, error)

	// DeleteGroupSnapshot deletes the group snapshot
	DeleteGroupSnapshot(groupSnapshotID string) error

	// GetGroupSnapshot fetches the group snapshot by ID
	GetGroupSnapshot(groupSnapshotID string) (*GroupSnapshot, error)

	// GetGroupSnapshotByName gets the group snapshot by name, scoped to a resource group
	GetGroupSnapshotByName(groupSnapshotName string, resourceGroupID string) (*GroupSnapshot, error)
}

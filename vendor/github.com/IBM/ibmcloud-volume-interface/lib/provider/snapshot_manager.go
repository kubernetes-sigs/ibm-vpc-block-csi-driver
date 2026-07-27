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

// Package provider ...
package provider

// SnapshotManager ...
type SnapshotManager interface {
	// CreateSnapshot creates the snapshot on the volume
	CreateSnapshot(sourceVolumeID string, snapshotParameters SnapshotParameters) (*Snapshot, error)

	// DeleteSnapshot deletes the snapshot
	DeleteSnapshot(*Snapshot) error

	// GetSnapshot fetches the snapshot using snapshotID
	GetSnapshot(snapshotID string, sourceVolumeID ...string) (*Snapshot, error)

	// GetSnapshotByName gets the snapshot by name and scoped parameters. scopeID is optional and driver-dependent
	// Block CSI driver passes resourceGroupID; File CSI driver passes it as sourceVolumeID
	GetSnapshotByName(snapshotName string, scopeID ...string) (*Snapshot, error)

	// ListSnapshots lists the snapshots by using tags
	ListSnapshots(limit int, start string, tags map[string]string) (*SnapshotList, error)
}

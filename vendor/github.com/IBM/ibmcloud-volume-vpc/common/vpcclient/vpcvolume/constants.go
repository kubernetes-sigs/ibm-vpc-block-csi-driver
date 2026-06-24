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

// Package vpcvolume ...
package vpcvolume

const (
	// Version of the VPC backend service
	Version       = "/v1"
	volumesPath   = Version + "/volumes"
	volumeIDParam = "volume-id"
	volumeIDPath  = volumesPath + "/{" + volumeIDParam + "}"

	snapshotsPath   = Version + "/snapshots"
	snapshotIDParam = "snapshot-id"
	snapshotIDPath  = snapshotsPath + "/{" + snapshotIDParam + "}"

	volumeTagsPath    = volumesPath + "/{" + volumeIDParam + "}/" + "tags"
	volumeTagParam    = "tag-name"
	volumeTagNamePath = volumeTagsPath + "/{" + volumeTagParam + "}"

	snapshotTagsPath    = snapshotIDPath + "/" + "tags"
	snapshotTagParam    = "tag-name"
	snapshotTagNamePath = snapshotTagsPath + "/{" + snapshotTagParam + "}"
	updateVolume        = "updateVolume"
)

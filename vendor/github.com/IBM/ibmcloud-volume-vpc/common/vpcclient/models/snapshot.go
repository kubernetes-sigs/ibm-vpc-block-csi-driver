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
type Snapshot struct {
	Href          string         `json:"href,omitempty"`
	ID            string         `json:"id,omitempty"`
	Name          string         `json:"name,omitempty"`
	ResourceGroup *ResourceGroup `json:"resource_group,omitempty"`
	CRN           string         `json:"crn,omitempty"`
	CreatedAt     *time.Time     `json:"created_at,omitempty"`
	Status        StatusType     `json:"status,omitempty"`
	Tags          []string       `json:"tags,omitempty"`
}

// SnapshotList ...
type SnapshotList struct {
	Snapshots []*Snapshot `json:"snapshot,omitempty"`
}

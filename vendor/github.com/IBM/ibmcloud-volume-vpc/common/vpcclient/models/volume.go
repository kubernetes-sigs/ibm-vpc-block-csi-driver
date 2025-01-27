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

import (
	"strconv"
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
)

const (
	//ClusterIDTagName ...
	ClusterIDTagName = "clusterid"
	//VolumeStatus ...
	VolumeStatus = "status"
)

// Volume ...
type Volume struct {
	Href                string               `json:"href,omitempty"`
	ID                  string               `json:"id,omitempty"`
	Name                string               `json:"name,omitempty"`
	Capacity            int64                `json:"capacity,omitempty"`
	Iops                int64                `json:"iops,omitempty"`
	VolumeEncryptionKey *VolumeEncryptionKey `json:"encryption_key,omitempty"`
	ResourceGroup       *ResourceGroup       `json:"resource_group,omitempty"`
	Tags                []string             `json:"tags,omitempty"` //We need to validate and remove this if not required.
	UserTags            []string             `json:"user_tags,omitempty"`
	Profile             *Profile             `json:"profile,omitempty"`
	SourceSnapshot      *Snapshot            `json:"source_snapshot,omitempty"`
	CreatedAt           *time.Time           `json:"created_at,omitempty"`
	Status              StatusType           `json:"status,omitempty"`
	VolumeAttachments   *[]VolumeAttachment  `json:"volume_attachments,omitempty"`

	Zone          *Zone                 `json:"zone,omitempty"`
	CRN           string                `json:"crn,omitempty"`
	Cluster       string                `json:"cluster,omitempty"`
	Provider      string                `json:"provider,omitempty"`
	VolumeType    string                `json:"volume_type,omitempty"`
	HealthState   string                `json:"health_state,omitempty"`
	HealthReasons *[]VolumeHealthReason `json:"health_reasons,omitempty"`
}

// ListVolumeFilters ...
type ListVolumeFilters struct {
	ResourceGroupID string `json:"resource_group.id,omitempty"`
	Tag             string `json:"tag,omitempty"`
	ZoneName        string `json:"zone.name,omitempty"`
	VolumeName      string `json:"name,omitempty"`
}

// VolumeList ...
type VolumeList struct {
	First      *HReference `json:"first,omitempty"`
	Next       *HReference `json:"next,omitempty"`
	Volumes    []*Volume   `json:"volumes"`
	Limit      int         `json:"limit,omitempty"`
	TotalCount int         `json:"total_count,omitempty"`
}

// HReference ...
type HReference struct {
	Href string `json:"href,omitempty"`
}

// NewVolume created model volume from provider volume
func NewVolume(volumeRequest provider.Volume) Volume {
	// Build the template to send to backend

	volume := Volume{
		ID:   volumeRequest.VolumeID,
		CRN:  volumeRequest.CRN,
		Tags: volumeRequest.VPCVolume.Tags,
		Zone: &Zone{
			Name: volumeRequest.Az,
		},
		Provider:   string(volumeRequest.Provider),
		VolumeType: string(volumeRequest.VolumeType),
	}
	if volumeRequest.Name != nil {
		volume.Name = *volumeRequest.Name
	}
	if volumeRequest.Capacity != nil {
		volume.Capacity = int64(*volumeRequest.Capacity)
	}
	if volumeRequest.VPCVolume.Profile != nil {
		volume.Profile = &Profile{
			Name: volumeRequest.VPCVolume.Profile.Name,
		}
	}
	if volumeRequest.VPCVolume.ResourceGroup != nil {
		volume.ResourceGroup = &ResourceGroup{
			ID:   volumeRequest.VPCVolume.ResourceGroup.ID,
			Name: volumeRequest.VPCVolume.ResourceGroup.Name,
		}
	}

	if volumeRequest.Iops != nil {
		value, err := strconv.ParseInt(*volumeRequest.Iops, 10, 64)
		if err != nil {
			volume.Iops = 0
		}
		volume.Iops = value
	}
	if volumeRequest.VPCVolume.VolumeEncryptionKey != nil && len(volumeRequest.VPCVolume.VolumeEncryptionKey.CRN) > 0 {
		encryptionKeyCRN := volumeRequest.VPCVolume.VolumeEncryptionKey.CRN
		volume.VolumeEncryptionKey = &VolumeEncryptionKey{CRN: encryptionKeyCRN}
	}

	volume.Cluster = volumeRequest.Attributes[ClusterIDTagName]
	volume.Status = StatusType(volumeRequest.Attributes[VolumeStatus])
	return volume
}

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

import (
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/client"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// VolumeManager operations
type VolumeManager interface {
	// Create the volume with authorisation by passing required information in the volume object
	CreateVolume(volumeTemplate *models.Volume, ctxLogger *zap.Logger) (*models.Volume, error)
	// UpdateVolume updates the volume with authorisation by passing required information in the volume object
	UpdateVolume(volumeTemplate *models.Volume, ctxLogger *zap.Logger) error

	// ExpandVolume ...
	ExpandVolume(volumeID string, volumeTemplate *models.Volume, ctxLogger *zap.Logger) (*models.Volume, error)

	// Delete the volume
	DeleteVolume(volumeID string, ctxLogger *zap.Logger) error

	// Get the volume by using ID
	GetVolume(volumeID string, ctxLogger *zap.Logger) (*models.Volume, error)

	// Get the volume by using volume name
	GetVolumeByName(volumeName string, ctxLogger *zap.Logger) (*models.Volume, error)

	// Others
	// Get volume lists by using snapshot tags
	ListVolumes(limit int, start string, filters *models.ListVolumeFilters, ctxLogger *zap.Logger) (*models.VolumeList, error)

	// Set tag for a volume
	SetVolumeTag(volumeID string, tagName string, ctxLogger *zap.Logger) error

	// Delete tag of a volume
	DeleteVolumeTag(volumeID string, tagName string, ctxLogger *zap.Logger) error

	// List all tags of a volume
	ListVolumeTags(volumeID string, ctxLogger *zap.Logger) (*[]string, error)

	// Check if the given tag exists on a volume
	CheckVolumeTag(volumeID string, tagName string, ctxLogger *zap.Logger) error
}

// VolumeService ...
type VolumeService struct {
	client client.SessionClient
}

var _ VolumeManager = &VolumeService{}

// New ...
func New(client client.SessionClient) VolumeManager {
	return &VolumeService{
		client: client,
	}
}

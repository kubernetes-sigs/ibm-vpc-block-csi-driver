/**
 * Copyright 2021 IBM Corp.
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

import (
	"net/http"
	"time"
)

//VolumeFileAccessPointManager ...
type VolumeFileAccessPointManager interface {
	//CreateVolumeAccessPoint to create a access point
	CreateVolumeAccessPoint(accessPointRequest VolumeAccessPointRequest) (*VolumeAccessPointResponse, error)

	//DeleteVolumeAccessPoint method delete a access point
	DeleteVolumeAccessPoint(deleteAccessPointRequest VolumeAccessPointRequest) (*http.Response, error)

	//WaitForCreateVolumeAccessPoint waits for the volume access point to be created
	//Return error if wait is timed out OR there is other error
	WaitForCreateVolumeAccessPoint(accessPointRequest VolumeAccessPointRequest) (*VolumeAccessPointResponse, error)

	//WaitForDeleteVolumeAccessPoint waits for the volume access point to be deleted
	//Return error if wait is timed out OR there is other error
	WaitForDeleteVolumeAccessPoint(deleteAccessPointRequest VolumeAccessPointRequest) error

	//GetVolumeAccessPoint retrieves the current status of given volume AccessPoint request
	GetVolumeAccessPoint(accessPointRequest VolumeAccessPointRequest) (*VolumeAccessPointResponse, error)
}

//VolumeAccessPointRequest used for both create and delete access point
type VolumeAccessPointRequest struct {

	//AccessPoint name is optional.
	AccessPointName string `json:"name,omitempty"`

	//Volume to create the AccessPoint for
	VolumeID string `json:"volumeID"`

	//AccessPointID to search or delete access point
	AccessPointID string `json:"accessPointID,omitempty"`

	//Subnet to create AccessPoint for
	SubnetID string `json:"subnet_id,omitempty"`

	//VPC to create AccessPoint for
	VPCID string `json:"vpc_id,omitempty"`
}

//VolumeAccessPointResponse used for both delete and create access point
type VolumeAccessPointResponse struct {
	VolumeID      string     `json:"volumeID"`
	AccessPointID string     `json:"AccessPointID"`
	Status        string     `json:"status"`
	MountPath     string     `json:"mount_path"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
}

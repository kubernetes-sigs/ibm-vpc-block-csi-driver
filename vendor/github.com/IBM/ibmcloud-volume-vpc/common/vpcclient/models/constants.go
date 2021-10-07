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

const (
	// APIVersion is the target RIaaS API spec version
	APIVersion = "2019-07-02"

	// APIGeneration ...
	APIGeneration = 1

	// UserAgent identifies IKS to the RIaaS API
	UserAgent = "IBM-Kubernetes-Service"

	// GTypeClassic ...
	GTypeClassic = "gc"

	// GTypeClassicDevicePrefix ...
	GTypeClassicDevicePrefix = "/dev/"

	// GTypeG2 ...
	GTypeG2 = "g2"

	// GTypeG2DevicePrefix ...
	GTypeG2DevicePrefix = "/dev/disk/by-id/virtio-"

	// GTypeG2DeviceIDLength ...
	GTypeG2DeviceIDLength = 20

	// VolumeAttached ...
	VolumeAttached = "attached"
)

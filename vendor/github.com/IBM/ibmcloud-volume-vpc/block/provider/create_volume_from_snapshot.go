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

import (
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	"go.uber.org/zap"
)

// CreateVolumeFromSnapshot creates the volume by using ID
func (vpcs *VPCSession) CreateVolumeFromSnapshot(snapshot provider.Snapshot, tags map[string]string) (*provider.Volume, error) {
	vpcs.Logger.Info("Entry CreateVolumeFromSnapshot", zap.Reflect("Snapshot", snapshot))
	defer vpcs.Logger.Info("Exit CreateVolumeFromSnapshot", zap.Reflect("Snapshot", snapshot))

	return nil, nil
}

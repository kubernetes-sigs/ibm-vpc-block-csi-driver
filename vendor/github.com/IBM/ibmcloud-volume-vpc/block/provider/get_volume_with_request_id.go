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

// GetVolumeByRequestID get volume by volume ID
func (vpcs *VPCSession) GetVolumeByRequestID(requestID string) (*provider.Volume, error) {
	vpcs.Logger.Info("Entry GetVolumeByRequestID", zap.Reflect("requestID", requestID))
	defer vpcs.Logger.Info("Exit GetVolumeByRequestID", zap.Reflect("requestID", requestID))

	return nil, nil
}

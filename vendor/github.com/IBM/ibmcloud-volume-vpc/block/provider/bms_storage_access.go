/**
 * Copyright 2025 IBM Corp.
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

// bms_storage_access.go implements the BMS-specific VPC API calls needed by
// the CSI controller's ControllerPublishVolume STEP 4:
//
//  1. RotateBMSEncryptedSecret — POST .../storage_access/encrypted_secret/rotate
//     body: {"key":{"id":keyID}}
//     VPC re-encrypts the NVMe-oF PSK with the RSA public key registered as
//     VPC key {keyID}.
//
//  2. GetBMSServer — GET .../bare_metal_servers/{serverID}
//     Returns BOTH the host NQN (nvme_qualified_name) AND the encrypted PSK
//     (storage_access.encrypted_secret) in one round-trip, per the VPC BMS
//     Block Storage SDP User Guide.
//
// All methods delegate to getBMSVolAttachMgr() which returns the
// lazily-initialised BMS VolumeAttachService.

import (
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/metrics"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	"go.uber.org/zap"
)

// RotateBMSEncryptedSecret calls
// POST v1/bare_metal_servers/{serverID}/storage_access/encrypted_secret/rotate
// body: {"key":{"id":keyID}}
//
// Tells VPC to re-encrypt the NVMe-oF PSK with the RSA public key registered
// as VPC key {keyID}.  Must be called before GetBMSServer so that
// the returned ciphertext is encrypted with this cluster's key.
func (vpcs *VPCSession) RotateBMSEncryptedSecret(serverID, keyID string) error {
	vpcs.Logger.Debug("Entry of RotateBMSEncryptedSecret...")
	defer vpcs.Logger.Debug("Exit from RotateBMSEncryptedSecret...")
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "RotateBMSEncryptedSecret", time.Now())

	if err := isValidServiceSession(vpcs); err != nil {
		return err
	}

	vpcs.Logger.Info("RotateBMSEncryptedSecret: calling VPC API", // pragma: allowlist secret
		zap.String("serverID", serverID), zap.String("keyID", keyID))

	if err := vpcs.getBMSVolAttachMgr().RotateEncryptedSecret(serverID, keyID, vpcs.Logger); err != nil { // pragma: allowlist secret
		vpcs.Logger.Error("RotateBMSEncryptedSecret: API call failed", // pragma: allowlist secret
			zap.String("serverID", serverID), zap.Error(err))
		return err
	}

	vpcs.Logger.Info("RotateBMSEncryptedSecret: succeeded", zap.String("serverID", serverID)) // pragma: allowlist secret
	return nil
}

// GetBMSServer calls GET v1/bare_metal_servers/{serverID} and returns both
// the host NQN and the RSA-encrypted PSK ciphertext in a single round-trip.
//
// Per the VPC BMS Block Storage SDP User Guide:
//   nvme_qualified_name            → HOSTNQN  (initiator NQN of this BMS)
//   storage_access.encrypted_secret → PSK ciphertext encrypted with this cluster's RSA key
//
// Call RotateBMSEncryptedSecret first so the ciphertext is encrypted with
// this cluster's RSA public key before calling GetBMSServer.
func (vpcs *VPCSession) GetBMSServer(serverID string) (*provider.BMSServerInfo, error) {
	vpcs.Logger.Debug("Entry of GetBMSServer...")
	defer vpcs.Logger.Debug("Exit from GetBMSServer...")
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "GetBMSServer", time.Now())

	if err := isValidServiceSession(vpcs); err != nil {
		return nil, err
	}

	vpcs.Logger.Info("GetBMSServer: calling VPC API", zap.String("serverID", serverID))

	resp, err := vpcs.getBMSVolAttachMgr().GetBMSServer(serverID, vpcs.Logger)
	if err != nil {
		vpcs.Logger.Error("GetBMSServer: API call failed",
			zap.String("serverID", serverID), zap.Error(err))
		return nil, err
	}

	info := &provider.BMSServerInfo{
		HostNQN: resp.NVMeQualifiedName,
	}
	if resp.StorageAccess != nil {
		info.EncryptedSecret = resp.StorageAccess.EncryptedSecret // pragma: allowlist secret
	}

	vpcs.Logger.Info("GetBMSServer: succeeded",
		zap.String("serverID", serverID),
		zap.String("hostNQN", info.HostNQN),
		zap.Bool("encryptedSecretPresent", info.EncryptedSecret != ""))
	return info, nil
}

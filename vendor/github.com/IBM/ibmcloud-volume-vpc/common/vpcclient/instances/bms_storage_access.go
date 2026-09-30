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

// Package instances ...
package instances

// bms_storage_access.go implements the BMS-specific VPC API methods as
// methods on VolumeAttachService (reusing the same client, pathPrefix and
// error handler already wired by NewBMSVolumeAttachmentManager).
//
// IKSVolumeAttachService stubs return errors because these operations are
// BMS-only and must never be called via the IKS path.
//
// Paths (relative to vs.pathPrefix = "v1/bare_metal_servers"):
//   POST /{instance-id}/storage_access/encrypted_secret/rotate
//   GET  /{instance-id}

import (
	"fmt"
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/metrics"
	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/client"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// ─── VolumeAttachService (BMS path) ──────────────────────────────────────────

// RotateEncryptedSecret calls
// POST v1/bare_metal_servers/{serverID}/storage_access/encrypted_secret/rotate
// body: {"key":{"id":keyID}}
func (vs *VolumeAttachService) RotateEncryptedSecret(serverID, keyID string, ctxLogger *zap.Logger) error {
	methodName := "VolumeAttachService.RotateEncryptedSecret"
	defer util.TimeTracker(methodName, time.Now())
	defer metrics.UpdateDurationFromStart(ctxLogger, methodName, time.Now())

	operation := &client.Operation{
		Name:        "RotateEncryptedSecret",
		Method:      "POST",
		PathPattern: vs.pathPrefix + bmsRotateSecretSuffix,
	}

	reqBody := &models.BMSStorageAccessRotateRequest{
		Key: models.BMSKeyRef{ID: keyID},
	}

	apiErr := vs.receiverError
	operationRequest := vs.client.NewRequest(operation)

	ctxLogger.Info("Equivalent curl command and payload details",
		zap.Reflect("URL", operationRequest.URL()),
		zap.Reflect("Payload", reqBody),
		zap.Reflect("Operation", operation),
		zap.String("serverID", serverID))

	_, err := operationRequest.
		PathParameter(instanceIDParam, serverID).
		JSONBody(reqBody).
		JSONError(apiErr).
		Invoke()
	if err != nil {
		return err
	}

	ctxLogger.Info("RotateEncryptedSecret: succeeded", zap.String("serverID", serverID)) // pragma: allowlist secret
	return nil
}

// GetBMSServer calls
// GET v1/bare_metal_servers/{serverID}
// and returns the HOSTNQN + encrypted_secret in one round-trip.
func (vs *VolumeAttachService) GetBMSServer(serverID string, ctxLogger *zap.Logger) (*models.BMSServerResponse, error) {
	methodName := "VolumeAttachService.GetBMSServer"
	defer util.TimeTracker(methodName, time.Now())
	defer metrics.UpdateDurationFromStart(ctxLogger, methodName, time.Now())

	operation := &client.Operation{
		Name:        "GetBMSServer",
		Method:      "GET",
		PathPattern: vs.pathPrefix + bmsServerSuffix,
	}

	var server models.BMSServerResponse
	apiErr := vs.receiverError
	operationRequest := vs.client.NewRequest(operation)

	ctxLogger.Info("Equivalent curl command and payload details",
		zap.Reflect("URL", operationRequest.URL()),
		zap.Reflect("Operation", operation),
		zap.String("serverID", serverID))

	_, err := operationRequest.
		PathParameter(instanceIDParam, serverID).
		JSONSuccess(&server).
		JSONError(apiErr).
		Invoke()
	if err != nil {
		return nil, err
	}

	ctxLogger.Info("GetBMSServer: succeeded",
		zap.String("serverID", serverID),
		zap.String("hostNQN", server.NVMeQualifiedName),
		zap.Bool("encryptedSecretPresent", server.StorageAccess != nil && server.StorageAccess.EncryptedSecret != ""))
	return &server, nil
}

// ─── IKSVolumeAttachService stubs (BMS operations not supported via IKS) ─────

// RotateEncryptedSecret is not supported via the IKS path.
func (vs *IKSVolumeAttachService) RotateEncryptedSecret(serverID, keyID string, ctxLogger *zap.Logger) error {
	return fmt.Errorf("RotateEncryptedSecret is not supported via the IKS volume attach path")
}

// GetBMSServer is not supported via the IKS path.
func (vs *IKSVolumeAttachService) GetBMSServer(serverID string, ctxLogger *zap.Logger) (*models.BMSServerResponse, error) {
	return nil, fmt.Errorf("GetBMSServer is not supported via the IKS volume attach path")
}

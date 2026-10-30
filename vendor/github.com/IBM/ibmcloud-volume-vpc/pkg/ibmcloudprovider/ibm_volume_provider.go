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

// Package ibmcloudprovider ...
package ibmcloudprovider

import (
	"context"

	"github.com/IBM/ibmcloud-volume-interface/config"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	"go.uber.org/zap"
)

// CloudProviderInterface ...
type CloudProviderInterface interface {
	GetProviderSession(ctx context.Context, logger *zap.Logger) (provider.Session, error)
	GetConfig() *config.Config
	GetClusterID() string
	// GetBMSPrivKey returns the PEM-encoded RSA private key loaded at startup
	// from the k8s secret "storage-bm-secret" (key "rsa_private_key").
	// Returns nil on non-BMS nodes where the secret does not exist.
	GetBMSPrivKey() []byte
	// GetBMSKeyID returns the VPC SSH Key resource ID loaded at startup from
	// the k8s secret "storage-bm-secret" (key "key_id").
	// Used by the controller in STEP 4 to call the rotate API.
	// Returns empty string on non-BMS clusters.
	GetBMSKeyID() string
}

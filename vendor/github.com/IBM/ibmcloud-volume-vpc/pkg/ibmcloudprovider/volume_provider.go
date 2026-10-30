/**
 * Copyright 2024 IBM Corp.
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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/IBM/ibmcloud-volume-interface/config"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	"github.com/IBM/ibmcloud-volume-interface/provider/local"
	provider_util "github.com/IBM/ibmcloud-volume-vpc/block/utils"
	vpcconfig "github.com/IBM/ibmcloud-volume-vpc/block/vpcconfig"
	"github.com/IBM/ibmcloud-volume-vpc/common/registry"
	utilsConfig "github.com/IBM/secret-utils-lib/pkg/config"
	"github.com/IBM/secret-utils-lib/pkg/k8s_utils"
	"go.uber.org/zap"
)

const (
	// bmsSecretName is the k8s Secret created by the cluster admin that holds
	// the RSA key pair used for BMS NVMe-oF/TCP PSK encryption.
	//
	// Expected secret fields (create once per cluster):
	//
	//   key_id          – VPC SSH Key resource ID returned by:
	//                       ibmcloud is key-create <name> --key-material @rsa_public_key.pub
	//                     The controller passes this ID to the rotate API so VPC
	//                     knows which stored public key to encrypt the PSK with.
	//
	//   rsa_private_key – PEM-encoded RSA private key.  Read by the node
	//                     DaemonSet to RSA-OAEP decrypt the ciphertext received
	//                     via publishContext at NodeStageVolume time.
	//
	//   rsa_public_key  – PEM-encoded RSA public key (stored for reference only).
	//                     The driver does NOT read this field at runtime; it must
	//                     be registered with VPC via `ibmcloud is key-create`
	//                     before creating BMS volume attachments so VPC can
	//                     encrypt the PSK ciphertext.
	bmsSecretName = "storage-bm-secret"

	// bmsSecretPrivKeyField is the Secret data key that holds the
	// PEM-encoded RSA private key (used by the node DaemonSet only).
	bmsSecretPrivKeyField = "rsa_private_key"

	// bmsSecretKeyIDField is the Secret data key that holds the VPC SSH Key
	// resource ID used by the controller to call the rotate API.
	// The corresponding RSA public key must already be registered in VPC as an
	// SSH key with this ID so VPC can encrypt the PSK for this cluster.
	bmsSecretKeyIDField = "key_id"

	// bmsSecretPubKeyField names the public-key field stored in the secret for
	// operator reference.  The driver does not read this field; it is used only
	// during one-time VPC SSH Key registration.
	bmsSecretPubKeyField = "rsa_public_key" //nolint:deadcode,unused
)

// IBMCloudStorageProvider Provider
type IBMCloudStorageProvider struct {
	ProviderName   string
	ProviderConfig *config.Config
	Registry       registry.Providers
	ClusterID      string
	// BMSPrivKey is the PEM-encoded RSA private key read once at startup from
	// the k8s secret "storage-bm-secret".  Nil on non-BMS clusters where the
	// secret does not exist — all BMS code paths guard on this being non-nil.
	BMSPrivKey []byte
	// BMSKeyID is the VPC SSH Key resource ID read once at startup from the
	// k8s secret "storage-bm-secret" (key "key_id").
	// Used by the controller in STEP 4 to call the rotate API so VPC encrypts
	// the PSK with the cluster's RSA public key.
	// Empty on non-BMS clusters.
	BMSKeyID string
}

var _ CloudProviderInterface = &IBMCloudStorageProvider{}

// NewIBMCloudStorageProvider ...
func NewIBMCloudStorageProvider(clusterVolumeLabel string, k8sClient *k8s_utils.KubernetesClient, logger *zap.Logger) (*IBMCloudStorageProvider, error) {
	logger.Info("NewIBMCloudStorageProvider-Reading provider configuration...")
	// Load config file
	conf, err := config.ReadConfig(*k8sClient, logger)
	if err != nil {
		logger.Error("Error loading configuration")
		return nil, err
	}

	// Get only VPC_API_VERSION, in "2019-07-02T00:00:00.000Z" case vpc need only 2019-07-02"
	dateTime, err := time.Parse(time.RFC3339, conf.VPC.APIVersion)
	if err == nil {
		conf.VPC.APIVersion = fmt.Sprintf("%d-%02d-%02d", dateTime.Year(), dateTime.Month(), dateTime.Day())
	} else {
		logger.Warn("Failed to parse VPC_API_VERSION, setting default value")
		conf.VPC.APIVersion = "2022-11-11" // setting default values
	}

	var clusterInfo utilsConfig.ClusterConfig
	logger.Info("Fetching clusterInfo")
	if conf.IKS != nil && conf.IKS.Enabled || os.Getenv("IKS_ENABLED") == "True" {
		clusterInfo, err = utilsConfig.GetClusterInfo(*k8sClient, logger)
		if err != nil {
			logger.Error("Unable to load ClusterInfo", local.ZapError(err))
			return nil, err
		}
		logger.Info("Fetched clusterInfo..")
	}

	//Initialize the clusterVolumeLabel once which will be used for tagging by the library.
	conf.VPC.ClusterVolumeLabel = clusterVolumeLabel

	vpcBlockConfig := &vpcconfig.VPCBlockConfig{
		VPCConfig:    conf.VPC,
		IKSConfig:    conf.IKS,
		APIConfig:    conf.API,
		ServerConfig: conf.Server,
	}
	// Prepare provider registry
	registry, err := provider_util.InitProviders(vpcBlockConfig, k8sClient, logger)
	if err != nil {
		logger.Error("Error configuring providers", local.ZapError(err))
		return nil, err
	}

	var providerName string
	if isRunningInIKS() && conf.IKS.Enabled {
		providerName = conf.IKS.IKSBlockProviderName
	} else if conf.VPC.Enabled {
		providerName = conf.VPC.VPCBlockProviderName
	}

	// Read BMS credentials from "storage-bm-secret".  Both reads are non-fatal:
	// if the secret does not exist (non-BMS cluster) the fields stay at their
	// zero values and all BMS code paths are skipped automatically.
	//
	// rsa_private_key — node DaemonSet only: decrypt PSK at NodeStageVolume
	// key_id          — controller only:     passed in the rotate API body
	var bmsPrivKey []byte
	var bmsKeyID string

	if privKeyPEM, err := k8s_utils.GetSecretData(*k8sClient, bmsSecretName, bmsSecretPrivKeyField); err != nil {
		logger.Warn("NewIBMCloudStorageProvider: BMS private key not found — BMS NVMe attach will be disabled on node",
			zap.String("secret", bmsSecretName), zap.String("key", bmsSecretPrivKeyField), zap.Error(err))
	} else {
		bmsPrivKey = []byte(privKeyPEM)
		logger.Info("NewIBMCloudStorageProvider: BMS private key loaded successfully",
			zap.String("secret", bmsSecretName))
	}

	if keyID, err := k8s_utils.GetSecretData(*k8sClient, bmsSecretName, bmsSecretKeyIDField); err != nil {
		logger.Warn("NewIBMCloudStorageProvider: BMS key_id not found — encrypted_secret rotate will be disabled on controller",
			zap.String("secret", bmsSecretName), zap.String("key", bmsSecretKeyIDField), zap.Error(err))
	} else {
		bmsKeyID = keyID
		logger.Info("NewIBMCloudStorageProvider: BMS key_id loaded successfully",
			zap.String("secret", bmsSecretName))
	}

	cloudProvider := &IBMCloudStorageProvider{
		ProviderName:   providerName,
		ProviderConfig: conf,
		Registry:       registry,
		ClusterID:      clusterInfo.ClusterID,
		BMSPrivKey:     bmsPrivKey,
		BMSKeyID:       bmsKeyID,
	}
	logger.Info("Successfully read provider configuration")
	return cloudProvider, nil
}

func isRunningInIKS() bool {
	return true //TODO Check the master KUBE version
}

// GetProviderSession ...
func (icp *IBMCloudStorageProvider) GetProviderSession(ctx context.Context, logger *zap.Logger) (provider.Session, error) {
	logger.Info("IBMCloudStorageProvider-GetProviderSession...")
	if icp.ProviderConfig.API == nil {
		icp.ProviderConfig.API = &config.APIConfig{
			PassthroughSecret: string([]byte{}), // // TODO~ Need to remove it
		}
	}

	prov, err := icp.Registry.Get(icp.ProviderName)
	if err != nil {
		logger.Error("Not able to get the said provider, might be its not registered", local.ZapError(err))
		return nil, err
	}

	// Populating vpcBlockConfig which is used to open session
	vpcBlockConfig := &vpcconfig.VPCBlockConfig{
		VPCConfig:    icp.ProviderConfig.VPC,
		IKSConfig:    icp.ProviderConfig.IKS,
		APIConfig:    icp.ProviderConfig.API,
		ServerConfig: icp.ProviderConfig.Server,
	}

	session, _, err := provider_util.OpenProviderSessionWithContext(ctx, prov, vpcBlockConfig, icp.ProviderName, logger)
	if err == nil {
		logger.Info("Successfully got the provider session", zap.Reflect("ProviderName", session.ProviderName()))
		return session, nil
	}
	logger.Error("Failed to get provider session", zap.Reflect("Error", err))
	return nil, err
}

// GetConfig ...
func (icp *IBMCloudStorageProvider) GetConfig() *config.Config {
	return icp.ProviderConfig
}

// GetClusterID ...
func (icp *IBMCloudStorageProvider) GetClusterID() string {
	return icp.ClusterID
}

// GetBMSPrivKey returns the PEM-encoded RSA private key loaded at startup from
// the k8s secret "storage-bm-secret".  Returns nil on non-BMS nodes.
func (icp *IBMCloudStorageProvider) GetBMSPrivKey() []byte {
	return icp.BMSPrivKey
}

// GetBMSKeyID returns the VPC SSH Key resource ID loaded at startup from
// the k8s secret "storage-bm-secret".  Returns empty string on non-BMS clusters.
func (icp *IBMCloudStorageProvider) GetBMSKeyID() string {
	return icp.BMSKeyID
}

// CorrectEndpointURL corrects endpoint url if it is of form "http://"
func getEndpointURL(url string, logger *zap.Logger) string {
	if strings.Contains(url, "http://") {
		logger.Warn("Token exchange endpoint URL is of the form 'http' instead 'https'. Correcting it for valid request.", zap.Reflect("Endpoint URL: ", url))
		return strings.Replace(url, "http", "https", 1)
	}
	return url
}

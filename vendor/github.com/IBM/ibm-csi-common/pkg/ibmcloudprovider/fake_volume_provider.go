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

// Package ibmcloudprovider ...
package ibmcloudprovider

import (
	"bytes"
	"testing"

	"github.com/IBM/ibm-csi-common/pkg/utils"
	"github.com/IBM/ibmcloud-volume-interface/config"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider/fake"
	"github.com/IBM/ibmcloud-volume-interface/provider/local"
	provider_util "github.com/IBM/ibmcloud-volume-vpc/block/utils"
	vpcconfig "github.com/IBM/ibmcloud-volume-vpc/block/vpcconfig"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/context"
)

const (
	// TestProviderAccountID ...
	TestProviderAccountID = "test-provider-account"

	// TestProviderAccessToken ...
	TestProviderAccessToken = "test-provider-access-token"

	// TestIKSAccountID ...
	TestIKSAccountID = "test-iks-account"

	// TestZone ...
	TestZone = "test-zone"

	// IAMURL ...
	IAMURL = "test-iam-url"

	// IAMClientID ...
	IAMClientID = "test-iam_client_id"

	// IAMClientSecret ...
	IAMClientSecret = "test-iam_client_secret"

	// IAMAPIKey ...
	IAMAPIKey = "test-iam_api_key"

	// RefreshToken ...
	RefreshToken = "test-refresh_token"

	// TestEndpointURL ...
	TestEndpointURL = "http://some_endpoint"

	// TestAPIVersion ...
	TestAPIVersion = "2019-07-02"
)

// GetTestLogger ...
func GetTestLogger(t *testing.T) (logger *zap.Logger, teardown func()) {
	atom := zap.NewAtomicLevel()
	atom.SetLevel(zap.DebugLevel)

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	buf := &bytes.Buffer{}

	logger = zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.AddSync(buf),
			atom,
		),
		zap.AddCaller(),
	)

	teardown = func() {
		_ = logger.Sync()
		if t.Failed() {
			t.Log(buf)
		}
	}
	return
}

// GetTestProvider ...
func GetTestProvider(t *testing.T, logger *zap.Logger) (*IBMCloudStorageProvider, error) {
	logger.Info("GetTestProvider-Getting New test Provider")
	// vpcBlockConfig struct
	vpcBlockConfig := &vpcconfig.VPCBlockConfig{
		VPCConfig: &config.VPCProviderConfig{
			Enabled:              true,
			VPCBlockProviderName: "vpc-classic",
			EndpointURL:          TestEndpointURL,
			VPCTimeout:           "30s",
			MaxRetryAttempt:      5,
			MaxRetryGap:          10,
			APIVersion:           TestAPIVersion,
			IamClientID:          IAMClientID,
			IamClientSecret:      IAMClientSecret,
		},
		IKSConfig: &config.IKSConfig{
			Enabled:              true,
			IKSBlockProviderName: "iks-block",
		},
		APIConfig: &config.APIConfig{
			PassthroughSecret: "",
		},
		ServerConfig: &config.ServerConfig{
			DebugTrace: true,
		},
	}
	// full config struct
	conf := &config.Config{
		Server: &config.ServerConfig{
			DebugTrace: true,
		},
		Bluemix: &config.BluemixConfig{
			IamURL:          IAMURL,
			IamClientID:     IAMClientID,
			IamClientSecret: IAMClientSecret,
			IamAPIKey:       IAMClientSecret,
			RefreshToken:    RefreshToken,
		},
		VPC: &config.VPCProviderConfig{
			Enabled:              true,
			VPCBlockProviderName: "vpc-classic",
			EndpointURL:          TestEndpointURL,
			VPCTimeout:           "30s",
			MaxRetryAttempt:      5,
			MaxRetryGap:          10,
			APIVersion:           TestAPIVersion,
		},
		IKS: &config.IKSConfig{
			Enabled:              true,
			IKSBlockProviderName: "iks-block",
		},
	}

	// Prepare provider registry
	registry, err := provider_util.InitProviders(vpcBlockConfig, logger)
	if err != nil {
		logger.Fatal("Error configuring providers", local.ZapError(err))
	}

	cloudProvider := &IBMCloudStorageProvider{
		ProviderName:   "vpc-classic",
		ProviderConfig: conf,
		Registry:       registry,
		ClusterInfo:    nil,
	}
	logger.Info("Successfully read provider configuration...")
	return cloudProvider, nil
}

// FakeIBMCloudStorageProvider Provider
type FakeIBMCloudStorageProvider struct {
	ProviderName   string
	ProviderConfig *config.Config
	ClusterInfo    *utils.ClusterInfo
	fakeSession    *fake.FakeSession
}

var _ CloudProviderInterface = &FakeIBMCloudStorageProvider{}

// NewFakeIBMCloudStorageProvider ...
func NewFakeIBMCloudStorageProvider(configPath string, logger *zap.Logger) (*FakeIBMCloudStorageProvider, error) {
	return &FakeIBMCloudStorageProvider{ProviderName: "FakeIBMCloudStorageProvider",
		ProviderConfig: &config.Config{VPC: &config.VPCProviderConfig{VPCBlockProviderName: "VPCFakeProvider"}},
		ClusterInfo:    &utils.ClusterInfo{}, fakeSession: &fake.FakeSession{}}, nil
}

// GetProviderSession ...
func (ficp *FakeIBMCloudStorageProvider) GetProviderSession(ctx context.Context, logger *zap.Logger) (provider.Session, error) {
	return ficp.fakeSession, nil
}

// GetConfig ...
func (ficp *FakeIBMCloudStorageProvider) GetConfig() *config.Config {
	return ficp.ProviderConfig
}

// GetClusterInfo ...
func (ficp *FakeIBMCloudStorageProvider) GetClusterInfo() *utils.ClusterInfo {
	return ficp.ClusterInfo
}

// UpdateAPIKey ...
func (ficp *FakeIBMCloudStorageProvider) UpdateAPIKey(logger *zap.Logger) error {
	return nil
}

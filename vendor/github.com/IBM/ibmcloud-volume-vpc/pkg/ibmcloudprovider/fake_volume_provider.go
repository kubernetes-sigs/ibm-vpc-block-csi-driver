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
	"bytes"	
	"testing"
	"github.com/IBM/ibmcloud-volume-interface/config"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider/fake"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
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

// FakeIBMCloudStorageProvider Provider
type FakeIBMCloudStorageProvider struct {
	ProviderName   string
	ProviderConfig *config.Config
	ClusterID      string
	fakeSession    *fake.FakeSession
}

var _ CloudProviderInterface = &FakeIBMCloudStorageProvider{}

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
		err := logger.Sync()
		assert.Nil(t, err)

		if t.Failed() {
			t.Log(buf)
		}
	}

	return
}

// NewFakeIBMCloudStorageProvider ...
func NewFakeIBMCloudStorageProvider(configPath string, logger *zap.Logger) (*FakeIBMCloudStorageProvider, error) {
	return &FakeIBMCloudStorageProvider{ProviderName: "FakeIBMCloudStorageProvider",
		ProviderConfig: &config.Config{VPC: &config.VPCProviderConfig{VPCBlockProviderName: "VPCFakeProvider"}},
		ClusterID:      "fake-clusterID", fakeSession: &fake.FakeSession{}}, nil
}

// GetProviderSession ...
func (ficp *FakeIBMCloudStorageProvider) GetProviderSession(ctx context.Context, logger *zap.Logger) (provider.Session, error) {
	return ficp.fakeSession, nil
}

// GetConfig ...
func (ficp *FakeIBMCloudStorageProvider) GetConfig() *config.Config {
	return ficp.ProviderConfig
}

// GetClusterID ...
func (ficp *FakeIBMCloudStorageProvider) GetClusterID() string {
	return ficp.ClusterID
}

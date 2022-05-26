/**
 * Copyright 2022 IBM Corp.
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

package secret_provider

import (
	"os"
	"strings"

	sp "github.com/IBM/secret-utils-lib/pkg/secret_provider"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewSecretProvider initializes new secret provider
func NewSecretProvider() (sp.SecretProviderInterface, error) {
	var managed bool
	if iksEnabled := os.Getenv("IKS_ENABLED"); strings.ToLower(iksEnabled) == "true" {
		managed = true
	}
	logger := setUpLogger(managed)
	var secretprovider sp.SecretProviderInterface
	var err error
	if managed {
		secretprovider, err = newManagedSecretProvider(logger)
	} else {
		secretprovider, err = newUnmanagedSecretProvider(logger)
	}

	if err != nil {
		logger.Error("Error initializing secret provider", zap.Error(err))
		return nil, err
	}

	return secretprovider, nil
}

// setUpLogger ...
func setUpLogger(managed bool) *zap.Logger {
	// Prepare a new logger
	atom := zap.NewAtomicLevel()
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	var secretProviderType string
	if managed {
		secretProviderType = "managed-secret-provider"
	} else {
		secretProviderType = "unmanaged-secret-provider"
	}
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	), zap.AddCaller()).With(zap.String("name", "secret-provider")).With(zap.String("secret-provider-type", secretProviderType))

	atom.SetLevel(zap.InfoLevel)
	return logger
}

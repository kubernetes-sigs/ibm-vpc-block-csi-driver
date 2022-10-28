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

	localutils "github.com/IBM/secret-common-lib/pkg/utils"
	sp "github.com/IBM/secret-utils-lib/pkg/secret_provider"
	"github.com/IBM/secret-utils-lib/pkg/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	ProviderType string = "ProviderType"
	SecretKey    string = "SecretKey"
	VPC          string = "vpc"
	Bluemix      string = "bluemix"
	Softlayer    string = "softlayer"
)

// NewSecretProvider initializes new secret provider
// Note: providerType which can be VPC, Bluemix, Softlayer (the constants defined above) and is only used when we need to read storage-secret-store, this is kept to support backward compatibility.
func NewSecretProvider(optionalArgs ...map[string]string) (sp.SecretProviderInterface, error) {
	var managed bool
	if iksEnabled := os.Getenv("IKS_ENABLED"); strings.ToLower(iksEnabled) == "true" {
		managed = true
	}
	logger := setUpLogger(managed)

	err := validateArguments(optionalArgs...)
	if err != nil {
		logger.Error("Error seen while validating arguments", zap.Error(err), zap.Any("Provided arguments", optionalArgs))
		return nil, err
	}

	if managed { // If IKS_ENABLED is set to true
		if len(optionalArgs) == 0 {
			return newManagedSecretProvider(logger)
		}
		// If ProviderType is given, fetch providerName and pass to initialise managed secret provider
		if providerName, ok := optionalArgs[0][ProviderType]; ok {
			return newManagedSecretProvider(logger, providerName)
		}
	}

	// If a secret key was passed, or IKS ENABLED was set to false, initialise unmanaged secret provider
	return newUnmanagedSecretProvider(logger, optionalArgs...)
}

// validateArguments ...
func validateArguments(optionalArgs ...map[string]string) error {
	// Only one argument is expected
	if len(optionalArgs) > 1 {
		return utils.Error{Description: localutils.ErrMultipleKeysUnsupported}
	}

	if len(optionalArgs) == 1 {
		// If an argument is given and it is neither ProviderType nor SecretKey, return error
		providerName, providerExists := optionalArgs[0][ProviderType]
		secretKeyName, secretKeyExists := optionalArgs[0][SecretKey]
		if !providerExists && !secretKeyExists {
			return utils.Error{Description: localutils.ErrInvalidArgument}
		}

		// If secretKeyName is empty return error
		if secretKeyExists && secretKeyName == "" {
			return utils.Error{Description: localutils.ErrEmptySecretKeyProvided}
		}

		// If ProviderType is given, but it is invalid, return error
		if providerExists && !isProviderType(providerName) {
			return utils.Error{Description: localutils.ErrInvalidProviderType}
		}
	}

	return nil
}

// isProviderType ...
func isProviderType(arg string) bool {
	return (arg == VPC || arg == Bluemix || arg == Softlayer)
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

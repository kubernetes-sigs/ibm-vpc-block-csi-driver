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
	"github.com/IBM/secret-utils-lib/pkg/k8s_utils"
	sp "github.com/IBM/secret-utils-lib/pkg/secret_provider"
	"github.com/IBM/secret-utils-lib/pkg/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	ProviderType string = "ProviderType"
	SecretKey    string = "SecretKey"
	K8sClient    string = "K8sClient"
	VPC          string = "vpc"
	Bluemix      string = "bluemix"
	Softlayer    string = "softlayer"
)

// NewSecretProvider initializes new secret provider
// Note: providerType which can be VPC, Bluemix, Softlayer (the constants defined above) and is only used when we need to read storage-secret-store, this is kept to support backward compatibility.
func NewSecretProvider(optionalArgs ...map[string]interface{}) (sp.SecretProviderInterface, error) {
	var managed bool
	if iksEnabled := os.Getenv("IKS_ENABLED"); strings.ToLower(iksEnabled) == "true" {
		managed = true
	}
	logger := setUpLogger(managed)

	err := validateArguments(logger, optionalArgs...)
	if err != nil {
		logger.Error("Error seen while validating arguments", zap.Error(err), zap.Any("Provided arguments", optionalArgs))
		return nil, err
	}

	var secretKeyExists bool
	if len(optionalArgs) > 0 {
		_, secretKeyExists = optionalArgs[0][SecretKey]
	}

	// If IKS_ENABLED is set to true, and client has not passed any secret key, init managed secret provider
	if managed && !secretKeyExists {
		return newManagedSecretProvider(logger, optionalArgs...)
	}

	// If a secret key was passed, or IKS ENABLED was set to false, initialise unmanaged secret provider
	return newUnmanagedSecretProvider(logger, optionalArgs...)
}

// validateArguments ...
func validateArguments(logger *zap.Logger, optionalArgs ...map[string]interface{}) error {
	// Only one argument is expected
	if len(optionalArgs) > 1 {
		return utils.Error{Description: localutils.ErrMultipleArgsUnsupported}
	}

	if len(optionalArgs) == 1 {
		// If an argument is given and it is neither ProviderType nor SecretKey, return error
		providerNameInterface, providerExists := optionalArgs[0][ProviderType]
		secretKeyNameInterface, secretKeyExists := optionalArgs[0][SecretKey]
		k8sClientInterface, k8sClientExists := optionalArgs[0][K8sClient]

		if !secretKeyExists && !providerExists && !k8sClientExists {
			return utils.Error{Description: localutils.ErrInvalidArgument}
		}

		if secretKeyExists && !isSecretKey(logger, secretKeyNameInterface) {
			return utils.Error{Description: localutils.ErrInvalidSecretKey}
		}

		// If secretKeyName is empty return error
		if providerExists && !isProviderType(logger, providerNameInterface) {
			return utils.Error{Description: localutils.ErrInvalidProviderType}
		}

		// If ProviderType is given, but it is invalid, return error
		if k8sClientExists && !isK8SClient(logger, k8sClientInterface) {
			return utils.Error{Description: localutils.ErrInvalidK8sClient}
		}
	}

	return nil
}

// isProviderType ...
func isProviderType(logger *zap.Logger, arg interface{}) bool {
	providerType, valid := arg.(string)
	if valid {
		logger.Info("provider type", zap.String("Provider type", providerType))
		return providerType == VPC || providerType == Bluemix || providerType == Softlayer
	}

	logger.Error("Provider type is not of type string", zap.Any("Provider type", arg))
	return false
}

// isSecretKey ...
func isSecretKey(logger *zap.Logger, arg interface{}) bool {
	secretKey, valid := arg.(string)
	if valid && secretKey != "" {
		return true
	}

	logger.Error("Secret key is either empty or not of type string", zap.Any("Secret key", arg))
	return false
}

// isK8SClient ...
func isK8SClient(logger *zap.Logger, arg interface{}) bool {
	k8sClient, valid := arg.(k8s_utils.KubernetesClient)
	if !valid {
		logger.Error("Provided K8S client is not of type KubernetesClient")
		return false
	}

	if k8sClient.GetNameSpace() == "" {
		logger.Error("Namespace in the kubernetes client is not initialised")
		return false
	}

	if k8sClient.GetClientSet() == nil {
		logger.Error("K8S clientset is not initialised")
		return false
	}

	return true
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

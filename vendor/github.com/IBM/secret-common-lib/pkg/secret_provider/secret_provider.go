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
	"os/exec"
	"strings"
	"time"

	localutils "github.com/IBM/secret-common-lib/pkg/utils"
	"github.com/IBM/secret-utils-lib/pkg/k8s_utils"
	sp "github.com/IBM/secret-utils-lib/pkg/secret_provider"
	"github.com/IBM/secret-utils-lib/pkg/utils"
	"github.com/beevik/ntp"
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
// argument1: k8sClient - this is the k8s client which holds k8s clientset and namespace which the client code must pass, if they are intending to use only unmanaged secret provider.
// argument2: optionalArgs - in this map, two keys can be provided - 1. providerType which can be VPC, Bluemix, Softlayer (the constants defined above) and is only used when we need to read storage-secret-store, this is kept to support backward compatibility.
// and 2. SecretKey which is given, when different keys other than the default needs to be referred. (Defaults are slclient.toml in storage-secret-store, ibm-credetentials.env in ibm-cloud-credentials.)
func NewSecretProvider(k8sClient *k8s_utils.KubernetesClient, optionalArgs ...map[string]string) (sp.SecretProviderInterface, error) {
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

	if err := setNTPTime(logger); err != nil {
		logger.Warn("Unable to set current time according to remote NTP server")
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
	return newUnmanagedSecretProvider(k8sClient, logger, optionalArgs...)
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

// setNTPTime queries the ntp server for the approriate current time.
// if there is a difference between current time of the system and retrieved time from ntp, current time is reset to the retrieved time
// this is avoid any clock skew errors which can possibly be observed while validating jwt tokens.
func setNTPTime(logger *zap.Logger) error {
	options := ntp.QueryOptions{Timeout: 30 * time.Second, TTL: 5}
	response, err := ntp.QueryWithOptions("0.beevik-ntp.pool.ntp.org", options)
	if err != nil {
		return err
	}

	currentTime := time.Now().Unix()
	if currentTime == response.Time.Unix() {
		logger.Info("Current and ntp requested time is same")
		return nil
	}
	logger.Info("Current time v/s ntp time", zap.Int64("current time", currentTime), zap.Int64("time received from ntp server", response.Time.Unix()))

	currentDate, err := exec.LookPath("date")
	if err != nil {
		logger.Warn("Date binary not found, cannot set system date:", zap.Error(err))
		return err
	}
	logger.Info("Current time and date", zap.String("date", currentDate))

	dateString := response.Time.Format(currentDate)
	logger.Info("Setting system date", zap.String("date", dateString))
	args := []string{"--set", dateString}
	return exec.Command("date", args...).Run()

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

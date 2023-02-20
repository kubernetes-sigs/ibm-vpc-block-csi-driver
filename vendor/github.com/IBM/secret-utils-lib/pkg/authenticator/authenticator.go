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

// Package authenticator ...
package authenticator

import (
	"fmt"
	"strings"

	"github.com/IBM/secret-utils-lib/pkg/config"
	"github.com/IBM/secret-utils-lib/pkg/k8s_utils"
	"github.com/IBM/secret-utils-lib/pkg/utils"
	"go.uber.org/zap"
)

const (
	ProviderType string = "ProviderType"
	SecretKey    string = "SecretKey"
)

// Authenticator ...
type Authenticator interface {
	GetToken(freshTokenRequired bool) (string, uint64, error)
	GetSecret() string
	SetSecret(secret string)
	SetURL(url string)
	SetEncryption(bool)
	IsSecretEncrypted() bool
	getURL() string
}

// NewAuthenticator initializes the particular authenticator based on the configuration provided.
func NewAuthenticator(logger *zap.Logger, kc k8s_utils.KubernetesClient, optionalArgs ...map[string]string) (Authenticator, string, error) {
	logger.Info("Initializing authenticator")

	// Check if secretKey or providerType is provided
	var secretKeyName, providerName string
	var secretKeyExists, providerExists bool
	if len(optionalArgs) != 0 {
		secretKeyName, secretKeyExists = optionalArgs[0][SecretKey]
		providerName, providerExists = optionalArgs[0][ProviderType]
	}

	// If a secretKey (key in the k8s secret) is provided, first look for the key in ibm-cloud-credentials
	// If it is not found ibm-cloud-credentials, look for it in storage-secret-store
	// If it is not found in either of the secrets, return error
	if secretKeyExists {
		logger.Info("Key provided", zap.String("Key", secretKeyName))
		data, err := k8s_utils.GetSecretData(kc, utils.IBMCLOUD_CREDENTIALS_SECRET, secretKeyName)
		if err == nil {
			return initAuthenticatorForIBMCloudCredentials(logger, data)
		}

		logger.Warn("Unable to fetch ibm-cloud-credentials, fetching from storage-secret-store", zap.Error(err))
		data, err = k8s_utils.GetSecretData(kc, utils.STORAGE_SECRET_STORE_SECRET, secretKeyName)
		if err != nil {
			logger.Error("Error initializing authenticator", zap.Error(err))
			return nil, "", err
		}
		logger.Info("Initialized authenticator", zap.String("secret-name", utils.STORAGE_SECRET_STORE_SECRET), zap.String("key-name", secretKeyName))
		return NewIamAuthenticator(data, logger), utils.DEFAULT, nil
	}

	// If the secretKey is not provided,
	// Read ibm-credentials.env key from ibm-cloud-credentials
	data, err := k8s_utils.GetSecretData(kc, utils.IBMCLOUD_CREDENTIALS_SECRET, utils.CLOUD_PROVIDER_ENV)
	if err == nil {
		return initAuthenticatorForIBMCloudCredentials(logger, data)
	}

	// If ibm-cloud-credentials does not exist, read slclient.toml from storage-secret-store
	logger.Warn("Unable to fetch ibm-cloud-credentials", zap.Error(err), zap.String("key-name", utils.CLOUD_PROVIDER_ENV))
	data, err = k8s_utils.GetSecretData(kc, utils.STORAGE_SECRET_STORE_SECRET, utils.SECRET_STORE_FILE)
	if err != nil {
		logger.Error("Error initializing authenticator", zap.Error(err))
		return nil, "", err
	}

	// If providerType is given, check for the same in storage secret store
	if providerExists {
		return initAuthenticatorForStorageSecretStore(logger, providerName, data)
	}
	return initAuthenticatorForStorageSecretStore(logger, utils.VPC, data)
}

// isProviderType ...
func isProviderType(arg string) bool {
	return (arg == utils.VPC || arg == utils.Bluemix || arg == utils.Softlayer)
}

// initAuthenticatorForIBMCloudCredentials ...
func initAuthenticatorForIBMCloudCredentials(logger *zap.Logger, data string) (Authenticator, string, error) {
	credentialsmap, err := parseIBMCloudCredentials(logger, data)
	if err != nil {
		logger.Error("Error parsing credentials", zap.Error(err))
		return nil, "", err
	}

	var authenticator Authenticator
	var defaultSecret string
	credentialType := credentialsmap[utils.IBMCLOUD_AUTHTYPE]
	switch credentialType {
	case utils.IAM:
		defaultSecret = credentialsmap[utils.IBMCLOUD_APIKEY]
		authenticator = NewIamAuthenticator(defaultSecret, logger)
	case utils.PODIDENTITY:
		defaultSecret = credentialsmap[utils.IBMCLOUD_PROFILEID]
		authenticator = NewComputeIdentityAuthenticator(defaultSecret, logger)
	}

	logger.Info("Successfully initialized authenticator", zap.String("secret-name", utils.IBMCLOUD_CREDENTIALS_SECRET), zap.String("auth-type", credentialType))
	return authenticator, credentialType, nil
}

// initAuthenticatorForStorageSecretStore ...
func initAuthenticatorForStorageSecretStore(logger *zap.Logger, providerName, data string) (Authenticator, string, error) {
	conf, err := config.ParseConfig(logger, data)
	if err != nil {
		logger.Error("Error parsing config", zap.Error(err))
		return nil, "", err
	}

	var encryption bool
	var apiKey string
	switch providerName {
	case utils.VPC:
		encryption = conf.VPC.Encryption
		apiKey = conf.VPC.G2APIKey
	case utils.Bluemix:
		encryption = conf.Bluemix.Encryption
		apiKey = conf.Bluemix.IamAPIKey
	case utils.Softlayer:
		apiKey = conf.Softlayer.SoftlayerAPIKey
	default:
		return nil, "", utils.Error{Description: utils.ErrInvalidProviderType}
	}

	if apiKey == "" {
		logger.Error("Empty api key read from the secret", zap.String("provider", providerName))
		return nil, "", utils.Error{Description: utils.ErrAPIKeyNotProvided}
	}

	authenticator := NewIamAuthenticator(apiKey, logger)
	authenticator.SetEncryption(encryption)
	logger.Info("Successfully initialized authenticator", zap.String("secret-name", utils.STORAGE_SECRET_STORE_SECRET), zap.String("auth-type", utils.DEFAULT))
	return authenticator, utils.DEFAULT, nil
}

// parseIBMCloudCredentials: parses the given data into key value pairs
// a map of credentials.
func parseIBMCloudCredentials(logger *zap.Logger, data string) (map[string]string, error) {

	credentials := strings.Split(data, "\n")
	credentialsmap := make(map[string]string)
	for _, credential := range credentials {
		if credential == "" {
			continue
		}
		// Parse the property string into name and value tokens
		var tokens = strings.SplitN(credential, "=", 2)
		if len(tokens) == 2 {
			// Store the name/value pair in the map.
			credentialsmap[tokens[0]] = tokens[1]
		}
	}

	if len(credentialsmap) == 0 {
		logger.Error("Credentials provided are not in the expected format")
		return nil, utils.Error{Description: utils.ErrInvalidCredentialsFormat}
	}

	// validating credentials
	credentialType, ok := credentialsmap[utils.IBMCLOUD_AUTHTYPE]
	if !ok {
		logger.Error("IBMCLOUD_AUTHTYPE is undefined, expected - IAM or PODIDENTITY")
		return nil, utils.Error{Description: utils.ErrAuthTypeUndefined}
	}

	if credentialType != utils.IAM && credentialType != utils.PODIDENTITY {
		logger.Error("Credential type provided is unknown", zap.String("Credential type", credentialType))
		return nil, utils.Error{Description: fmt.Sprintf(utils.ErrUnknownCredentialType, credentialType)}
	}

	if credentialType == utils.IAM {
		if secret, ok := credentialsmap[utils.IBMCLOUD_APIKEY]; !ok || secret == "" {
			logger.Error("API key is empty")
			return nil, utils.Error{Description: utils.ErrAPIKeyNotProvided}
		}
	}

	if credentialType == utils.PODIDENTITY {
		if secret, ok := credentialsmap[utils.IBMCLOUD_PROFILEID]; !ok || secret == "" {
			logger.Error("Profile ID is empty")
			return nil, utils.Error{Description: utils.ErrProfileIDNotProvided}
		}
	}

	return credentialsmap, nil
}

// isTimeout ...
func isTimeout(err error) bool {
	// If the error message contains "Client.Timeout" substring, return true
	if strings.Contains(err.Error(), "Client.Timeout") {
		return true
	}
	return false
}

// resetURL resets URL from private IAM url to public IAM url ...
func resetIAMURL(auth Authenticator) bool {
	if strings.Contains(auth.getURL(), utils.ProdPrivateIAMURL) {
		auth.SetURL(utils.ProdPublicIAMURL + "/identity/token")
		return true
	}
	if strings.Contains(auth.getURL(), utils.StagePrivateIAMURL) {
		auth.SetURL(utils.StagePublicIAMURL + "/identity/token")
		return true
	}
	return false
}

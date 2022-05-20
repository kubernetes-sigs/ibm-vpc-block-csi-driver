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
	"encoding/base64"
	"os"
	"strings"

	localutils "github.com/IBM/secret-common-lib/pkg/utils"
	auth "github.com/IBM/secret-utils-lib/pkg/authenticator"
	"github.com/IBM/secret-utils-lib/pkg/config"
	"github.com/IBM/secret-utils-lib/pkg/k8s_utils"
	"github.com/IBM/secret-utils-lib/pkg/utils"
	"go.uber.org/zap"
)

// UnmanagedSecretProvider ...
type UnmanagedSecretProvider struct {
	authenticator    auth.Authenticator
	logger           *zap.Logger
	authType         string
	tokenExchangeURL string
}

// newUnmanagedSecretProvider ...
func newUnmanagedSecretProvider(logger *zap.Logger) (*UnmanagedSecretProvider, error) {
	logger.Info("Initliazing unmanaged secret provider")
	kc, err := k8s_utils.Getk8sClientSet(logger)
	if err != nil {
		logger.Info("Error fetching k8s client set", zap.Error(err))
		return nil, err
	}
	return initUnmanagedSecretProvider(logger, kc)
}

// initUnmanagedSecretProvider ...
func initUnmanagedSecretProvider(logger *zap.Logger, kc k8s_utils.KubernetesClient) (*UnmanagedSecretProvider, error) {
	authenticator, authType, err := auth.NewAuthenticator(logger, kc)
	if err != nil {
		logger.Error("Error initializing unmanaged secret provider", zap.Error(err))
		return nil, err
	}

	tokenExchangeURL := config.FrameTokenExchangeURL(kc, logger)
	authenticator.SetURL(tokenExchangeURL)

	if authenticator.IsSecretEncrypted() {
		logger.Error("Secret is encrypted, decryption is only supported by sidecar container")
		return nil, utils.Error{Description: localutils.ErrDecryptionNotSupported}
	}

	// Checking if the secret(api key) needs to be decoded
	if authType == utils.DEFAULT && os.Getenv("IS_SATELLITE") == "True" {
		logger.Info("Decoding apiKey since it's a satellite cluster")
		decodedSecret, err := base64.StdEncoding.DecodeString(authenticator.GetSecret())
		if err != nil {
			logger.Error("Error decoding the secret", zap.Error(err))
			return nil, err
		}
		// In the decoded secret, newline could be present, trimming the same to extract a valid api key.
		authenticator.SetSecret(strings.TrimSuffix(string(decodedSecret), "\n"))
	}
	logger.Info("Initliazed unmanaged secret provider")
	return &UnmanagedSecretProvider{authenticator: authenticator, logger: logger, authType: authType, tokenExchangeURL: tokenExchangeURL}, nil
}

// GetDefaultIAMToken ...
func (usp *UnmanagedSecretProvider) GetDefaultIAMToken(isFreshTokenRequired bool) (string, uint64, error) {
	usp.logger.Info("Fetching IAM token for default secret")
	return usp.authenticator.GetToken(true)
}

// GetIAMToken ...
func (usp *UnmanagedSecretProvider) GetIAMToken(secret string, isFreshTokenRequired bool) (string, uint64, error) {
	usp.logger.Info("Fetching IAM token the provided secret")
	var authenticator auth.Authenticator
	switch usp.authType {
	case utils.IAM, utils.DEFAULT:
		authenticator = auth.NewIamAuthenticator(secret, usp.logger)
	case utils.PODIDENTITY:
		authenticator = auth.NewComputeIdentityAuthenticator(secret, usp.logger)
	}

	authenticator.SetURL(usp.tokenExchangeURL)
	token, tokenlifetime, err := authenticator.GetToken(true)
	if err != nil {
		usp.logger.Error("Error fetching IAM token", zap.Error(err))
		return token, tokenlifetime, err
	}
	usp.logger.Info("Successfully fetched IAM token for the provided secret")
	return token, tokenlifetime, nil
}

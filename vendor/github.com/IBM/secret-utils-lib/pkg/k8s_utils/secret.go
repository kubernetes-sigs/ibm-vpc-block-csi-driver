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

// Package k8s_utils ...
package k8s_utils

import (
	"context"
	b64 "encoding/base64"
	"fmt"

	"github.com/IBM/secret-utils-lib/pkg/utils"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetSecretData ...
func GetSecretData(kc KubernetesClient) (string, string, error) {
	kc.logger.Info("Trying to fetch ibm-cloud-credentials secret")

	data, err := GetSecret(kc, utils.IBMCLOUD_CREDENTIALS_SECRET, utils.CLOUD_PROVIDER_ENV)
	if err == nil {
		kc.logger.Info("Successfully fetched secret data", zap.String("Secret", utils.IBMCLOUD_CREDENTIALS_SECRET))
		return data, utils.IBMCLOUD_CREDENTIALS_SECRET, nil
	}

	kc.logger.Error("Unable to find secret", zap.Error(err), zap.String("Secret name", utils.IBMCLOUD_CREDENTIALS_SECRET))
	kc.logger.Info("Trying to fetch storage-secret-store secret")
	data, err = GetSecret(kc, utils.STORAGE_SECRET_STORE_SECRET, utils.SECRET_STORE_FILE)
	if err != nil {
		kc.logger.Error("Unable to find secret", zap.Error(err), zap.String("Secret name", utils.STORAGE_SECRET_STORE_SECRET))
		return "", "", err
	}

	kc.logger.Info("Successfully fetched secret data", zap.String("Secret", utils.STORAGE_SECRET_STORE_SECRET))
	return data, utils.STORAGE_SECRET_STORE_SECRET, nil
}

// GetSecret ...
func GetSecret(kc KubernetesClient, secretname, dataname string) (string, error) {
	kc.logger.Info("Fetching config map")

	clientset := kc.GetClientSet()
	namespace := kc.GetNameSpace()

	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretname, v1.GetOptions{})
	if err != nil {
		kc.logger.Error("Error fetching secret", zap.Error(err), zap.String("secret-name", secretname))
		return "", utils.Error{Description: utils.ErrFetchingSecrets, BackendError: err.Error()}
	}

	if secret.Data == nil {
		kc.logger.Error("No data found in the secret")
		return "", utils.Error{Description: fmt.Sprintf(utils.ErrEmptyDataInSecret, secretname)}
	}

	byteData, ok := secret.Data[dataname]
	if !ok {
		kc.logger.Error("Expected data not present in the secret", zap.String("secret-name", secretname), zap.String("dataname", dataname))
		return "", utils.Error{Description: fmt.Sprintf(utils.ErrExpectedDataNotFound, dataname, secretname)}
	}

	sEnc := b64.StdEncoding.EncodeToString(byteData)

	sDec, err := b64.StdEncoding.DecodeString(sEnc)
	if err != nil {
		kc.logger.Error("Error decoding the secret data", zap.Error(err), zap.String("secret-name", secretname), zap.String("data-name", dataname))
		return "", utils.Error{Description: fmt.Sprintf(utils.ErrFetchingSecretData, secretname, dataname), BackendError: err.Error()}
	}

	kc.logger.Info("Successfully fetched secret data")
	return string(sDec), nil
}

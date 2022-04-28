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
	"io/ioutil"

	"github.com/IBM/secret-utils-lib/pkg/utils"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// nameSpacePath is the path from which namespace where the pod is running is obtained.
	nameSpacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

// Getk8sClientSet ...
func Getk8sClientSet(logger *zap.Logger) (*KubernetesClient, error) {
	logger.Info("Fetching k8s clientset")

	// Fetching cluster config used to create k8s client
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		logger.Error("Error fetching in cluster config", zap.Error(err))
		return nil, utils.Error{Description: utils.ErrFetchingClusterConfig, BackendError: err.Error()}
	}

	// Creating k8s client used to read secret
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		logger.Error("Error creating k8s client", zap.Error(err))
		return nil, utils.Error{Description: utils.ErrFetchingClusterConfig, BackendError: err.Error()}
	}
	logger.Info("Successfully fetched k8s client set")

	// Reading the namespace in which the pod is deployed
	logger.Info("Fetching namespace")
	byteData, err := ioutil.ReadFile(nameSpacePath)
	if err != nil {
		logger.Error("Error fetching namespace", zap.Error(err))
		return nil, utils.Error{Description: utils.ErrFetchingNamespace, BackendError: err.Error()}
	}

	namespace := string(byteData)
	if namespace == "" {
		logger.Error("Unable to fetch namespace", zap.Error(err))
		return nil, utils.Error{Description: utils.ErrFetchingNamespace, BackendError: "namespace empty"}
	}
	logger.Info("Successfully fetched namespace")

	return &KubernetesClient{logger: logger, clientset: clientset, namespace: namespace}, nil
}

// GetCredentials ...
func GetSecretData(kc *KubernetesClient) (string, string, error) {
	kc.logger.Info("Trying to fetch ibm-cloud-credentials secret")

	namespace := kc.GetNameSpace()
	clientset := kc.clientset
	var dataname string
	var secretname string
	// Fetching secret
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), utils.IBMCLOUD_CREDENTIALS_SECRET, v1.GetOptions{})
	if err == nil {
		dataname = utils.CLOUD_PROVIDER_ENV
		secretname = utils.IBMCLOUD_CREDENTIALS_SECRET
	} else {
		kc.logger.Error("Unable to find secret", zap.Error(err), zap.String("Secret name", utils.IBMCLOUD_CREDENTIALS_SECRET))
		kc.logger.Info("Trying to fetch storage-secret-store secret")
		secret, err = clientset.CoreV1().Secrets(namespace).Get(context.TODO(), utils.STORAGE_SECRET_STORE_SECRET, v1.GetOptions{})
		if err != nil {
			kc.logger.Error("Unable to find secret", zap.Error(err), zap.String("Secret name", utils.STORAGE_SECRET_STORE_SECRET))
			return "", "", utils.Error{Description: utils.ErrFetchingSecrets, BackendError: err.Error()}
		}
		dataname = utils.SECRET_STORE_FILE
		secretname = utils.STORAGE_SECRET_STORE_SECRET
	}

	if secret.Data == nil {
		kc.logger.Error("No data found in the secret")
		return "", "", utils.Error{Description: fmt.Sprintf(utils.ErrEmptyDataInSecret, secretname)}
	}

	byteData, ok := secret.Data[dataname]
	if !ok {
		kc.logger.Error("Expected data not found in the secret")
		return "", "", utils.Error{Description: fmt.Sprintf(utils.ErrExpectedDataNotFound, dataname, secretname)}
	}

	sEnc := b64.StdEncoding.EncodeToString(byteData)

	sDec, err := b64.StdEncoding.DecodeString(sEnc)
	if err != nil {
		kc.logger.Error("Error decoding the secret data", zap.Error(err), zap.String("Secret name", secretname), zap.String("Data name", dataname))
		return "", "", utils.Error{Description: fmt.Sprintf(utils.ErrFetchingSecretData, secretname, dataname), BackendError: err.Error()}
	}

	kc.logger.Info("Successfully fetched secret data")
	return string(sDec), secretname, nil
}

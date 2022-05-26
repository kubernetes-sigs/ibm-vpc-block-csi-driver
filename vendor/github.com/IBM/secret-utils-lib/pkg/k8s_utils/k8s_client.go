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
	"io/ioutil"

	"github.com/IBM/secret-utils-lib/pkg/utils"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// nameSpacePath is the path from which namespace where the pod is running is obtained.
	nameSpacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

// KubernetesClient ...
type KubernetesClient struct {
	namespace string
	logger    *zap.Logger
	clientset kubernetes.Interface
}

// GetNameSpace ...
func (kc KubernetesClient) GetNameSpace() string {
	return kc.namespace
}

// GetClientSet ...
func (kc KubernetesClient) GetClientSet() kubernetes.Interface {
	return kc.clientset
}

// Getk8sClientSet ...
func Getk8sClientSet(logger *zap.Logger) (KubernetesClient, error) {

	var kc KubernetesClient
	// Fetching cluster config used to create k8s client
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		logger.Error("Error fetching in cluster config", zap.Error(err))
		return kc, utils.Error{Description: utils.ErrFetchingK8sClusterConfig, BackendError: err.Error()}
	}

	// Creating k8s client used to read secret
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		logger.Error("Error creating k8s client", zap.Error(err))
		return kc, utils.Error{Description: utils.ErrFetchingK8sClusterConfig, BackendError: err.Error()}
	}

	namespace, err := getNameSpace(logger)
	if err != nil {
		logger.Error("Error fetching namespace", zap.Error(err))
		return kc, err
	}

	kc.clientset = clientset
	kc.logger = logger
	kc.namespace = namespace
	return kc, nil
}

// getNameSpace ...
func getNameSpace(logger *zap.Logger) (string, error) {
	// Reading the namespace in which the pod is deployed
	byteData, err := ioutil.ReadFile(nameSpacePath)
	if err != nil {
		logger.Error("Error fetching namespace", zap.Error(err))
		return "", utils.Error{Description: utils.ErrFetchingNamespace, BackendError: err.Error()}
	}

	namespace := string(byteData)
	if namespace == "" {
		logger.Error("Unable to fetch namespace", zap.Error(err))
		return "", utils.Error{Description: utils.ErrFetchingNamespace, BackendError: "namespace empty"}
	}

	return namespace, nil
}

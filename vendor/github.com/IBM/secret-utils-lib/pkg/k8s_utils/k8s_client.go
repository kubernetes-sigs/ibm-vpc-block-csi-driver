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
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

// KubernetesClient ...
type KubernetesClient struct {
	namespace string
	logger    *zap.Logger
	clientset kubernetes.Interface
}

// GetNameSpace ...
func (kc *KubernetesClient) GetNameSpace() string {
	kc.logger.Info("Fetching namespace")
	return kc.namespace
}

// GetClientSet ...
func (kc *KubernetesClient) GetClientSet() kubernetes.Interface {
	return kc.clientset
}

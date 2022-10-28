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
	"fmt"
	"strings"

	"github.com/IBM/secret-utils-lib/pkg/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetSecretData ...
func GetSecretData(kc KubernetesClient, secretName, secretKey string) (string, error) {
	namespace := kc.GetNameSpace()
	clientset := kc.clientset

	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, v1.GetOptions{})
	if err != nil {
		return "", err
	}

	if secret.Data == nil {
		return "", utils.Error{Description: fmt.Sprintf(utils.ErrEmptyDataInSecret, secretName)}
	}

	byteData, ok := secret.Data[secretKey]
	if !ok {
		return "", utils.Error{Description: fmt.Sprintf(utils.ErrExpectedDataNotFound, secretKey, secretName)}
	}

	return strings.TrimSuffix(string(byteData), "\n"), nil
}

/**
 * Copyright 2020 IBM Corp.
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

// Package registry ...
package registry

import (
	//"github.com/prometheus/client_golang/prometheus"
	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/IBM/ibmcloud-volume-interface/provider/local"
)

// Providers is a registry interface for IaaS providers
//
//go:generate counterfeiter -o fakes/provider_registry.go --fake-name Providers . Providers
type Providers interface {
	Get(providerID string) (local.Provider, error)
	Register(providerID string, prov local.Provider)
}

var _ Providers = &ProviderRegistry{}

// ProviderRegistry is the core implementation of the Providers registry
type ProviderRegistry struct {
	providers map[string]local.Provider
}

// Get returns the identified Provider
func (pr *ProviderRegistry) Get(providerID string) (prov local.Provider, err error) {
	prov = pr.providers[providerID]
	if prov == nil {
		err = util.NewError("ErrorUnclassified", "Provider unknown: "+providerID)
	}
	return
}

// Register registers a given provider under the supplied key
func (pr *ProviderRegistry) Register(providerID string, p local.Provider) {
	if pr.providers == nil {
		pr.providers = map[string]local.Provider{}
	}
	pr.providers[providerID] = p
}

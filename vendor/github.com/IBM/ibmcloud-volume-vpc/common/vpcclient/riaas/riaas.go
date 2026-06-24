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

// Package riaas ...
package riaas

import (
	"context"
	"net/url"
	"strconv"

	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/client"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/instances"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/vpcvolume"
)

// RegionalAPI is the main interface for the RIAAS API client. From here, service
// objects for the individual parts of the API can be obtained
//
//go:generate counterfeiter -o fakes/regional_api.go --fake-name RegionalAPI . RegionalAPI
type RegionalAPI interface {
	Login(token string) error

	VolumeService() vpcvolume.VolumeManager
	VolumeAttachService() instances.VolumeAttachManager
	IKSVolumeAttachService() instances.VolumeAttachManager
	SnapshotService() vpcvolume.SnapshotManager
}

var _ RegionalAPI = &Session{}

// Session is a base implementation of the RegionalAPI interface
type Session struct {
	client client.SessionClient
	config Config
}

// New creates a new Session volume, using the supplied config
func New(config Config) (*Session, error) {
	ctx := config.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Default API version
	backendAPIVersion := models.APIVersion

	// Overwrite if the version is passed
	if len(config.APIVersion) > 0 {
		backendAPIVersion = config.APIVersion
	}

	// Overwrite if the generation is passed
	apiGen := models.APIGeneration
	if config.APIGeneration > 0 {
		apiGen = config.APIGeneration
	}

	queryValues := url.Values{
		"version":    []string{backendAPIVersion},
		"generation": []string{strconv.Itoa(apiGen)},
	}

	riaasClient := client.New(ctx, config.baseURL(), queryValues, config.httpClient(), config.ContextID, config.ResourceGroup)

	if config.DebugWriter != nil {
		riaasClient.WithDebug(config.DebugWriter)
	}
	return &Session{
		client: riaasClient,
		config: config,
	}, nil
}

// Login configures the session with the supplied Authentication token
// which is used for all requests to the API
func (s *Session) Login(token string) error {
	s.client.WithAuthToken(token)
	return nil
}

// VolumeService returns the Volume service for managing volumes
func (s *Session) VolumeService() vpcvolume.VolumeManager {
	return vpcvolume.New(s.client)
}

// VolumeAttachService returns the VolumeAttachService for managing volumes
func (s *Session) VolumeAttachService() instances.VolumeAttachManager {
	return instances.New(s.client)
}

// IKSVolumeAttachService returns the VolumeAttachService for managing volumes through IKS
func (s *Session) IKSVolumeAttachService() instances.VolumeAttachManager {
	return instances.NewIKSVolumeAttachmentManager(s.client)
}

// SnapshotService returns the Snapshot service for managing snapshot
func (s *Session) SnapshotService() vpcvolume.SnapshotManager {
	return vpcvolume.NewSnapshotManager(s.client)
}

// RegionalAPIClientProvider declares an interface for a provider that can supply a new
// RegionalAPI client session
//
//go:generate counterfeiter -o fakes/client_provider.go --fake-name RegionalAPIClientProvider . RegionalAPIClientProvider
type RegionalAPIClientProvider interface {
	New(config Config) (RegionalAPI, error)
}

// DefaultRegionalAPIClientProvider declares a basic client provider that delegates to
// New(). Can be used for dependency injection.
type DefaultRegionalAPIClientProvider struct {
}

var _ RegionalAPIClientProvider = DefaultRegionalAPIClientProvider{}

// New creates a new Session volume, using the supplied config
func (d DefaultRegionalAPIClientProvider) New(config Config) (RegionalAPI, error) {
	return New(config)
}

// IKSSession ...
type IKSSession struct {
	Session
}

var _ RegionalAPI = &IKSSession{}

// VolumeService returns the Volume service for managing volumes
func (s *IKSSession) VolumeService() vpcvolume.VolumeManager {
	return vpcvolume.NewIKSVolumeService(s.client)
}

// IKSRegionalAPIClientProvider ...
type IKSRegionalAPIClientProvider struct {
	RegionalAPIClientProvider
}

var _ RegionalAPIClientProvider = IKSRegionalAPIClientProvider{}

// New creates a new Session , using the supplied config
func (d IKSRegionalAPIClientProvider) New(config Config) (RegionalAPI, error) {
	session, err := New(config)
	if err != nil || session == nil {
		return nil, err
	}
	iksSession := &IKSSession{
		Session: *session,
	}
	return iksSession, err
}

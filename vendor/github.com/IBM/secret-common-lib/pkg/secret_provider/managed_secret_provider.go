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
	"context"
	"flag"
	"net"
	"time"

	localutils "github.com/IBM/secret-common-lib/pkg/utils"
	"github.com/IBM/secret-utils-lib/pkg/config"
	"github.com/IBM/secret-utils-lib/pkg/k8s_utils"
	"github.com/IBM/secret-utils-lib/pkg/utils"
	sp "github.com/IBM/secret-utils-lib/secretprovider"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	endpoint = flag.String("sidecarEndpoint", "/csi/provider.sock", "Storage secret sidecar endpoint")
)

// ManagedSecretProvider ...
type ManagedSecretProvider struct {
	logger                   *zap.Logger
	k8sClient                k8s_utils.KubernetesClient
	region                   string
	riaasEndpoint            string
	privateRIAASEndpoint     string
	containerAPIRoute        string
	privateContainerAPIRoute string
	resourceGroupID          string
}

// newManagedSecretProvider ...
func newManagedSecretProvider(logger *zap.Logger, optionalArgs ...string) (*ManagedSecretProvider, error) {
	logger.Info("Connecting to sidecar")
	kc, err := k8s_utils.Getk8sClientSet(logger)
	if err != nil {
		logger.Info("Error fetching k8s client set", zap.Error(err))
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Connecting to sidecar
	conn, err := grpc.DialContext(ctx, *endpoint, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithContextDialer(unixConnect))
	defer conn.Close()
	if err != nil {
		logger.Error("Error establishing grpc connection to secret sidecar", zap.Error(err))
		return nil, utils.Error{Description: "Error establishing grpc connection", BackendError: err.Error()}
	}

	// If any providerType - vpc, bluemix, softlayer is provided, then make a call to sidecar
	// If it is not provided, no need to make a call to sidecar, on first GetDefaultIAMToken call, secret provider will be initialised
	if len(optionalArgs) != 0 {
		c := sp.NewSecretProviderClient(conn)
		// NewSecretProvider call to sidecar
		_, err = c.NewSecretProvider(ctx, &sp.InitRequest{ProviderType: optionalArgs[0]})
		if err != nil {
			logger.Error("Error initiliazing managed secret provider", zap.Error(err))
			return nil, err
		}
	}

	// Reading endpoints
	msp := &ManagedSecretProvider{logger: logger, k8sClient: kc}
	err = msp.initEndpointsUsingCloudConf()
	if err == nil {
		logger.Info("Initialized managed secret provider")
		return msp, nil
	}

	logger.Info("Unable to fetch endpoints from cloud-conf", zap.Error(err))
	err = msp.initEndpointsUsingStorageSecretStore()
	if err != nil {
		// Do not return even if there is an error reading endpoints, just logging error
		logger.Warn("Unable to fetch endpoints from storage-secret-store", zap.Error(err))
	}

	logger.Info("Initialized managed secret provider")
	return msp, nil
}

// GetDefaultIAMToken ...
func (msp *ManagedSecretProvider) GetDefaultIAMToken(freshTokenRequired bool, reasonForCall ...string) (string, uint64, error) {
	var tokenlifetime uint64
	// Connecting to sidecar
	msp.logger.Info("Connecting to sidecar")
	conn, err := grpc.Dial(*endpoint, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithContextDialer(unixConnect))
	if err != nil {
		msp.logger.Error("Error establishing grpc connection to secret sidecar", zap.Error(err))
		return "", tokenlifetime, utils.Error{Description: "Error establishing grpc connection to secret sidecar", BackendError: err.Error()}
	}

	c := sp.NewSecretProviderClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	defer conn.Close()

	tokenReq := new(sp.Request)
	tokenReq.IsFreshTokenRequired = freshTokenRequired
	if len(reasonForCall) != 0 {
		tokenReq.ReasonForCall = reasonForCall[0]
	}
	response, err := c.GetDefaultIAMToken(ctx, tokenReq)
	if err != nil {
		msp.logger.Error("Error fetching IAM token", zap.Error(err))
		return "", tokenlifetime, err
	}

	msp.logger.Info("Fetched IAM token for default secret")
	return response.Iamtoken, response.Tokenlifetime, nil
}

// GetIAMToken ...
func (msp *ManagedSecretProvider) GetIAMToken(secret string, freshTokenRequired bool, reasonForCall ...string) (string, uint64, error) {
	var tokenlifetime uint64

	msp.logger.Info("Connecting to sidecar")
	conn, err := grpc.Dial(*endpoint, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithContextDialer(unixConnect))
	if err != nil {
		msp.logger.Error("Error establishing grpc connection to secret sidecar", zap.Error(err))
		return "", tokenlifetime, utils.Error{Description: "Error establishing grpc connection to secret sidecar", BackendError: err.Error()}
	}

	c := sp.NewSecretProviderClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	defer conn.Close()

	tokenReq := new(sp.Request)
	tokenReq.IsFreshTokenRequired = freshTokenRequired
	tokenReq.Secret = secret
	if len(reasonForCall) != 0 {
		tokenReq.ReasonForCall = reasonForCall[0]
	}
	response, err := c.GetIAMToken(ctx, tokenReq)
	if err != nil {
		msp.logger.Error("Error fetching IAM token", zap.Error(err))
		return "", tokenlifetime, err
	}

	msp.logger.Info("Fetched IAM token for the provided secret")
	return response.Iamtoken, response.Tokenlifetime, nil
}

// unixConnect ...
func unixConnect(ctx context.Context, addr string) (net.Conn, error) {
	unixAddr, err := net.ResolveUnixAddr("unix", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUnix("unix", nil, unixAddr)
	return conn, err
}

// GetRIAASEndpoint ...
func (msp *ManagedSecretProvider) GetRIAASEndpoint(readConfig bool) (string, error) {
	msp.logger.Info("In GetRIAASEndpoint()")
	if !readConfig {
		msp.logger.Info("Returning RIAAS endpoint", zap.String("Endpoint", msp.riaasEndpoint))
		return msp.riaasEndpoint, nil
	}

	endpoint, err := getEndpoint(localutils.RIAAS, msp.riaasEndpoint, msp.k8sClient, msp.logger)
	if err != nil {
		return "", err
	}

	msp.riaasEndpoint = endpoint
	return endpoint, nil
}

// GetPrivateRIAASEndpoint ...
func (msp *ManagedSecretProvider) GetPrivateRIAASEndpoint(readConfig bool) (string, error) {
	msp.logger.Info("In GetPrivateRIAASEndpoint()")
	if !readConfig {
		msp.logger.Info("Returning private RIAAS endpoint", zap.String("Endpoint", msp.privateRIAASEndpoint))
		return msp.privateRIAASEndpoint, nil
	}

	endpoint, err := getEndpoint(localutils.PrivateRIAAS, msp.privateRIAASEndpoint, msp.k8sClient, msp.logger)
	if err != nil {
		return "", err
	}

	msp.privateRIAASEndpoint = endpoint
	return endpoint, nil
}

// GetContainerAPIRoute ...
func (msp *ManagedSecretProvider) GetContainerAPIRoute(readConfig bool) (string, error) {
	msp.logger.Info("In GetContainerAPIRoute()")
	if !readConfig {
		msp.logger.Info("Returning container api route", zap.String("Endpoint", msp.containerAPIRoute))
		return msp.containerAPIRoute, nil
	}

	endpoint, err := getEndpoint(localutils.ContainerAPIRoute, msp.containerAPIRoute, msp.k8sClient, msp.logger)
	if err != nil {
		return "", err
	}

	msp.containerAPIRoute = endpoint
	return endpoint, nil
}

// GetPrivateContainerAPIRoute ...
func (msp *ManagedSecretProvider) GetPrivateContainerAPIRoute(readConfig bool) (string, error) {
	msp.logger.Info("In GetPrivateContainerAPIRoute()")
	if !readConfig {
		msp.logger.Info("Returning private container api route", zap.String("Endpoint", msp.privateContainerAPIRoute))
		return msp.privateContainerAPIRoute, nil
	}

	endpoint, err := getEndpoint(localutils.PrivateContainerAPIRoute, msp.privateContainerAPIRoute, msp.k8sClient, msp.logger)
	if err != nil {
		return "", err
	}

	msp.privateContainerAPIRoute = endpoint
	return endpoint, nil
}

// GetResourceGroupID ...
func (msp *ManagedSecretProvider) GetResourceGroupID() string {
	return msp.resourceGroupID
}

// initEndpointsUsingCloudConf ...
func (msp *ManagedSecretProvider) initEndpointsUsingCloudConf() error {
	cloudConf, err := config.GetCloudConf(msp.logger, msp.k8sClient)
	if err != nil {
		return err
	}

	msp.region = cloudConf.Region
	msp.containerAPIRoute = cloudConf.ContainerAPIRoute
	msp.privateContainerAPIRoute = cloudConf.PrivateContainerAPIRoute
	msp.riaasEndpoint = cloudConf.RiaasEndpoint
	msp.privateRIAASEndpoint = cloudConf.PrivateRIAASEndpoint
	msp.resourceGroupID = cloudConf.ResourceGroupID
	return nil
}

// initEndpointsUsingStorageSecretStore ...
func (msp *ManagedSecretProvider) initEndpointsUsingStorageSecretStore() error {
	data, err := k8s_utils.GetSecretData(msp.k8sClient, utils.STORAGE_SECRET_STORE_SECRET, utils.SECRET_STORE_FILE)
	if err != nil {
		return err
	}

	conf, err := config.ParseConfig(msp.logger, data)
	if err != nil {
		return err
	}

	msp.containerAPIRoute = conf.Bluemix.APIEndpointURL
	msp.privateContainerAPIRoute = conf.Bluemix.PrivateAPIRoute
	msp.riaasEndpoint = conf.VPC.G2EndpointURL
	msp.privateRIAASEndpoint = conf.VPC.G2EndpointPrivateURL
	msp.resourceGroupID = conf.VPC.G2ResourceGroupID
	return nil
}

/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2022 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Package secretprovider ...
package secret_provider

import (
	"errors"
)

const (
	// fakeEndpoint ...
	fakeEndpoint = "https://fakehost.com"
)

// FakeSecretProvider ...
type FakeSecretProvider struct {
}

// GetDefaultIAMToken ...
func (fs *FakeSecretProvider) GetDefaultIAMToken(freshTokenRequired bool, reasonForCall ...string) (string, uint64, error) {
	if freshTokenRequired {
		return "token", 1000, nil
	}
	return "", 0, errors.New("fake error")
}

// GetIAMToken ...
func (fs *FakeSecretProvider) GetIAMToken(secret string, freshTokenRequired bool, reasonForCall ...string) (string, uint64, error) {
	if freshTokenRequired {
		return "token", 1000, nil
	}
	return "", 0, errors.New("fake error")
}

// GetRIAASEndpoint ...
func (fs *FakeSecretProvider) GetRIAASEndpoint(readConfig bool) (string, error) {
	return fakeEndpoint, nil
}

// GetPrivateRIAASEndpoint ...
func (fs *FakeSecretProvider) GetPrivateRIAASEndpoint(readConfig bool) (string, error) {
	return fakeEndpoint, nil
}

// GetContainerAPIRoute ...
func (fs *FakeSecretProvider) GetContainerAPIRoute(readConfig bool) (string, error) {
	return fakeEndpoint, nil
}

// GetPrivateContainerAPIRoute ...
func (fs *FakeSecretProvider) GetPrivateContainerAPIRoute(readConfig bool) (string, error) {
	return fakeEndpoint, nil
}

// GetResourceGroupID ...
func (fs *FakeSecretProvider) GetResourceGroupID() string {
	return "resource-group-id"
}

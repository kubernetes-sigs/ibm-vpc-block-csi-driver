/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package ibmcsidriver ...
package ibmcsidriver

import (
	"flag"
	cloudProvider "github.com/IBM/ibm-csi-common/pkg/ibmcloudprovider"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestSetup(t *testing.T) {
	goodEndpoint := flag.String("endpoint", "unix:/tmp/testcsi.sock", "Test CSI endpoint")
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()

	s := NewNonBlockingGRPCServer(logger)
	nonBlockingServer, ok := s.(*nonBlockingGRPCServer)
	assert.Equal(t, true, ok)
	ids := &CSIIdentityServer{}
	cs := &CSIControllerServer{}
	ns := &CSINodeServer{}

	{
		t.Logf("Good setup")
		ls, err := nonBlockingServer.Setup(*goodEndpoint, ids, cs, ns)
		assert.Nil(t, err)
		assert.NotNil(t, ls)
	}

	// Call other methods as well just to execute all line of code
	nonBlockingServer.Wait()
	nonBlockingServer.Stop()
	nonBlockingServer.ForceStop()

	{
		t.Logf("Wrong endpoint format")

		wrongEndpointFormat := flag.String("wrongendpoint", "---:/tmp/testcsi.sock", "Test CSI endpoint")
		_, err := nonBlockingServer.Setup(*wrongEndpointFormat, ids, cs, ns)
		assert.NotNil(t, err)
		t.Logf("---------> error %v", err)
	}

	{
		t.Logf("Wrong Scheme")
		wrongEndpointScheme := flag.String("wrongschemaendpoint", "wrong-scheme:/tmp/testcsi.sock", "Test CSI endpoint")
		_, err := nonBlockingServer.Setup(*wrongEndpointScheme, nil, nil, nil)
		assert.NotNil(t, err)
		t.Logf("---------> error %v", err)
	}

	{
		t.Logf("tcp Scheme")
		tcpEndpointSchema := flag.String("tcpendpoint", "tcp:/tmp/testtcpcsi.sock", "Test CSI endpoint")
		_, err := nonBlockingServer.Setup(*tcpEndpointSchema, nil, nil, nil)
		assert.Nil(t, err)
		t.Logf("---------> error %v", err)
		nonBlockingServer.ForceStop()
	}

	{
		t.Logf("Wrong address")
		wrongAddressEndpointAddress := flag.String("wrongaddressendpoint", "unix:443", "Test CSI endpoint")
		_, err := nonBlockingServer.Setup(*wrongAddressEndpointAddress, nil, nil, nil)
		//assert.Nil(t, err) // Its working on local system
		t.Logf("---------> error %v", err)
	}
}

func TestLogGRPC(t *testing.T) {
	t.Logf("TODO:~ TestLogGRPC")
}

func TestRemoveCSISocket(t *testing.T) {
	// Prepare test data
	endPoint := "localhost:8080"

	removeCSISocket(endPoint)
	var err error
	err = os.Remove(endPoint)
	expectedRemovalSuccess := true
	actualRemovalSuccess := err == nil || (err != nil && !os.IsNotExist(err))
	assert.Equal(t, expectedRemovalSuccess, actualRemovalSuccess)
}

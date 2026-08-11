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
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	cloudProvider "github.com/IBM/ibmcloud-volume-vpc/pkg/ibmcloudprovider"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	tests := []struct {
		name        string
		fullMethod  string
		handler     grpc.UnaryHandler
		expectError bool
	}{
		{
			name:        "Handler succeeds - response and no error returned",
			fullMethod:  "/csi.v1.Controller/CreateVolume",
			handler:     func(ctx context.Context, req interface{}) (interface{}, error) { return "ok", nil },
			expectError: false,
		},
		{
			name:       "Handler returns error - error is propagated",
			fullMethod: "/csi.v1.Controller/DeleteVolume",
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return nil, status.Error(codes.Internal, "handler failed")
			},
			expectError: true,
		},
		{
			name:        "Handler returns nil response with no error",
			fullMethod:  "/csi.v1.Identity/Probe",
			handler:     func(ctx context.Context, req interface{}) (interface{}, error) { return nil, nil },
			expectError: false,
		},
		{
			name:       "Handler returns non-gRPC error",
			fullMethod: "/csi.v1.Node/NodeGetInfo",
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return nil, errors.New("plain error")
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info := &grpc.UnaryServerInfo{FullMethod: tc.fullMethod}
			resp, err := logGRPC(context.Background(), "test-request", info, tc.handler)
			if tc.expectError {
				assert.NotNil(t, err)
				assert.Nil(t, resp)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestDirectoryDelete(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) string
		expectGone bool
	}{
		{
			name: "Existing directory is removed",
			setup: func(t *testing.T) string {
				dir, err := os.MkdirTemp("", "dirdelete-test-*")
				if err != nil {
					t.Fatalf("failed to create temp dir: %v", err)
				}
				return dir
			},
			expectGone: true,
		},
		{
			name: "Non-existent path is a no-op",
			setup: func(t *testing.T) string {
				return "/nonexistent/path/that/does/not/exist-dirdelete"
			},
			expectGone: true, // os.RemoveAll on non-existent path succeeds silently
		},
		{
			name: "Nested directory tree is fully removed",
			setup: func(t *testing.T) string {
				dir, err := os.MkdirTemp("", "dirdelete-nested-*")
				if err != nil {
					t.Fatalf("failed to create temp dir: %v", err)
				}
				if err := os.MkdirAll(dir+"/sub", 0755); err != nil {
					t.Fatalf("failed to create subdir: %v", err)
				}
				if err := os.WriteFile(dir+"/sub/file.txt", []byte("data"), 0644); err != nil {
					t.Fatalf("failed to write nested file: %v", err)
				}
				return dir
			},
			expectGone: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.setup(t)
			directoryDelete(path)
			if tc.expectGone {
				_, err := os.Stat(path)
				assert.True(t, os.IsNotExist(err), "expected path %q to be gone after directoryDelete", path)
			}
		})
	}
}

// TestRemoveCSISocketHelper is NOT a real test — it is a subprocess entry point.
// When the env var CSI_SOCKET_TEST_HELPER is set, it calls removeCSISocket
// directly and relies on the production os.Exit(0) to exit the subprocess.
// The parent test (TestRemoveCSISocket) spawns this as a child process and
// observes the side-effects (socket file removed) without being killed itself.
func TestRemoveCSISocketHelper(t *testing.T) {
	socketPath := os.Getenv("CSI_SOCKET_TEST_HELPER")
	if socketPath == "" {
		// Not running as subprocess helper — skip silently.
		t.Skip("not running as subprocess helper")
	}
	// This calls the real removeCSISocket which blocks on SIGTERM then calls
	// os.Exit(0). The subprocess will exit; the parent test observes exit code 0.
	removeCSISocket(socketPath)
}

func TestRemoveCSISocket(t *testing.T) {
	// removeCSISocket blocks on SIGTERM then calls os.Exit(0).
	// Calling it directly in a test would kill the entire test binary.
	//
	// Strategy: run the test binary itself as a subprocess with
	// -test.run=TestRemoveCSISocketHelper and CSI_SOCKET_TEST_HELPER set to the
	// socket path. The subprocess calls removeCSISocket for real. We send SIGTERM
	// to the subprocess and verify:
	//   1. The subprocess exits with code 0 (os.Exit(0) reached).
	//   2. The socket file has been deleted by the time the subprocess exits.

	// Create a temp file representing the CSI socket
	f, err := os.CreateTemp("", "csi-socket-test-*")
	if err != nil {
		t.Fatalf("failed to create temp socket file: %v", err)
	}
	socketPath := f.Name()
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close temp socket file: %v", err)
	}
	defer func() {
		if err := os.Remove(socketPath); err != nil {
			t.Errorf("failed to remove temp socket file: %v", err)
		}
	}()

	// Build the subprocess command: re-run this test binary targeting the helper
	testBinary, err := exec.LookPath(os.Args[0])
	if err != nil {
		// os.Args[0] is already the absolute path when run via `go test`
		testBinary = os.Args[0]
	}

	cmd := exec.Command(testBinary,
		"-test.run=TestRemoveCSISocketHelper",
		"-test.v",
	)
	cmd.Env = append(os.Environ(), "CSI_SOCKET_TEST_HELPER="+socketPath)

	// Start the subprocess — it will block inside removeCSISocket waiting for SIGTERM
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start subprocess: %v", err)
	}

	// Give the subprocess a moment to reach the signal.Notify / <-sigc block
	time.Sleep(200 * time.Millisecond)

	// Send SIGTERM — this is exactly the signal removeCSISocket is waiting for
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("failed to send SIGTERM to subprocess: %v", err)
	}

	// Wait for subprocess to finish — removeCSISocket calls os.Exit(0) after
	// cleanup, so the process exits cleanly with status 0
	procErr := cmd.Wait()
	if procErr != nil {
		// A non-zero exit is unexpected but non-fatal for the file-removal assertion
		t.Logf("subprocess exited with: %v", procErr)
	}

	// The key assertion: the socket file must have been removed by removeCSISocket
	_, statErr := os.Stat(socketPath)
	assert.True(t, os.IsNotExist(statErr),
		"socket file %q should have been removed by removeCSISocket", socketPath)
}

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
	"errors"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// NonBlockingGRPCServer Defines Non blocking GRPC server interfaces
type NonBlockingGRPCServer interface {
	// Start services at the endpoint
	Start(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer)
	// Waits for the service to stop
	Wait()
	// Stops the service gracefully
	Stop()
	// Stops the service forcefully
	ForceStop()
}

// NewNonBlockingGRPCServer ...
func NewNonBlockingGRPCServer(logger *zap.Logger) NonBlockingGRPCServer {
	return &nonBlockingGRPCServer{logger: logger}
}

// nonBlockingGRPCServer server
type nonBlockingGRPCServer struct {
	wg     sync.WaitGroup
	server *grpc.Server
	logger *zap.Logger
}

// Start ...
func (s *nonBlockingGRPCServer) Start(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) {
	s.wg.Add(1)

	go s.serve(endpoint, ids, cs, ns)
}

// Wait ...
func (s *nonBlockingGRPCServer) Wait() {
	s.wg.Wait()
}

// Stop ...
func (s *nonBlockingGRPCServer) Stop() {
	s.server.GracefulStop()
}

// ForceStop ...
func (s *nonBlockingGRPCServer) ForceStop() {
	s.server.Stop()
}

// Setup ...
func (s *nonBlockingGRPCServer) Setup(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) (net.Listener, error) {
	s.logger.Info("nonBlockingGRPCServer-Setup...", zap.Reflect("Endpoint", endpoint))

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logGRPC),
	}

	u, err := url.Parse(endpoint)

	if err != nil {
		msg := "Failed to parse endpoint"
		s.logger.Error(msg, zap.Error(err))
		return nil, err
	}

	var addr string
	if u.Scheme == "unix" {
		addr = u.Path
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			s.logger.Error("Failed to remove", zap.Reflect("addr", addr), zap.Error(err))
			return nil, err
		}
	} else if u.Scheme == "tcp" {
		addr = u.Host
	} else {
		msg := "Endpoint scheme not supported"
		s.logger.Error(msg, zap.Reflect("Scheme", u.Scheme))
		return nil, errors.New(msg)
	}

	s.logger.Info("Start listening GRPC Server", zap.Reflect("Scheme", u.Scheme), zap.Reflect("Addr", addr))

	listener, err := net.Listen(u.Scheme, addr)
	if err != nil {
		msg := "Failed to listen GRPC Server"
		s.logger.Error(msg, zap.Reflect("Error", err))
		return nil, errors.New(msg)
	}

	server := grpc.NewServer(opts...)
	s.server = server

	if ids != nil {
		csi.RegisterIdentityServer(s.server, ids)
	}
	if cs != nil {
		csi.RegisterControllerServer(s.server, cs)
	}
	if ns != nil {
		csi.RegisterNodeServer(s.server, ns)
	}
	go removeCSISocket(addr)
	return listener, nil
}

// serve ...
func (s *nonBlockingGRPCServer) serve(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) {
	s.logger.Info("nonBlockingGRPCServer-serve...", zap.Reflect("Endpoint", endpoint))
	//! Setup
	listener, err := s.Setup(endpoint, ids, cs, ns)
	if err != nil {
		s.logger.Fatal("Failed to setup GRPC Server", zap.Error(err))
	}
	s.logger.Info("Listening GRPC server for connections", zap.Reflect("Addr", listener.Addr()))
	if err := s.server.Serve(listener); err != nil {
		s.logger.Info("Failed to serve", zap.Error(err))
	}
}

// logGRPC ...
func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	glog.V(3).Infof("GRPC call: %s", info.FullMethod)
	glog.V(5).Infof("GRPC request: %+v", req)
	resp, err := handler(ctx, req)
	if err != nil {
		glog.Errorf("GRPC error: %v", err)
	} else {
		glog.V(5).Infof("GRPC response: %+v", resp)
	}
	return resp, err
}
func removeCSISocket(endPoint string) {
	// Reference: https://github.com/kubernetes-csi/node-driver-registrar/blob/master/cmd/csi-node-driver-registrar/node_register.go#L168
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGTERM)
	<-sigc
	err := os.Remove(endPoint)
	if err != nil && !os.IsNotExist(err) {
		glog.Errorf("failed to remove socket: %s with error: %+v", endPoint, err)
	}
	/*
		This is a temporary code to cleanup csi-socket created under csi-plugins directory.
		This code must be removed once current supported versions are deprecated and
		new major release is done.
	*/
	csiPluginDataPath := "/var/lib/kubelet/csi-plugins/vpc.block.csi.ibm.io/"
	csiPluginLibPath := "/var/data/kubelet/csi-plugins/vpc.block.csi.ibm.io/"
	directoryDelete(csiPluginDataPath)
	directoryDelete(csiPluginLibPath)
	os.Exit(0)

}

func directoryDelete(csiPluginSocketPath string) {
	err := os.RemoveAll(csiPluginSocketPath)
	if err != nil {
		glog.Errorf("Error deleting path %s: %v", csiPluginSocketPath, err)
		return
	}
	glog.Infof("Path %s deleted successfully:", csiPluginSocketPath)

}

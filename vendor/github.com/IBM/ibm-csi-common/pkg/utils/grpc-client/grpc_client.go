/**
 * Copyright 2021 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

//Package grpcclient ...
package grpcclient

import (
	"google.golang.org/grpc"
)

//GrpcSessionFactory defines NewGrpcSession
type GrpcSessionFactory interface {
	NewGrpcSession() GrpcSession
}

//GrpcSession defines GrpcDial
type GrpcSession interface {
	GrpcDial(cc ClientConn, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
}

//ConnObjFactory defines empty object
type ConnObjFactory struct{}

//ClientConn defines main gRPC functionality
type ClientConn interface {
	Connect(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
	Close() error
}

//GrpcSes implements ClientConn and GrpcSession
type GrpcSes struct {
	conn *grpc.ClientConn
	cc   ClientConn
}

//Connect creates a client connection to a given target
func (c *GrpcSes) Connect(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	var err error
	c.conn, err = grpc.Dial(target, opts...)
	return c.conn, err
}

//Close tears down the client connection and all underlying connections.
func (c *GrpcSes) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

//NewGrpcSession returns empty GrpcSes object.
func (c *ConnObjFactory) NewGrpcSession() GrpcSession {
	return &GrpcSes{}
}

// GrpcDial establishes a grpc-client client server connection
func (c *GrpcSes) GrpcDial(cc ClientConn, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	conn, err := cc.Connect(target, opts...)
	if err != nil {
		return nil, err
	}
	return conn, err
}

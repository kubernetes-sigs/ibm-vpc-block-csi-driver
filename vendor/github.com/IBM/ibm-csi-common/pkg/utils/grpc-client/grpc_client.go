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

package grpc_client

import (
	"google.golang.org/grpc"
)

type GrpcSessionFactory interface {
	NewGrpcSession() GrpcSession
}

type GrpcSession interface {
	GrpcDial(cc ClientConn, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
}

type ConnObjFactory struct{}

type ClientConn interface {
	Connect(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
	Close() error
}

type GrpcSes struct {
	conn *grpc.ClientConn
	cc   ClientConn
}

func (gs *GrpcSes) Connect(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	var err error
	gs.conn, err = grpc.Dial(target, opts...)
	return gs.conn, err
}

func (gs *GrpcSes) Close() error {
	if gs.conn != nil {
		return gs.conn.Close()
	}
	return nil
}

func (c *ConnObjFactory) NewGrpcSession() GrpcSession {
	return &GrpcSes{}
}

var cc ClientConn = &GrpcSes{}

// GrpcDial establishes a grpc-client client server connection
func (c *GrpcSes) GrpcDial(cc ClientConn, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	conn, err := cc.Connect(target, opts...)
	if err != nil {
		return nil, err
	}
	return conn, err
}

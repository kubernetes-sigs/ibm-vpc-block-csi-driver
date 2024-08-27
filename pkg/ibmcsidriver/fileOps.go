/**
 *
 * Copyright 2024- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
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

// Package ibmcsidriver ...
package ibmcsidriver

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

import (
	"os"
	"strconv"

	"go.uber.org/zap"
)

const (
	filePermission = 0660
)

//counterfeiter:generate . socketPermission

// socketPermission represents file system operations
type socketPermission interface {
	Chown(name string, uid, gid int) error
	Chmod(name string, mode os.FileMode) error
}

// realSocketPermission implements socketPermission
type opsSocketPermission struct{}

func (f *opsSocketPermission) Chown(name string, uid, gid int) error {
	return os.Chown(name, uid, gid)
}

func (f *opsSocketPermission) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

// setupSidecar updates owner/group and permission of the file given(addr)
func setupSidecar(addr string, ops socketPermission, logger *zap.Logger) error {
	groupSt := os.Getenv("SIDECAR_GROUP_ID")

	logger.Info("Setting owner and permissions of csi socket file. SIDECAR_GROUP_ID env must match the 'livenessprobe' sidecar container groupID for csi socket connection.")

	// If env is not set, set default to 0
	if groupSt == "" {
		logger.Warn("Unable to fetch SIDECAR_GROUP_ID environment variable. Sidecar container(s) might fail...")
		groupSt = "0"
	}

	group, err := strconv.Atoi(groupSt)
	if err != nil {
		return err
	}

	// Change group of csi socket to non-root user for enabling the csi sidecar
	if err := ops.Chown(addr, -1, group); err != nil {
		return err
	}

	// Modify permissions of csi socket
	// Only the users and the group owners will have read/write access to csi socket
	if err := ops.Chmod(addr, filePermission); err != nil {
		return err
	}

	logger.Info("Successfully set owner and permissions of csi socket file.")

	return nil
}

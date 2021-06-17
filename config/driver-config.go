/**
 *
 * Copyright 2021- IBM Inc. All rights reserved
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

// Package config ...
package config

const (

	// CSIDriverName this name for csi and Kubernetes
	CSIDriverName = "vpc.block.csi.ibm.io"

	// CSISocketName ...
	CSISocketName = "unix:/tmp/csi.sock"

	// CSIDriverLogName ...
	CSIDriverLogName = "IBM VPC block driver"

	// CSIDriverGithubName ...
	CSIDriverGithubName = "ibm-vpc-block-csi-driver"

	// CSIProviderName for unit test
	CSIProviderName = "VPC-Classic"

	// CSIProviderVolumeType ... same is used to update the volume
	CSIProviderVolumeType = "block"
)

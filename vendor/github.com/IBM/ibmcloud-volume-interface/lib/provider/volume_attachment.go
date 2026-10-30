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

// Package provider ...
package provider

import (
	"net/http"
	"time"
)

const (
	// SUCCESS ...
	SUCCESS = "Success"
	// FAILURE ...
	FAILURE = "Failure"
	// NOTSUPPORTED ...
	NOTSUPPORTED = "Not supported"
)

// VolumeAttachManager ...
type VolumeAttachManager interface {
	//Attach method attaches a volume/ fileset to a server
	//Its non bloking call and does not wait to complete the attachment
	AttachVolume(attachRequest VolumeAttachmentRequest) (*VolumeAttachmentResponse, error)
	//Detach detaches the volume/ fileset from the server
	//Its non bloking call and does not wait to complete the detachment
	DetachVolume(detachRequest VolumeAttachmentRequest) (*http.Response, error)

	//WaitForAttachVolume waits for the volume to be attached to the host
	//Return error if wait is timed out OR there is other error
	WaitForAttachVolume(attachRequest VolumeAttachmentRequest) (*VolumeAttachmentResponse, error)

	//WaitForDetachVolume waits for the volume to be detached from the host
	//Return error if wait is timed out OR there is other error
	WaitForDetachVolume(detachRequest VolumeAttachmentRequest) error

	//GetAttachAttachment retirves the current status of given volume attach request
	GetVolumeAttachment(attachRequest VolumeAttachmentRequest) (*VolumeAttachmentResponse, error)

	// RotateBMSEncryptedSecret calls
	// POST v1/bare_metal_servers/{serverID}/storage_access/encrypted_secret/rotate
	// body: {"key":{"id":keyID}}
	// BMS-only: tells VPC to re-encrypt the NVMe-oF PSK with the RSA public key
	// registered as VPC key {keyID}.  Non-BMS implementations return an error.
	RotateBMSEncryptedSecret(serverID, keyID string) error

	// GetBMSServer calls GET v1/bare_metal_servers/{serverID} and returns both
	// BMS fields needed by the CSI controller in a single round-trip:
	//   nvme_qualified_name            → HOSTNQN (stable initiator NQN of this server)
	//   storage_access.encrypted_secret → RSA-encrypted PSK ciphertext
	// Per the VPC BMS Block Storage SDP User Guide both fields are present in
	// GET /bare_metal_servers/{id}, making a separate GET /storage_access call
	// unnecessary.  Non-BMS implementations return an error.
	GetBMSServer(serverID string) (*BMSServerInfo, error)
}

// BMSServerInfo carries the two BMS server fields the CSI controller needs
// from GET v1/bare_metal_servers/{id}.
type BMSServerInfo struct {
	// HostNQN is the initiator NQN of the bare metal server (nvme_qualified_name).
	// Stable for the lifetime of the server.
	// Used as: nvme connect --hostnqn <HostNQN>
	HostNQN string `json:"host_nqn,omitempty"`

	// EncryptedSecret is the RSA-encrypted PSK ciphertext
	// (storage_access.encrypted_secret).
	// Safe to store in etcd / publishContext — useless without rsa_private_key.
	EncryptedSecret string `json:"encrypted_secret,omitempty"`
}

// VolumeAttachmentResponse used for both attach and detach operation
type VolumeAttachmentResponse struct {
	VolumeAttachmentRequest
	//Status status of the volume attachment success, failed, attached, attaching, detaching
	Status    string     `json:"status,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	// BMSResponse carries BMS SDP-specific fields from a successful attachment.
	// Nil for VSI and IKS attachments.
	BMSResponse *BMSVolumeAttachmentResponse `json:"bmsResponse,omitempty"`
}

// BMSVolumeAttachmentResponse carries the NVMe-oF fields returned by the
// BMS SDP attach API that the CSI controller needs to build publishContext.
type BMSVolumeAttachmentResponse struct {
	// SubsystemNQN is the NVMe qualified name of the volume's subsystem
	SubsystemNQN string `json:"subsystemNQN,omitempty"`
	// TargetIPs is the list of NVMe-oF TCP target IPs (up to 4)
	TargetIPs []string `json:"targetIPs,omitempty"`
}

// VolumeAttachmentRequest  used for both attach and detach operation
type VolumeAttachmentRequest struct {
	VolumeID   string `json:"volumeID"`
	InstanceID string `json:"instanceID"`
	// Only for SL provider
	SoftlayerOptions map[string]string `json:"softlayerOptions,omitempty"`
	// Only for VPC provider
	VPCVolumeAttachment *VolumeAttachment `json:"vpcVolumeAttachment"`
	// Only IKS provider
	IKSVolumeAttachment *IKSVolumeAttachment `json:"iksVolumeAttachment"`
	// Only for BMS SDP (self-managed cluster, bare metal node). Nil on IKS and VSI clusters.
	BMSVolumeAttachment *BMSVolumeAttachment `json:"bmsVolumeAttachment,omitempty"`
}

// BMSVolumeAttachment carries BMS SDP-specific attachment parameters.
// Protocol should be set to "nvme_tcp" for NVMe-oF over TCP.
type BMSVolumeAttachment struct {
	Protocol string `json:"protocol,omitempty"`
}

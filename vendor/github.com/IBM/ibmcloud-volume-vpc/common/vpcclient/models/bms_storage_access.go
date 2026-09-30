/**
 * Copyright 2025 IBM Corp.
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

// Package models ...
package models

// BMSStorageAccessRotateRequest is the request body for
// POST v1/bare_metal_servers/{serverID}/storage_access/encrypted_secret/rotate
//
// Example curl:
//
//	curl -X POST \
//	  '.../v1/bare_metal_servers/{id}/storage_access/encrypted_secret/rotate?version=...&generation=2' \
//	  -H 'Authorization: Bearer TOKEN' \
//	  -H 'Content-Type: application/json' \
//	  -d '{"key":{"id":"r134-e2f1a01f-bf35-444b-ba20-e3633cc201cd"}}'
//
// VPC re-encrypts the NVMe-oF PSK with the RSA public key registered as the
// VPC key resource identified by Key.ID and stores the ciphertext.
type BMSStorageAccessRotateRequest struct {
	Key BMSKeyRef `json:"key"`
}

// BMSKeyRef references a VPC key resource by ID.
type BMSKeyRef struct {
	ID string `json:"id"`
}

// BMSStorageAccessResponse is the response from
// GET v1/bare_metal_servers/{serverID}/storage_access
// It is also embedded inside BMSServerResponse as the storage_access sub-object.
type BMSStorageAccessResponse struct {
	// EncryptedSecret is the base64-encoded RSA ciphertext of the NVMe-oF PSK.
	// Encrypted with the public key identified by the key ID used in the
	// preceding rotate call.  Safe to store in etcd via publishContext because
	// it is useless without the rsa_private_key held only by the node DaemonSet.
	EncryptedSecret string `json:"encrypted_secret"`
}

// BMSServerResponse is the relevant subset of the response from
// GET v1/bare_metal_servers/{id}
//
// According to the VPC BMS Block Storage SDP User Guide:
//
//	nvme_qualified_name          → HOSTNQN: the host/initiator NQN used in ALL nvme commands
//	storage_access.encrypted_secret → PSK ciphertext (same as GET /storage_access)
//
// Both fields are returned in a single API call, so the controller can get
// the host NQN and the encrypted PSK together after one rotate + one GET.
type BMSServerResponse struct {
	// NVMeQualifiedName is the HOSTNQN — the initiator NQN of the bare metal
	// server itself.  Stable for the lifetime of the server.
	// Used as: nvme connect --hostnqn <NVMeQualifiedName>
	NVMeQualifiedName string `json:"nvme_qualified_name,omitempty"`

	// StorageAccess embeds the storage_access sub-object which contains the
	// encrypted PSK ciphertext (same data as GET /storage_access).
	StorageAccess *BMSStorageAccessResponse `json:"storage_access,omitempty"`
}

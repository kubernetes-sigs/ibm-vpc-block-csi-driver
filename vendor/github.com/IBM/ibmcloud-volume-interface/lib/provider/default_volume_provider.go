/**
 * Copyright 2021 IBM Corp.
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

import "net/http"

//DefaultVolumeProvider Implementation
type DefaultVolumeProvider struct {
	sess *Session
}

var _ Session = &DefaultVolumeProvider{sess: nil}

//ProviderName returns provider
func (volprov *DefaultVolumeProvider) ProviderName() VolumeProvider {
	return VolumeProvider("")
}

//Type returns the underlying volume type
func (volprov *DefaultVolumeProvider) Type() VolumeType {
	return ""
}

//CreateVolume creates a volume
func (volprov *DefaultVolumeProvider) CreateVolume(VolumeRequest Volume) (*Volume, error) {
	return nil, nil
}

//AttachVolume attaches a volume
func (volprov *DefaultVolumeProvider) AttachVolume(attachRequest VolumeAttachmentRequest) (*VolumeAttachmentResponse, error) {
	return nil, nil
}

//CreateVolumeFromSnapshot creates a volume from snapshot
func (volprov *DefaultVolumeProvider) CreateVolumeFromSnapshot(snapshot Snapshot, tags map[string]string) (*Volume, error) {
	return nil, nil
}

//UpdateVolume the volume
func (volprov *DefaultVolumeProvider) UpdateVolume(Volume) error {
	return nil
}

//DeleteVolume deletes the volume
func (volprov *DefaultVolumeProvider) DeleteVolume(*Volume) error {
	return nil
}

//GetVolume by using ID
func (volprov *DefaultVolumeProvider) GetVolume(id string) (*Volume, error) {
	return nil, nil
}

// GetVolumeByName gets volume by name,
// actually some of providers(like VPC) has the capability to provide volume
// details by usig user provided volume name
func (volprov *DefaultVolumeProvider) GetVolumeByName(name string) (*Volume, error) {
	return nil, nil
}

//ListVolumes Get volume lists by using filters
func (volprov *DefaultVolumeProvider) ListVolumes(limit int, start string, tags map[string]string) (*VolumeList, error) {
	return nil, nil
}

// GetVolumeByRequestID fetch the volume by request ID.
// Request Id is the one that is returned when volume is provsioning request is
// placed with Iaas provider.
func (volprov *DefaultVolumeProvider) GetVolumeByRequestID(requestID string) (*Volume, error) {
	return nil, nil
}

//AuthorizeVolume allows aceess to volume  based on given authorization
func (volprov *DefaultVolumeProvider) AuthorizeVolume(volumeAuthorization VolumeAuthorization) error {
	return nil
}

// DetachVolume  by passing required information in the volume object
func (volprov *DefaultVolumeProvider) DetachVolume(detachRequest VolumeAttachmentRequest) (*http.Response, error) {
	return nil, nil
}

//WaitForAttachVolume waits for the volume to be attached to the host
//Return error if wait is timed out OR there is other error
func (volprov *DefaultVolumeProvider) WaitForAttachVolume(attachRequest VolumeAttachmentRequest) (*VolumeAttachmentResponse, error) {
	return nil, nil
}

//WaitForDetachVolume waits for the volume to be detached from the host
//Return error if wait is timed out OR there is other error
func (volprov *DefaultVolumeProvider) WaitForDetachVolume(detachRequest VolumeAttachmentRequest) error {
	return nil
}

//GetVolumeAttachment retirves the current status of given volume attach request
func (volprov *DefaultVolumeProvider) GetVolumeAttachment(attachRequest VolumeAttachmentRequest) (*VolumeAttachmentResponse, error) {
	return nil, nil
}

//OrderSnapshot orders the snapshot
func (volprov *DefaultVolumeProvider) OrderSnapshot(VolumeRequest Volume) error {
	return nil
}

// CreateSnapshot on the volume
func (volprov *DefaultVolumeProvider) CreateSnapshot(sourceVolumeID string, snapshotParameters SnapshotParameters) (*Snapshot, error) {
	return nil, nil
}

//DeleteSnapshot deletes the snapshot
func (volprov *DefaultVolumeProvider) DeleteSnapshot(*Snapshot) error {
	return nil
}

//GetSnapshot gets the snapshot
func (volprov *DefaultVolumeProvider) GetSnapshot(snapshotID string) (*Snapshot, error) {
	return nil, nil
}

//GetSnapshotByName gets the snapshot
func (volprov *DefaultVolumeProvider) GetSnapshotByName(snapshotName string) (*Snapshot, error) {
	return nil, nil
}

//ListSnapshots list the snapshots
func (volprov *DefaultVolumeProvider) ListSnapshots(limit int, start string, tags map[string]string) (*SnapshotList, error) {
	return nil, nil
}

//ExpandVolume expand the volume with authorization by passing required information in the volume object
func (volprov *DefaultVolumeProvider) ExpandVolume(expandVolumeRequest ExpandVolumeRequest) (int64, error) {
	return 0, nil
}

//GetProviderDisplayName gets provider by displayname
func (volprov *DefaultVolumeProvider) GetProviderDisplayName() VolumeProvider {
	return ""
}

//Close is called when the Session is nolonger required
func (volprov *DefaultVolumeProvider) Close() {
}

//CreateVolumeAccessPoint to create access point
func (volprov *DefaultVolumeProvider) CreateVolumeAccessPoint(accessPointRequest VolumeAccessPointRequest) (*VolumeAccessPointResponse, error) {
	return nil, nil
}

//DeleteVolumeAccessPoint method delete a access point
func (volprov *DefaultVolumeProvider) DeleteVolumeAccessPoint(deleteAccessPointRequest VolumeAccessPointRequest) (*http.Response, error) {
	return nil, nil
}

//WaitForCreateVolumeAccessPoint waits for the volume access point to be created
//Return error if wait is timed out OR there is other error
func (volprov *DefaultVolumeProvider) WaitForCreateVolumeAccessPoint(accessPointRequest VolumeAccessPointRequest) (*VolumeAccessPointResponse, error) {
	return nil, nil
}

//WaitForDeleteVolumeAccessPoint waits for the volume access point to be deleted
//Return error if wait is timed out OR there is other error
func (volprov *DefaultVolumeProvider) WaitForDeleteVolumeAccessPoint(deleteAccessPointRequest VolumeAccessPointRequest) error {
	return nil
}

//GetVolumeAccessPoint retrieves the current status of given volume AccessPoint request
func (volprov *DefaultVolumeProvider) GetVolumeAccessPoint(accessPointRequest VolumeAccessPointRequest) (*VolumeAccessPointResponse, error) {
	return nil, nil
}

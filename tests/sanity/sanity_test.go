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

//Package sanity ...

package sanity

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	nodeInfo "github.com/IBM/ibm-csi-common/pkg/metadata/fake"
	"github.com/IBM/ibmcloud-volume-interface/config"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	providerError "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/google/uuid"
	sanity "github.com/kubernetes-csi/csi-test/v4/pkg/sanity"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	cloudProvider "github.com/IBM/ibm-csi-common/pkg/ibmcloudprovider"
	nodeMetadata "github.com/IBM/ibm-csi-common/pkg/metadata"

	mountManager "github.com/IBM/ibm-csi-common/pkg/mountmanager"

	csiConfig "github.com/kubernetes-sigs/ibm-vpc-block-csi-driver/config"
	csiDriver "github.com/kubernetes-sigs/ibm-vpc-block-csi-driver/pkg/ibmcsidriver"
)

const (
	// ProviderName ...
	ProviderName = provider.VolumeProvider(csiConfig.CSIProviderName)

	// VolumeType ...
	VolumeType = provider.VolumeType(csiConfig.CSIProviderVolumeType)

	// FakeNodeID
	FakeNodeID = "fake-node-id"
)

var (
	// Set up variables
	TempDir = "/tmp/csi"

	// CSIEndpoint ...
	CSIEndpoint = fmt.Sprintf("unix:%s/csi.sock", TempDir)

	// TargetPath ...
	TargetPath = path.Join(TempDir, "mount")

	// StagePath ...
	StagePath = path.Join(TempDir, "stage")
)

func TestSanity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sanity testing...")
	}
	skipTests := strings.Join([]string{
		"NodeExpandVolume.*should work if node-expand is called after node-publish",
		//	"NodeExpandVolume.*should fail when volume is not found",
		//	"ListSnapshots.*should return snapshots that match the specified source volume id",
	}, "|")

	// Create a fake CSI driver
	csiSanityDriver := initCSIDriverForSanity(t)

	//  Create the temp directory for fake sanity driver
	err := os.MkdirAll(TempDir, 0755) // #nosec
	if err != nil {
		t.Fatalf("Failed to create sanity temp working dir %s: %v", TempDir, err)
	}
	defer func() {
		// Clean up tmp dir
		if err = os.RemoveAll(TempDir); err != nil {
			t.Fatalf("Failed to clean up sanity temp working dir %s: %v", TempDir, err)
		}
	}()

	go func() {
		csiSanityDriver.Run(CSIEndpoint)
	}()

	// TODO(#818): Fix failing tests and remove test skip flag.
	err = flag.Set("ginkgo.skip", skipTests)

	// Run sanity test
	config := sanity.TestConfig{
		TargetPath:               TargetPath,
		StagingPath:              StagePath,
		Address:                  CSIEndpoint,
		DialOptions:              []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
		IDGen:                    &providerIDGenerator{},
		TestVolumeAccessType:     "mount",
		TestVolumeParametersFile: os.Getenv("SANITY_PARAMS_FILE"),
		TestVolumeSize:           10737418240, // i.e 10 GB
		CreateTargetDir: func(targetPath string) (string, error) {
			return targetPath, createTargetDir(targetPath)
		},
		CreateStagingDir: func(stagePath string) (string, error) {
			return stagePath, createTargetDir(stagePath)
		},
	}
	sanity.Test(t, config)
}

var _ sanity.IDGenerator = &providerIDGenerator{}

type providerIDGenerator struct {
}

func (v providerIDGenerator) GenerateUniqueValidVolumeID() string {
	return fmt.Sprintf("vol-uuid-test-vol-%s", uuid.New().String()[:10])
}

func (v providerIDGenerator) GenerateInvalidVolumeID() string {
	return "invalid-vol-id"
}

func (v providerIDGenerator) GenerateUniqueValidNodeID() string {
	return fmt.Sprintf("%s-%s", FakeNodeID, uuid.New().String()[:10])
}

func (v providerIDGenerator) GenerateInvalidNodeID() string {
	return "invalid-Node-ID"
}

func initCSIDriverForSanity(t *testing.T) *csiDriver.IBMCSIDriver {
	vendorVersion := "test-vendor-version-1.1.2"
	driver := "fakedriver"

	// Creating test logger
	logger, teardown := cloudProvider.GetTestLogger(t)
	defer teardown()
	csiSanityDriver := csiDriver.GetIBMCSIDriver()

	// Create fake provider and mounter
	provider, _ := NewFakeSanityCloudProvider("", logger)
	mounter := mountManager.NewFakeNodeMounter()

	statsUtil := &MockStatSanity{}

	// fake node metadata
	fakeNodeData := nodeMetadata.FakeNodeMetadata{}
	fakeNodeInfo := nodeInfo.FakeNodeInfo{}
	fakeNodeData.GetRegionReturns("testregion")
	fakeNodeData.GetZoneReturns("testzone")
	fakeNodeData.GetWorkerIDReturns("testworker")
	fakeNodeInfo.NewNodeMetadataReturns(&fakeNodeData, nil)

	// Setup the IBM CSI Driver
	err := csiSanityDriver.SetupIBMCSIDriver(provider, mounter, statsUtil, &fakeNodeData, &fakeNodeInfo, logger, driver, vendorVersion)
	if err != nil {
		t.Fatalf("Failed to setup IBM CSI Driver: %v", err)
	}

	return csiSanityDriver
}

// Fake State interface methods implementation for getting
type MockStatSanity struct {
}

// FSInfo ...
func (su *MockStatSanity) FSInfo(path string) (int64, int64, int64, int64, int64, int64, error) {
	return 1, 1, 1, 1, 1, 1, nil
}

// DeviceInfo ...
func (su *MockStatSanity) DeviceInfo(path string) (int64, error) {
	return 1, nil
}

// IsBlockDevice ..
func (su *MockStatSanity) IsBlockDevice(devicePath string) (bool, error) {
	if !strings.Contains(devicePath, TargetPath) {
		return false, fs.ErrNotExist
	}
	return true, nil
}

func (su *MockStatSanity) IsDevicePathNotExist(devicePath string) bool {
	// return true if not matched
	return !strings.Contains(devicePath, TargetPath)
}

// FakeSanityCloudProvider Provider
type FakeSanityCloudProvider struct {
	ProviderName   string
	ProviderConfig *config.Config
	fakeSession    *fakeProviderSession
	ClusterID      string
}

var _ cloudProvider.CloudProviderInterface = &FakeSanityCloudProvider{}

// NewFakeSanityCloudProvider ...
func NewFakeSanityCloudProvider(configPath string, logger *zap.Logger) (*FakeSanityCloudProvider, error) {
	return &FakeSanityCloudProvider{ProviderName: "FakeSanityCloudProvider",
		ProviderConfig: &config.Config{VPC: &config.VPCProviderConfig{VPCBlockProviderName: "VPCFakeProvider"}},
		ClusterID:      "", fakeSession: newFakeProviderSession()}, nil
}

// GetProviderSession ...
func (ficp *FakeSanityCloudProvider) GetProviderSession(ctx context.Context, logger *zap.Logger) (provider.Session, error) {
	return ficp.fakeSession, nil
}

// GetConfig ...
func (ficp *FakeSanityCloudProvider) GetConfig() *config.Config {
	return ficp.ProviderConfig
}

// GetClusterID ...
func (ficp *FakeSanityCloudProvider) GetClusterID() string {
	return ficp.ClusterID
}

type fakeVolume struct {
	*provider.Volume
}

type fakeSnapshot struct {
	*provider.Snapshot
	tags map[string]string
}

type fakeProviderSession struct {
	provider.DefaultVolumeProvider
	volumes      map[string]*fakeVolume
	snapshots    map[string]*fakeSnapshot
	pub          map[string]string
	providerName provider.VolumeProvider
	providerType provider.VolumeType
	tokens       map[string]int
}

func newFakeProviderSession() *fakeProviderSession {
	return &fakeProviderSession{
		volumes:      make(map[string]*fakeVolume),
		snapshots:    make(map[string]*fakeSnapshot),
		pub:          make(map[string]string),
		providerName: csiConfig.CSIProviderName,
		providerType: csiConfig.CSIProviderVolumeType,
		tokens:       make(map[string]int),
	}
}

//##############################################################################
// Following are the fake interface methods from open source common library
// If there is any changes in the interface in the libarary then these also need
// to validate and modify accordingly
//##############################################################################

// ProviderName ...
func (c *fakeProviderSession) ProviderName() provider.VolumeProvider {
	return ProviderName
}

// Type returns the underlying volume type
func (c *fakeProviderSession) Type() provider.VolumeType {
	return VolumeType
}

func (c *fakeProviderSession) Close() {
	// Do nothing for now
}

// GetProviderDisplayName returns the provider name
func (c *fakeProviderSession) GetProviderDisplayName() provider.VolumeProvider {
	return ProviderName
}

// Volume operations
// Create the volume with authorization by passing required information in the volume object
func (c *fakeProviderSession) CreateVolume(volumeRequest provider.Volume) (*provider.Volume, error) {
	if len(volumeRequest.SnapshotID) > 0 {
		if _, ok := c.snapshots[volumeRequest.SnapshotID]; !ok {
			errorMsg := providerError.Message{
				Code:        "SnapshotIDNotFound",
				Description: "Snapshot ID not found",
				Type:        providerError.RetrivalFailed,
			}
			return nil, errorMsg
		}
	}
	if volumeRequest.Name == nil || len(*volumeRequest.Name) == 0 {
		return nil, errors.New("no Volume name passed")
	}
	fakeVolume := &fakeVolume{
		Volume: &provider.Volume{
			VolumeID:   fmt.Sprintf("vol-uuid-test-vol-%s", uuid.New().String()[:10]),
			Name:       volumeRequest.Name,
			Region:     volumeRequest.Region,
			Capacity:   volumeRequest.Capacity,
			SnapshotID: volumeRequest.SnapshotID,
		},
	}

	c.volumes[*volumeRequest.Name] = fakeVolume
	return fakeVolume.Volume, nil
}

func (c *fakeProviderSession) UpdateVolume(volumeRequest provider.Volume) error {
	return nil
}

// Create the volume from snapshot with snapshot tags
func (c *fakeProviderSession) CreateVolumeFromSnapshot(snapshot provider.Snapshot, tags map[string]string) (*provider.Volume, error) {
	return nil, nil
}

// Delete the volume
func (c *fakeProviderSession) DeleteVolume(vol *provider.Volume) error {
	for volName, f := range c.volumes {
		if f.Volume.VolumeID == vol.VolumeID {
			delete(c.volumes, volName)
			return nil
		}
	}
	erroMsg := providerError.Message{
		Code:        "FailedToDeleteVolume",
		Description: "Volume not found for deletion",
		Type:        providerError.DeletionFailed,
	}

	return erroMsg
}

// Expand the volume
func (c *fakeProviderSession) ExpandVolume(expandVolumeRequest provider.ExpandVolumeRequest) (int64, error) {
	volumeID := expandVolumeRequest.VolumeID
	capacity := expandVolumeRequest.Capacity
	if len(volumeID) == 0 {
		return -1, errors.New("no Volume id passed")
	}

	if capacity == -1 {
		return -1, errors.New("no capacity passed")
	}

	for _, f := range c.volumes {
		if f.Volume.VolumeID == volumeID {
			return capacity, nil
		}
	}
	return -1, errors.New("no volume found")
}

// Get the volume by using ID  //
func (c *fakeProviderSession) GetVolume(id string) (*provider.Volume, error) {
	for _, f := range c.volumes {
		if f.Volume.VolumeID == id {
			return f.Volume, nil
		}
	}
	errorMsg := providerError.Message{
		Code:        "StorageFindFailedWithVolumeID",
		Description: "Volume not found by volume ID",
		Type:        providerError.RetrivalFailed,
	}
	return nil, errorMsg
}

// Get the volume by using Name
func (c *fakeProviderSession) GetVolumeByName(name string) (*provider.Volume, error) {
	for _, f := range c.volumes {
		if *f.Volume.Name == name {
			return f.Volume, nil
		}
	}
	errorMsg := providerError.Message{
		Code:        "StorageFindFailedWithVolumeName",
		Description: "Volume not found by name",
		Type:        providerError.RetrivalFailed,
	}
	return nil, errorMsg
}

// Get volume lists
func (c *fakeProviderSession) ListVolumes(limit int, start string, tags map[string]string) (*provider.VolumeList, error) {
	maxLimit := 100
	var respVolumesList = &provider.VolumeList{}
	errorMsg := providerError.Message{
		Code:        "StartVolumeIDNotFound",
		Description: "The volume ID specified in the start parameter of the list volume call could not be found.",
		Type:        providerError.InvalidRequest,
	}
	if start != "" {
		if _, ok := c.volumes[start]; !ok {
			return nil, errorMsg
		}
	}

	if limit == 0 {
		limit = 50
	} else if limit > maxLimit {
		limit = maxLimit
	}
	i := 1
	for _, f := range c.volumes {
		if i > limit {
			break
		}
		respVolumesList.Volumes = append(respVolumesList.Volumes, f.Volume)
		i++
	}
	return respVolumesList, nil
}

// Others
// GetVolumeByRequestID fetch the volume by request ID.
// Request Id is the one that is returned when volume is provsioning request is
// placed with Iaas provider.
func (c *fakeProviderSession) GetVolumeByRequestID(requestID string) (*provider.Volume, error) {
	return nil, nil
}

// AuthorizeVolume allows aceess to volume  based on given authorization
func (c *fakeProviderSession) AuthorizeVolume(volumeAuthorization provider.VolumeAuthorization) error {
	return nil
}

func (c *fakeProviderSession) AttachVolume(attachRequest provider.VolumeAttachmentRequest) (*provider.VolumeAttachmentResponse, error) {
	if len(attachRequest.InstanceID) == 0 {
		return nil, errors.New("instance does not passed")
	}

	if strings.Contains(attachRequest.InstanceID, FakeNodeID) {
		return nil, providerError.Message{
			Code:        "AttachFailed",
			Description: "Volume attachment fail due to instance not exist",
			Type:        providerError.NodeNotFound,
		}
	}

	attachmentDetails := &provider.VolumeAttachmentResponse{
		VolumeAttachmentRequest: provider.VolumeAttachmentRequest{
			VolumeID:            attachRequest.VolumeID,
			InstanceID:          attachRequest.InstanceID,
			VPCVolumeAttachment: &provider.VolumeAttachment{DevicePath: "/csi/mount/vol1"},
		},
	}
	return attachmentDetails, nil
}

// Detach detaches the volume/ fileset from the server
// Its non bloking call and does not wait to complete the detachment
func (c *fakeProviderSession) DetachVolume(detachRequest provider.VolumeAttachmentRequest) (*http.Response, error) {
	return nil, nil
}

// WaitForAttachVolume waits for the volume to be attached to the host
// Return error if wait is timed out OR there is other error
func (c *fakeProviderSession) WaitForAttachVolume(attachRequest provider.VolumeAttachmentRequest) (*provider.VolumeAttachmentResponse, error) {
	if len(attachRequest.InstanceID) == 0 {
		return nil, errors.New("no instance ID passed")
	}

	return &provider.VolumeAttachmentResponse{
		VolumeAttachmentRequest: provider.VolumeAttachmentRequest{
			VolumeID:            attachRequest.VolumeID,
			InstanceID:          attachRequest.InstanceID,
			VPCVolumeAttachment: &provider.VolumeAttachment{DevicePath: "/csi/mount/vol1"},
		},
	}, nil
}

// WaitForDetachVolume waits for the volume to be detached from the host
// Return error if wait is timed out OR there is other error
func (c *fakeProviderSession) WaitForDetachVolume(detachRequest provider.VolumeAttachmentRequest) error {
	return nil
}

// GetAttachAttachment retirves the current status of given volume attach request
func (c *fakeProviderSession) GetVolumeAttachment(attachRequest provider.VolumeAttachmentRequest) (*provider.VolumeAttachmentResponse, error) {
	return nil, nil
}

// Snapshot operations
// Create the snapshot on the volume
func (c *fakeProviderSession) CreateSnapshot(sourceVolumeID string, snapshotParameters provider.SnapshotParameters) (*provider.Snapshot, error) {
	snapshotID := fmt.Sprintf("vol-uuid-test-vol-%s", uuid.New().String()[:10])
	for _, existingSnapshot := range c.snapshots {
		if existingSnapshot.Snapshot.SnapshotID == snapshotID && existingSnapshot.Snapshot.VolumeID == sourceVolumeID {
			return nil, errors.New("snapshot already present for same volume")
		}
	}
	fakeSnapshot := &fakeSnapshot{
		Snapshot: &provider.Snapshot{
			VolumeID:             sourceVolumeID,
			SnapshotID:           snapshotID,
			ReadyToUse:           false,
			SnapshotSize:         1,
			SnapshotCreationTime: time.Now(),
		},
		tags: snapshotParameters.SnapshotTags,
	}

	c.snapshots[snapshotID] = fakeSnapshot
	return fakeSnapshot.Snapshot, nil

}

// Delete the snapshot
func (c *fakeProviderSession) DeleteSnapshot(snap *provider.Snapshot) error {
	delete(c.snapshots, snap.SnapshotID)
	return nil
}

// Get the snapshot
func (c *fakeProviderSession) GetSnapshot(snapshotID string) (*provider.Snapshot, error) {
	fmt.Println("GetSnapshot", c.snapshots)
	fmt.Println("snapshotID", snapshotID)
	fmt.Println("c.snapshots[snapshotID]", c.snapshots[snapshotID])
	ret, err := c.snapshots[snapshotID]
	fmt.Println(ret)
	if !err {
		fmt.Println("Error")
		return nil, errors.New("error")
	}
	return ret.Snapshot, nil
}

// Snapshot list by using tags
func (c *fakeProviderSession) ListSnapshots(maxResults int, nextToken string, tags map[string]string) (*provider.SnapshotList, error) {
	var snapshots []*provider.Snapshot
	var retToken string
	for _, fakeSnapshot := range c.snapshots {
		if fakeSnapshot.Snapshot.VolumeID == tags["source_volume.id"] || len(tags["source_volume.id"]) == 0 {
			snapshots = append(snapshots, fakeSnapshot.Snapshot)
		}
	}
	if maxResults > 0 {
		r1 := rand.New(rand.NewSource(time.Now().UnixNano()))
		retToken = fmt.Sprintf("token-%d", r1.Uint64())
		c.tokens[retToken] = maxResults
		snapshots = snapshots[0:maxResults]
		fmt.Printf("%v\n", snapshots)
	}
	if len(nextToken) != 0 {
		snapshots = snapshots[c.tokens[nextToken]:]
	}
	return &provider.SnapshotList{
		Snapshots: snapshots,
		Next:      retToken,
	}, nil
}

// Get the snapshot By name
func (c *fakeProviderSession) GetSnapshotByName(snapshotName string) (*provider.Snapshot, error) {
	if len(snapshotName) == 0 {
		return nil, errors.New("no name passed")
	}
	var snapshots []*fakeSnapshot
	for _, s := range c.snapshots {
		name, exists := s.tags["name"]
		if !exists {
			continue
		}
		if name == snapshotName {
			fmt.Println("name is same")
			snapshots = append(snapshots, s)
		}
	}
	if len(snapshots) == 0 {
		errorMsg := providerError.Message{
			Code:        "StorageFindFailedWithSnapshotName",
			Description: "Snapshot not found by name",
			Type:        providerError.RetrivalFailed,
		}
		return nil, errorMsg
	}

	return snapshots[0].Snapshot, nil
}

func createTargetDir(targetPath string) error {
	fileInfo, err := os.Stat(targetPath)
	if err != nil && os.IsNotExist(err) {
		return os.MkdirAll(targetPath, 0755)
	} else if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		return fmt.Errorf("target location %s is not a directory", targetPath)
	}

	return nil
}

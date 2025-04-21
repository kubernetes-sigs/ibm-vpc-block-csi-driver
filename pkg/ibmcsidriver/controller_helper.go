/*
Copyright 2024 The Kubernetes Authors.

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
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/IBM/ibm-csi-common/pkg/utils"
	"github.com/IBM/ibmcloud-volume-interface/config"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	providerError "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// normalize the requested capacity(in GiB) to what is supported by the driver
func getRequestedCapacity(capRange *csi.CapacityRange, profileName string) (int64, error) {
	// Input is in bytes from csi
	var capBytes int64
	// Default case where nothing is set
	if capRange == nil {
		if profileName == SDPProfile { // SDP profile minimum size is 1GB
			capBytes = MinimumSDPVolumeSizeInBytes
		} else {
			capBytes = utils.MinimumVolumeSizeInBytes // tier and custom profile minimum size is 10 GB
		}
		// returns in GiB
		return capBytes, nil
	}

	rBytes := capRange.GetRequiredBytes()
	rSet := rBytes > 0
	lBytes := capRange.GetLimitBytes()
	lSet := lBytes > 0

	if lSet && rSet && lBytes < rBytes {
		return 0, fmt.Errorf("limit bytes %v is less than required bytes %v", lBytes, rBytes)
	}
	if lSet && lBytes < utils.MinimumVolumeSizeInBytes {
		return 0, fmt.Errorf("limit bytes %v is less than minimum volume size: %v", lBytes, utils.MinimumVolumeSizeInBytes)
	}

	// If Required set just set capacity to that which is Required
	if rSet {
		capBytes = rBytes
	}

	// Roundup the volume size to the next integer value
	capBytes = utils.RoundUpBytes(capBytes)

	// Limit is more than Required, but larger than Minimum. So we just set capcity to Minimum
	// Too small, default
	// If profile is SDP profile then no need to check for minimum size as the RoundUpBytes will giving minimum value as 1 GiB
	if capBytes < utils.MinimumVolumeSizeInBytes && profileName != SDPProfile {
		capBytes = utils.MinimumVolumeSizeInBytes
	}

	return capBytes, nil
}

// Verify that Requested volume capabailites match with what is supported by the driver
func areVolumeCapabilitiesSupported(volCaps []*csi.VolumeCapability, driverVolumeCaps []*csi.VolumeCapability_AccessMode) bool {
	isSupport := func(cap *csi.VolumeCapability) bool {
		for _, c := range driverVolumeCaps {
			if c.GetMode() == cap.AccessMode.GetMode() {
				return true
			}
		}
		return false
	}

	allSupported := true
	for _, c := range volCaps {
		if !isSupport(c) {
			allSupported = false
		}
	}
	return allSupported
}

// getVolumeParameters this function get the parameters from storage class, this also validate
// all parameters passed in storage class or not which are mandatory.
func getVolumeParameters(logger *zap.Logger, req *csi.CreateVolumeRequest, config *config.Config) (*provider.Volume, error) {
	var encrypt = "undef"
	var err error
	volume := &provider.Volume{}
	volume.Name = &req.Name
	for key, value := range req.GetParameters() {
		switch key {
		case Profile:
			if utils.ListContainsSubstr(SupportedProfile, value) {
				volume.Profile = &provider.Profile{Name: value}
			} else {
				err = fmt.Errorf("%s:<%v> unsupported profile. Supported profiles are: %v", key, value, SupportedProfile)
			}
		case Zone:
			if len(value) > ZoneNameMaxLen {
				err = fmt.Errorf("%s:<%v> exceeds %d chars", key, value, ZoneNameMaxLen)
			} else {
				volume.Az = value
			}
		case Region:
			if len(value) > RegionMaxLen {
				err = fmt.Errorf("%s:<%v> exceeds %d chars", key, value, RegionMaxLen)
			} else {
				volume.Region = value
			}
		case Tag:
			if len(value) != 0 {
				tagstr := strings.TrimSpace(value)
				volume.Tags = strings.Split(tagstr, ",")
			}

		case ResourceGroup:
			if len(value) > ResourceGroupIDMaxLen {
				err = fmt.Errorf("%s:<%v> exceeds %d chars", key, value, ResourceGroupIDMaxLen)
			}
			volume.ResourceGroup = &provider.ResourceGroup{ID: value}

		case BillingType:
			// Its not supported by RIaaS, but this is just information for the user

		case Encrypted:
			if value != TrueStr && value != FalseStr {
				err = fmt.Errorf("'<%v>' is invalid, value of '%s' should be [true|false]", value, key)
			} else {
				encrypt = value
			}
		case EncryptionKey:
			if len(value) > EncryptionKeyMaxLen {
				err = fmt.Errorf("%s: exceeds %d bytes", key, EncryptionKeyMaxLen)
			} else {
				if len(value) != 0 {
					volume.VolumeEncryptionKey = &provider.VolumeEncryptionKey{CRN: value}
				}
			}

		case ClassVersion:
			// Not needed by RIaaS, this is just info for the user
			logger.Info("Ignoring storage class parameter", zap.Any("ClassParameter", ClassVersion))

		case Generation:
			// Ignore... Provided in SC just for backward compatibility
			logger.Info("Ignoring storage class parameter, for backward compatibility", zap.Any("ClassParameter", Generation))

		case IOPS:
			// Default IOPS can be specified in Custom or sdp class
			if len(value) != 0 {
				iops := value
				volume.Iops = &iops
			}
		case Throughput: // getting throughput value from storage class if it is provided
			if len(value) != 0 {
				bandwidth, errParse := strconv.ParseInt(value, 10, 32)
				if errParse != nil {
					err = fmt.Errorf("'<%v>' is invalid, value of '%s' should be an int32 type", value, key)
				} else {
					volume.Bandwidth = int32(bandwidth)
				}
			}
		default:
			err = fmt.Errorf("<%s> is an invalid parameter", key)
		}
		if err != nil {
			logger.Error("getVolumeParameters", zap.NamedError("SC Parameters", err))
			return volume, err
		}
	}
	// If encripted is set to false
	if encrypt == FalseStr {
		volume.VolumeEncryptionKey = nil
	}

	if volume.Profile == nil {
		err = fmt.Errorf("volume profile is empty, you need to pass valid profile name")
		logger.Error("getVolumeParameters", zap.NamedError("InvalidRequest", err))
		return volume, err
	}

	// Get the requested capacity from the request
	capacityRange := req.GetCapacityRange()
	capBytes, err := getRequestedCapacity(capacityRange, volume.Profile.Name)
	if err != nil {
		err = fmt.Errorf("invalid PVC capacity size: '%v'", err)
		logger.Error("getVolumeParameters", zap.NamedError("invalid parameter", err))
		return volume, err
	}
	logger.Info("Volume size in bytes", zap.Any("capacity", capBytes))

	// Convert size/capacity in GiB, as this is needed by RIaaS
	fsSize := utils.BytesToGiB(capBytes)
	// Assign the size to volume object
	volume.Capacity = &fsSize
	logger.Info("Volume size in GiB", zap.Any("capacity", fsSize))

	// volume.Capacity should be set before calling overrideParams
	err = overrideParams(logger, req, config, volume)
	if err != nil {
		return volume, err
	}

	// Check if the provided fstype is supported one
	volumeCapabilities := req.GetVolumeCapabilities()
	if volumeCapabilities == nil {
		err = fmt.Errorf("volume capabilities are empty")
		logger.Error("overrideParams", zap.NamedError("invalid parameter", err))
		return volume, err
	}

	for _, vcap := range volumeCapabilities {
		mnt := vcap.GetMount()
		if mnt == nil {
			continue
		}
		if len(mnt.FsType) == 0 {
			volume.VolumeType = provider.VolumeType(defaultFsType)
		} else {
			if utils.ListContainsSubstr(SupportedFS, mnt.FsType) {
				volume.VolumeType = provider.VolumeType(mnt.FsType)
			} else {
				err = fmt.Errorf("unsupported fstype <%s>. Supported types: %v", mnt.FsType, SupportedFS)
			}
		}
		break
	}
	if err != nil {
		logger.Error("getVolumeParameters", zap.NamedError("InvalidParameter", err))
		return volume, err
	}

	if volume.Profile != nil && (volume.Profile.Name != CustomProfile && volume.Profile.Name != SDPProfile) {
		// Specify IOPS only for custom or SDP class
		volume.Iops = nil
	}

	//If  zone not provided in storage class parameters then we pick from the Topology
	if len(strings.TrimSpace(volume.Az)) == 0 {
		zones, err := pickTargetTopologyParams(req.GetAccessibilityRequirements())
		if err != nil {
			err = fmt.Errorf("unable to fetch zone information from topology: '%v'", err)
			logger.Error("getVolumeParameters", zap.NamedError("InvalidParameter", err))
			return volume, err
		}
		volume.Az = zones[utils.NodeZoneLabel]

	}

	return volume, nil
}

func overrideParams(logger *zap.Logger, req *csi.CreateVolumeRequest, config *config.Config, volume *provider.Volume) error {
	var encrypt = "undef"
	var err error
	if volume == nil {
		return fmt.Errorf("invalid volume parameter")
	}

	for key, value := range req.GetSecrets() {
		switch key {
		case ResourceGroup:
			if len(value) > ResourceGroupIDMaxLen {
				err = fmt.Errorf("%s:<%v> exceeds %d bytes ", key, value, ResourceGroupIDMaxLen)
			} else {
				logger.Info("override", zap.Any(ResourceGroup, value))
				volume.ResourceGroup = &provider.ResourceGroup{ID: value}
			}
		case Encrypted:
			if value != TrueStr && value != FalseStr {
				err = fmt.Errorf("<%v> is invalid, value for '%s' should be [true|false]", value, key)
			} else {
				logger.Info("override", zap.Any(Encrypted, value))
				encrypt = value
			}
		case EncryptionKey:
			if len(value) > EncryptionKeyMaxLen {
				err = fmt.Errorf("%s exceeds %d bytes", key, EncryptionKeyMaxLen)
			} else {
				if len(value) != 0 {
					logger.Info("override", zap.String("parameter", EncryptionKey))
					volume.VolumeEncryptionKey = &provider.VolumeEncryptionKey{CRN: value}
				}
			}
		case Tag:
			if len(value) != 0 {
				logger.Info("append", zap.Any(Tag, value))
				tagstr := strings.TrimSpace(value)
				secretTags := strings.Split(tagstr, ",")
				volume.Tags = append(volume.Tags, secretTags...)
			}

		case Zone:
			if len(value) > ZoneNameMaxLen {
				err = fmt.Errorf("%s:<%v> exceeds %d chars", key, value, ZoneNameMaxLen)
			} else {
				logger.Info("override", zap.Any(Zone, value))
				volume.Az = value
			}
		case Region:
			if len(value) > RegionMaxLen {
				err = fmt.Errorf("%s:<%v> exceeds %d chars", key, value, RegionMaxLen)
			} else {
				volume.Region = value
			}
		case IOPS:
			// Override IOPS only for custom or sdp class
			if len(value) != 0 {
				iops := value
				volume.Iops = &iops
			}
		case Throughput: // getting throughput value from storage class if it is provided
			if len(value) != 0 {
				bandwidth, errParse := strconv.ParseInt(value, 10, 32)
				if errParse != nil {
					err = fmt.Errorf("'<%v>' is invalid, value of '%s' should be an int32 type", value, key)
				} else {
					volume.Bandwidth = int32(bandwidth)
				}
			}
		default:
			err = fmt.Errorf("<%s> is an invalid parameter", key)
		}
		if err != nil {
			logger.Error("overrideParams", zap.NamedError("Secret Parameters", err))
			return err
		}
	}
	// Assign ResourceGroupID from config
	if volume.ResourceGroup == nil || len(volume.ResourceGroup.ID) < 1 {
		volume.ResourceGroup = &provider.ResourceGroup{ID: config.VPC.G2ResourceGroupID}
	}
	if encrypt == FalseStr {
		volume.VolumeEncryptionKey = nil
	}
	return nil
}

// checkIfVolumeExists ...
func checkIfVolumeExists(session provider.Session, vol provider.Volume, ctxLogger *zap.Logger) (*provider.Volume, error) {
	// Check if Requested Volume exists
	// Cases to check - If Volume is Not Found,  Multiple Disks with same name, or Size Don't match
	// Todo: convert to switch statement.
	var err error
	var existingVol *provider.Volume

	if vol.Name != nil && *vol.Name != "" {
		existingVol, err = session.GetVolumeByName(*vol.Name)
	} else if vol.VolumeID != "" {
		existingVol, err = session.GetVolume(vol.VolumeID)
	} else {
		return nil, fmt.Errorf("both volume name and ID are nil")
	}

	if err != nil {
		ctxLogger.Error("checkIfVolumeExists", zap.NamedError("Error", err))
		errorType := providerError.GetErrorType(err)
		switch errorType {
		case providerError.EntityNotFound:
			return nil, nil
		case providerError.RetrivalFailed:
			return nil, nil
		default:
			return nil, err
		}
	}
	// Update the region as its not getting updated in the common library because
	// RIaaS does not provide Region details
	if existingVol != nil {
		existingVol.Region = vol.Region
	}
	return existingVol, err
}

// createCSIVolumeResponse ...
func createCSIVolumeResponse(vol provider.Volume, capBytes int64, zones []string, clusterID string, region string) *csi.CreateVolumeResponse {
	var src *csi.VolumeContentSource
	if vol.SnapshotID != "" {
		src = &csi.VolumeContentSource{
			Type: &csi.VolumeContentSource_Snapshot{
				Snapshot: &csi.VolumeContentSource_SnapshotSource{
					SnapshotId: vol.SnapshotID,
				},
			},
		}
	}
	labels := map[string]string{}

	// Update labels for PV objects
	labels[VolumeIDLabel] = vol.VolumeID
	labels[VolumeCRNLabel] = vol.CRN
	labels[ClusterIDLabel] = clusterID
	labels[Tag] = strings.Join(vol.Tags, ",")
	if vol.Iops != nil && len(*vol.Iops) > 0 {
		labels[IOPSLabel] = *vol.Iops
	}

	if vol.Region != "" {
		labels[utils.NodeRegionLabel] = vol.Region
	} else {
		labels[utils.NodeRegionLabel] = region
	}
	labels[utils.NodeZoneLabel] = vol.Az

	topology := &csi.Topology{
		Segments: map[string]string{
			utils.NodeRegionLabel: labels[utils.NodeRegionLabel],
			utils.NodeZoneLabel:   labels[utils.NodeZoneLabel],
		},
	}

	// Create csi volume response
	volResp := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			CapacityBytes:      capBytes,
			VolumeId:           vol.VolumeID,
			VolumeContext:      labels,
			AccessibleTopology: []*csi.Topology{topology},
			ContentSource:      src,
		},
	}
	return volResp
}

// getAccountID ...
func getAccountID(input string) string {
	tokens := strings.Split(input, "/")

	if len(tokens) > 1 {
		return tokens[1]
	} else {
		return ""
	}
}

// getSnapshotAndAccountIDsFromCRN ...
func getSnapshotAndAccountIDsFromCRN(crn string) (string, string) {
	// This method will be able to handle either crn is actual crn or caller passed snapshot ID also
	// expected CRN -> crn:v1:service:public:is:us-south:a/c468d8642937fecd8a0860fe0f379bf9::snapshot:r006-1234fe0c-3d9b-4c95-a6d1-8e0d4bcb6ecb
	// or crn passed as sanpshotID like r006-1234fe0c-3d9b-4c95-a6d1-8e0d4bcb6ecb
	crnTokens := strings.Split(crn, ":")

	if len(crnTokens) > 9 {
		return crnTokens[len(crnTokens)-1], getAccountID(crnTokens[len(crnTokens)-4])
	}
	return crn, "" // assuming that crn will contain only snapshotID
}

// createCSISnapshotResponse ...
func createCSISnapshotResponse(snapshot provider.Snapshot) *csi.CreateSnapshotResponse {
	ts := timestamppb.New(snapshot.SnapshotCreationTime)
	return &csi.CreateSnapshotResponse{
		Snapshot: &csi.Snapshot{
			SnapshotId:     snapshot.SnapshotCRN,
			SourceVolumeId: snapshot.VolumeID,
			SizeBytes:      snapshot.SnapshotSize,
			CreationTime:   ts,
			ReadyToUse:     snapshot.ReadyToUse,
		},
	}
}

func createControllerPublishVolumeResponse(volumeAttachmentResponse provider.VolumeAttachmentResponse, extraPublishInfo map[string]string) *csi.ControllerPublishVolumeResponse {
	publishContext := map[string]string{
		PublishInfoVolumeID:   volumeAttachmentResponse.VolumeID,
		PublishInfoNodeID:     volumeAttachmentResponse.InstanceID,
		PublishInfoStatus:     volumeAttachmentResponse.Status,
		PublishInfoDevicePath: volumeAttachmentResponse.VPCVolumeAttachment.DevicePath,
	}
	// append extraPublishInfo
	for k, v := range extraPublishInfo {
		publishContext[k] = v
	}
	return &csi.ControllerPublishVolumeResponse{
		PublishContext: publishContext,
	}
}

func pickTargetTopologyParams(top *csi.TopologyRequirement) (map[string]string, error) {
	prefTopologyParams, err := getPrefedTopologyParams(top.GetPreferred())
	if err != nil {
		return nil, fmt.Errorf("could not get zones from preferred topology: %v", err)
	}

	return prefTopologyParams, nil
}

func getPrefedTopologyParams(topList []*csi.Topology) (map[string]string, error) {
	for _, top := range topList {
		segment := top.GetSegments()
		if segment != nil {
			return segment, nil
		}
	}
	return nil, fmt.Errorf("preferred topologies specified but no segments")
}

/*
1.) IF user does not given the value DEFAULT_SNAPSHOT_CREATE_DELAY mins
2.) IF user has given more than MAX_SNAPSHOT_CREATE_DELAY default is MAX_SNAPSHOT_CREATE_DELAY
3.) In case of any invalid value DEFAULT_SNAPSHOT_CREATE_DELAY mins
*/
func getMaxDelaySnapshotCreate(ctxLogger *zap.Logger) int {
	userDelayEnv := os.Getenv("CUSTOM_SNAPSHOT_CREATE_DELAY")
	if userDelayEnv == "" {
		return DEFAULT_SNAPSHOT_CREATE_DELAY
	}

	customSnapshotCreateDelay, err := strconv.Atoi(userDelayEnv)
	if err != nil {
		ctxLogger.Warn("Error while processing CUSTOM_SNAPSHOT_CREATE_DELAY value.Expecting integer value in seconds", zap.Any("CUSTOM_SNAPSHOT_CREATE_DELAY", customSnapshotCreateDelay), zap.Any("Considered value", DEFAULT_SNAPSHOT_CREATE_DELAY), zap.Error(err))
		return DEFAULT_SNAPSHOT_CREATE_DELAY // min 300 seconds default
	}
	if customSnapshotCreateDelay > MAX_SNAPSHOT_CREATE_DELAY {
		ctxLogger.Warn("CUSTOM_SNAPSHOT_CREATE_DELAY value cannot exceed the limits", zap.Any("CUSTOM_SNAPSHOT_CREATE_DELAY", customSnapshotCreateDelay), zap.Any("Limit value", MAX_SNAPSHOT_CREATE_DELAY))
		return MAX_SNAPSHOT_CREATE_DELAY // max 900 seconds
	}

	return customSnapshotCreateDelay
}

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

//Package ibmcsidriver ...
package ibmcsidriver

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/IBM/ibm-csi-common/pkg/utils"
	"github.com/IBM/ibmcloud-volume-interface/config"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	providerError "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"go.uber.org/zap"
)

// Capacity vs IOPS range for Custom Class
type classRange struct {
	minSize int
	maxSize int
	minIops int
	maxIops int
}

// Range as per IBM volume provider Storage
var customCapacityIopsRanges = []classRange{
	{10, 39, 100, 1000},
	{40, 79, 100, 2000},
	{80, 99, 100, 4000},
	{100, 499, 100, 6000},
	{500, 999, 100, 10000},
	{1000, 1999, 100, 20000},
}

// normalize the requested capacity(in GiB) to what is supported by the driver
func getRequestedCapacity(capRange *csi.CapacityRange) (int64, error) {
	// Input is in bytes from csi
	var capBytes int64
	// Default case where nothing is set
	if capRange == nil {
		capBytes = utils.MinimumVolumeSizeInBytes
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
	if capBytes < utils.MinimumVolumeSizeInBytes {
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

//getVolumeParameters this function get the parameters from storage class, this also validate
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
				volume.VPCVolume.Profile = &provider.Profile{Name: value}
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
			if len(value) > TagMaxLen {
				err = fmt.Errorf("%s:<%v> exceeds %d chars", key, value, TagMaxLen)
			}
			if len(value) != 0 {
				volume.VPCVolume.Tags = []string{value}
			}

		case ResourceGroup:
			if len(value) > ResourceGroupIDMaxLen {
				err = fmt.Errorf("%s:<%v> exceeds %d chars", key, value, ResourceGroupIDMaxLen)
			}
			volume.VPCVolume.ResourceGroup = &provider.ResourceGroup{ID: value}

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
					volume.VPCVolume.VolumeEncryptionKey = &provider.VolumeEncryptionKey{CRN: value}
				}
			}

		case ClassVersion:
			// Not needed by RIaaS, this is just info for the user
			logger.Info("Ignoring storage class parameter", zap.Any("ClassParameter", ClassVersion))

		case Generation:
			// Ignore... Provided in SC just for backward compatibility
			logger.Info("Ignoring storage class parameter, for backward compatibility", zap.Any("ClassParameter", Generation))

		case IOPS:
			// Default IOPS can be specified in Custom class
			if len(value) != 0 {
				iops := value
				volume.Iops = &iops
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
		volume.VPCVolume.VolumeEncryptionKey = nil
	}

	// Get the requested capacity from the request
	capacityRange := req.GetCapacityRange()
	capBytes, err := getRequestedCapacity(capacityRange)
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

	if volume.VPCVolume.Profile != nil && volume.VPCVolume.Profile.Name != CustomProfile {
		// Specify IOPS only for custom class
		volume.Iops = nil
	}

	//if zone is not given in SC parameters, but region is given, error out
	if len(strings.TrimSpace(volume.Az)) == 0 && len(strings.TrimSpace(volume.Region)) != 0 {
		err = fmt.Errorf("zone parameter is empty in storage class for region %s", strings.TrimSpace(volume.Region))
		return volume, err
	}

	//If both zone and region not provided in storage class parameters then we pick from the Topology
	//if zone is provided but region is not provided, fetch region for specified zone
	if len(strings.TrimSpace(volume.Region)) == 0 {
		zones, err := pickTargetTopologyParams(req.GetAccessibilityRequirements())
		if err != nil {
			err = fmt.Errorf("unable to fetch zone information from topology: '%v'", err)
			logger.Error("getVolumeParameters", zap.NamedError("InvalidParameter", err))
			return volume, err
		}
		volume.Region = zones[utils.NodeRegionLabel]
		if len(strings.TrimSpace(volume.Az)) == 0 {
			volume.Az = zones[utils.NodeZoneLabel]
		}

	}

	return volume, nil
}

// Validate size and iops for custom class
func isValidCapacityIOPS4CustomClass(size int, iops int) (bool, error) {
	var ind = -1
	for i, entry := range customCapacityIopsRanges {
		if size >= entry.minSize && size <= entry.maxSize {
			ind = i
			break
		}
	}

	if ind < 0 {
		return false, fmt.Errorf("invalid PVC size for custom class: <%v>. Should be in range [%d - %d]GiB",
			size, utils.MinimumVolumeDiskSizeInGb, utils.MaximumVolumeDiskSizeInGb)
	}

	if iops < customCapacityIopsRanges[ind].minIops || iops > customCapacityIopsRanges[ind].maxIops {
		return false, fmt.Errorf("invalid IOPS: <%v> for capacity: <%vGiB>. Should be in range [%d - %d]",
			iops, size, customCapacityIopsRanges[ind].minIops, customCapacityIopsRanges[ind].maxIops)
	}
	return true, nil
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
				volume.VPCVolume.ResourceGroup = &provider.ResourceGroup{ID: value}
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
					volume.VPCVolume.VolumeEncryptionKey = &provider.VolumeEncryptionKey{CRN: value}
				}
			}
		case Tag:
			if len(value) > TagMaxLen {
				err = fmt.Errorf("%s:<%v> exceeds %d chars", key, value, TagMaxLen)
			} else {
				if len(value) != 0 {
					logger.Info("append", zap.Any(Tag, value))
					volume.VPCVolume.Tags = append(volume.VPCVolume.Tags, value)
				}
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
			// Override IOPS only for custom class
			if volume.Capacity != nil && volume.VPCVolume.Profile != nil && volume.VPCVolume.Profile.Name == "custom" {
				var iops int
				var check bool
				iops, err = strconv.Atoi(value)
				if err != nil {
					err = fmt.Errorf("%v:<%v> invalid value", key, value)
				} else {
					if check, err = isValidCapacityIOPS4CustomClass(*(volume.Capacity), iops); check {
						iopsStr := value
						logger.Info("override", zap.Any(IOPS, value))
						volume.Iops = &iopsStr
					}
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
	if volume.VPCVolume.ResourceGroup == nil || len(volume.VPCVolume.ResourceGroup.ID) < 1 {
		volume.VPCVolume.ResourceGroup = &provider.ResourceGroup{ID: config.VPC.ResourceGroupID}
	}
	if encrypt == FalseStr {
		volume.VPCVolume.VolumeEncryptionKey = nil
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
func createCSIVolumeResponse(vol provider.Volume, capBytes int64, zones []string, clusterID string) *csi.CreateVolumeResponse {
	labels := map[string]string{}

	// Update labels for PV objects
	labels[VolumeIDLabel] = vol.VolumeID
	labels[VolumeCRNLabel] = vol.CRN
	labels[ClusterIDLabel] = clusterID
	labels[Tag] = strings.Join(vol.Tags, ",")
	if vol.Iops != nil && len(*vol.Iops) > 0 {
		labels[IOPSLabel] = *vol.Iops
	}
	labels[utils.NodeRegionLabel] = vol.Region
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
		},
	}
	return volResp
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

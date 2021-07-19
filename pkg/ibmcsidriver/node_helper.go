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
	"os"
	"os/exec"
	"path/filepath"
	"time"

	commonError "github.com/IBM/ibm-csi-common/pkg/messages"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"go.uber.org/zap"
)

// findDevicePath finds path of device and verifies its existence
func (csiNS *CSINodeServer) findDevicePathSource(ctxLogger *zap.Logger, devicePath string, volumeID string /*TODO may be required in future*/) (string, error) {
	ctxLogger.Info("CSINodeServer-findDevicePathSource...")
	exists, err := csiNS.Mounter.ExistsPath(devicePath)
	if err != nil || !exists {
		ctxLogger.Warn("Device path not found, trying to fix by udevadm trigger", zap.String("DevicePath", devicePath))
		if err = csiNS.udevadmTrigger(ctxLogger); err != nil {
			ctxLogger.Error("Failed to execute udevadm trigger, will try to check device path again", zap.Error(err))
		}
		// Re-verifying device path and returning error accordingly
		exists, err = csiNS.Mounter.ExistsPath(devicePath)
		if err != nil {
			return "", err
		}
	}
	// If the path exists, assume it is not nvme device
	if exists {
		return devicePath, nil
	}
	ctxLogger.Warn("Device Path is nvme. Try to find nvme device")
	return devicePath, nil
	// TODO  Find NVMe path. Currently volume provider instance does not have NVMe
	//For example, /dev/disk/by-uuid/e75b09ee-27d5-491a-85cd-c380f0e8ef5e -> ../../nvme2n1
}

func (csiNS *CSINodeServer) processMount(ctxLogger *zap.Logger, requestID, stagingTargetPath, targetPath, fsType string, options []string) (*csi.NodePublishVolumeResponse, error) {
	stagingTargetPathField := zap.String("stagingTargetPath", stagingTargetPath)
	targetPathField := zap.String("targetPath", targetPath)
	fsTypeField := zap.String("fsType", fsType)
	optionsField := zap.Reflect("options", options)
	ctxLogger.Info("CSINodeServer-processMount...", stagingTargetPathField, targetPathField, fsTypeField, optionsField)
	if err := csiNS.Mounter.Interface.MakeDir(targetPath); err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.TargetPathCreateFailed, requestID, err, targetPath)
	}
	err := csiNS.Mounter.Interface.Mount(stagingTargetPath, targetPath, fsType, options)
	if err != nil {
		notMnt, mntErr := csiNS.Mounter.Interface.IsLikelyNotMountPoint(targetPath)
		if mntErr != nil {
			return nil, commonError.GetCSIError(ctxLogger, commonError.MountPointValidateError, requestID, mntErr, targetPath)
		}
		if !notMnt {
			if mntErr = csiNS.Mounter.Interface.Unmount(targetPath); mntErr != nil {
				return nil, commonError.GetCSIError(ctxLogger, commonError.UnmountFailed, requestID, mntErr, targetPath)
			}
			notMnt, mntErr = csiNS.Mounter.Interface.IsLikelyNotMountPoint(targetPath)
			if mntErr != nil {
				return nil, commonError.GetCSIError(ctxLogger, commonError.MountPointValidateError, requestID, mntErr, targetPath)
			}
			if !notMnt {
				// This is very odd, we don't expect it.  We'll try again next sync loop.
				return nil, commonError.GetCSIError(ctxLogger, commonError.UnmountFailed, requestID, err, targetPath)
			}
		}
		_ = os.Remove(targetPath)
		return nil, commonError.GetCSIError(ctxLogger, commonError.CreateMountTargetFailed, requestID, err, targetPath)
	}

	ctxLogger.Info("CSINodeServer-processMount successfully mounted", stagingTargetPathField, targetPathField, fsTypeField, optionsField)
	return &csi.NodePublishVolumeResponse{}, nil
}

//This will handle raw block volume mounts
//Incase of RAW volume mount, the Target will be devicefilepath  and NOT a mount directory.
// The mountType is "bind" mount and will not specify any FORMAT(e.g ext4, ext3..)
// e.g SOURCE (volume provider attached device on Host): /dev/xvde
// e.g TARGET (SoftLink to User defined POD device /dev/sda) : "/var/data/kubelet/plugins/kubernetes.io/csi/volumeDevices/publish/pvc-9b82dced-fcd6-4181-968e-ae269e0f2311"
func (csiNS *CSINodeServer) processMountForBlock(ctxLogger *zap.Logger, requestID, devicePath, target, volumeID string, options []string) (*csi.NodePublishVolumeResponse, error) {
	ctxLogger.Info("CSINodeServer-processMountForBlock", zap.String("devicePath", devicePath), zap.String("target", target), zap.Reflect("options", options))

	//get devicepath to be used as mountpoint source
	if len(devicePath) == 0 {
		return nil, commonError.GetCSIError(ctxLogger, commonError.EmptyDevicePath, requestID, nil)
	}
	// Check source Path existence
	source, err := csiNS.findDevicePathSource(ctxLogger, devicePath, volumeID)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.DevicePathFindFailed, requestID, err, devicePath)
	}
	ctxLogger.Info("Found device path ", zap.String("devicePath", devicePath), zap.String("source", source))

	targetDir := filepath.Dir(target)
	exists, err := csiNS.Mounter.ExistsPath(targetDir)
	if err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.TargetPathCheckFailed, requestID, err, targetDir)
	}

	if !exists {
		if err := csiNS.Mounter.MakeDir(targetDir); err != nil {
			return nil, commonError.GetCSIError(ctxLogger, commonError.TargetPathCreateFailed, requestID, err, targetDir)
		}
	}

	// Create the mount point as a file since bind mount device node requires it to be a file
	ctxLogger.Info("Making target file", zap.String("target", target))
	err = csiNS.Mounter.MakeFile(target)
	if err != nil {
		if removeErr := os.Remove(target); removeErr != nil {
			return nil, commonError.GetCSIError(ctxLogger, commonError.RemoveMountTargetFailed, requestID, removeErr, target)
		}
		return nil, commonError.GetCSIError(ctxLogger, commonError.CreateMountTargetFailed, requestID, err, target)
	}

	ctxLogger.Info("Mounting source to target", zap.String("source", source), zap.String("target", target))
	if err := csiNS.Mounter.Interface.Mount(source, target, "", options); err != nil {
		if removeErr := os.Remove(target); removeErr != nil {
			return nil, commonError.GetCSIError(ctxLogger, commonError.RemoveMountTargetFailed, requestID, removeErr, target)
		}
		return nil, commonError.GetCSIError(ctxLogger, commonError.MountFailed, requestID, err, source, target)
	}

	ctxLogger.Info("Block volume mounted successfully", zap.String("source", source), zap.String("target", target))
	return &csi.NodePublishVolumeResponse{}, nil
}

func (csiNS *CSINodeServer) udevadmTrigger(ctxLogger *zap.Logger) error {
	ctxLogger.Info("CSINodeServer-udevadmTrigger refreshing all devices...")
	out, err := exec.Command(
		"udevadm",
		"trigger").CombinedOutput()
	if err != nil {
		return fmt.Errorf("udevadmTrigger: udevadm trigger failed, output %s, error: %v", string(out), err)
	}

	// Sleep for 20 seconds so that udevadm trigger will do its magic
	duration, _ := time.ParseDuration("20s")
	time.Sleep(duration)

	ctxLogger.Info("udevadmTrigger: Successfully executed udevadm trigger to referesh all devices.")
	return nil
}

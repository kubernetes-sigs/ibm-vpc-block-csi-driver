/*
Copyright 2021-2026 The Kubernetes Authors.

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
	"path/filepath"
	"strconv"
	"time"

	commonError "github.com/IBM/ibm-csi-common/pkg/messages"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"go.uber.org/zap"
)

// findDevicePath finds path of device and verifies its existence
func (csiNS *CSINodeServer) findDevicePathSource(ctxLogger *zap.Logger, devicePath string, volumeID string /*TODO may be required in future*/) (string, error) {
	ctxLogger.Info("CSINodeServer-findDevicePathSource...", zap.String("devicePath", devicePath), zap.String("volumeID", volumeID))

	// Validate input parameters
	if devicePath == "" {
		ctxLogger.Error("Device path cannot be empty")
		return "", fmt.Errorf("device path cannot be empty")
	}

	// First attempt: Check if device path exists
	exists, err := csiNS.Mounter.PathExists(devicePath)
	if err != nil {
		// Real error occurred while checking path (permissions, I/O error, etc.)
		ctxLogger.Error("Failed to check device path existence",
			zap.String("devicePath", devicePath),
			zap.Error(err))
		return "", fmt.Errorf("failed to check device path existence: %w", err)
	}

	if exists {
		ctxLogger.Info("Device path found successfully", zap.String("devicePath", devicePath))
		return devicePath, nil
	}

	// Device path doesn't exist - try udevadm trigger to refresh device nodes
	ctxLogger.Warn("Device path not found on first attempt, triggering udevadm to refresh device nodes",
		zap.String("devicePath", devicePath))

	if err = csiNS.udevadmTrigger(ctxLogger); err != nil {
		// udevadm trigger failed - this is critical as device won't appear
		ctxLogger.Error("Failed to execute udevadm trigger - device path cannot be recovered",
			zap.String("devicePath", devicePath),
			zap.Error(err),
			zap.String("recommendation", "Ensure udevadm is installed and accessible"))
		return "", fmt.Errorf("device path not found and udevadm trigger failed (device: %s): %w", devicePath, err)
	}

	// Wait for device path to appear with retry logic
	maxRetries := 15                 // Default: 15 retries
	retryInterval := 2 * time.Second // Default: 2 seconds between retries

	// Allow configuration via environment variables
	if retriesEnv := os.Getenv("UDEVADM_MAX_RETRIES"); retriesEnv != "" {
		if retries, err := strconv.Atoi(retriesEnv); err == nil && retries > 0 {
			maxRetries = retries
		}
	}
	if intervalEnv := os.Getenv("UDEVADM_RETRY_INTERVAL"); intervalEnv != "" {
		if interval, err := time.ParseDuration(intervalEnv); err == nil {
			retryInterval = interval
		}
	}

	ctxLogger.Info("Waiting for device path to appear after udevadm trigger",
		zap.String("devicePath", devicePath),
		zap.Int("maxRetries", maxRetries),
		zap.Duration("retryInterval", retryInterval))

	if err := csiNS.waitForDevicePath(ctxLogger, devicePath, maxRetries, retryInterval); err != nil {
		// Device path still doesn't exist after retries
		// This could be an NVMe device or the device is not attached properly
		ctxLogger.Error("Device path not found even after udevadm trigger and retries - refusing to proceed",
			zap.String("devicePath", devicePath),
			zap.String("volumeID", volumeID),
			zap.String("possibleCause", "Device may be NVMe (not yet supported) or not properly attached"),
			zap.String("action", "Verify volume attachment and device path"),
			zap.Error(err))
		return "", fmt.Errorf("device path not found: %s (volume: %s). Possible causes: NVMe device (not supported), device not attached, or incorrect device path: %w", devicePath, volumeID, err)
	}

	ctxLogger.Info("Device path found after udevadm trigger and retry",
		zap.String("devicePath", devicePath))
	return devicePath, nil
	// TODO: Implement NVMe device path resolution when NVMe support is added
	// For example, /dev/disk/by-uuid/e75b09ee-27d5-491a-85cd-c380f0e8ef5e -> ../../nvme2n1
}

func (csiNS *CSINodeServer) processMount(ctxLogger *zap.Logger, requestID, stagingTargetPath, targetPath, fsType string, options []string) (*csi.NodePublishVolumeResponse, error) {
	stagingTargetPathField := zap.String("stagingTargetPath", stagingTargetPath)
	targetPathField := zap.String("targetPath", targetPath)
	fsTypeField := zap.String("fsType", fsType)
	optionsField := zap.Reflect("options", options)
	ctxLogger.Info("CSINodeServer-processMount...", stagingTargetPathField, targetPathField, fsTypeField, optionsField)
	if err := csiNS.Mounter.MakeDir(targetPath); err != nil {
		return nil, commonError.GetCSIError(ctxLogger, commonError.TargetPathCreateFailed, requestID, err, targetPath)
	}
	err := csiNS.Mounter.Mount(stagingTargetPath, targetPath, fsType, options)
	if err != nil {
		notMnt, mntErr := csiNS.Mounter.IsLikelyNotMountPoint(targetPath)
		if mntErr != nil {
			return nil, commonError.GetCSIError(ctxLogger, commonError.MountPointValidateError, requestID, mntErr, targetPath)
		}
		if !notMnt {
			if mntErr = csiNS.Mounter.Unmount(targetPath); mntErr != nil {
				return nil, commonError.GetCSIError(ctxLogger, commonError.UnmountFailed, requestID, mntErr, targetPath)
			}
			notMnt, mntErr = csiNS.Mounter.IsLikelyNotMountPoint(targetPath)
			if mntErr != nil {
				return nil, commonError.GetCSIError(ctxLogger, commonError.MountPointValidateError, requestID, mntErr, targetPath)
			}
			if !notMnt {
				// This is very odd, we don't expect it.  We'll try again next sync loop.
				return nil, commonError.GetCSIError(ctxLogger, commonError.UnmountFailed, requestID, err, targetPath)
			}
		}
		err = os.Remove(targetPath)
		if err != nil {
			ctxLogger.Warn("processMount: Remove targePath Failed", zap.String("targetPath", targetPath), zap.Error(err))
		}
		return nil, commonError.GetCSIError(ctxLogger, commonError.CreateMountTargetFailed, requestID, err, targetPath)
	}

	ctxLogger.Info("CSINodeServer-processMount successfully mounted", stagingTargetPathField, targetPathField, fsTypeField, optionsField)
	return &csi.NodePublishVolumeResponse{}, nil
}

// This will handle raw block volume mounts
// Incase of RAW volume mount, the Target will be devicefilepath  and NOT a mount directory.
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
	exists, err := csiNS.Mounter.PathExists(targetDir)
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
	if err := csiNS.Mounter.Mount(source, target, "", options); err != nil {
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

	// Use the mounter's executor for better testability
	executor := csiNS.Mounter.GetSafeFormatAndMount().Exec

	// Step 1: Trigger udev to refresh device nodes
	triggerCmd := executor.Command("udevadm", "trigger")
	out, err := triggerCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("udevadmTrigger: udevadm trigger failed, output %s, error: %v", string(out), err)
	}
	ctxLogger.Info("udevadm trigger executed successfully")

	// Step 2: Wait for udev event queue to settle
	// This is more efficient than polling - waits for udev to finish processing events
	settleTimeout := "30" // Default 30 seconds timeout
	if envTimeout := os.Getenv("UDEVADM_SETTLE_TIMEOUT"); envTimeout != "" {
		settleTimeout = envTimeout
	}

	ctxLogger.Info("Waiting for udev event queue to settle",
		zap.String("timeout", settleTimeout+"s"))

	settleCmd := executor.Command("udevadm", "settle", "--timeout="+settleTimeout)
	settleOut, settleErr := settleCmd.CombinedOutput()
	if settleErr != nil {
		// udevadm settle failure is not critical - we'll fall back to retry logic
		ctxLogger.Warn("udevadm settle failed, will use retry fallback",
			zap.Error(settleErr),
			zap.String("output", string(settleOut)))
	} else {
		ctxLogger.Info("udev event queue settled successfully")
	}

	return nil
}

// waitForDevicePath polls for device path existence with retry logic
func (csiNS *CSINodeServer) waitForDevicePath(ctxLogger *zap.Logger, devicePath string, maxRetries int, retryInterval time.Duration) error {
	ctxLogger.Info("Waiting for device path to appear",
		zap.String("devicePath", devicePath),
		zap.Int("maxRetries", maxRetries),
		zap.Duration("retryInterval", retryInterval))

	for attempt := 1; attempt <= maxRetries; attempt++ {
		exists, err := csiNS.Mounter.PathExists(devicePath)
		if err != nil {
			ctxLogger.Warn("Error checking device path existence",
				zap.Int("attempt", attempt),
				zap.Int("maxRetries", maxRetries),
				zap.String("devicePath", devicePath),
				zap.Error(err))
			// Continue retrying even on errors as they might be transient
		} else if exists {
			ctxLogger.Info("Device path found successfully",
				zap.String("devicePath", devicePath),
				zap.Int("attempt", attempt),
				zap.Duration("totalWaitTime", time.Duration(attempt-1)*retryInterval))
			return nil
		}

		if attempt < maxRetries {
			ctxLogger.Debug("Device path not found yet, retrying",
				zap.Int("attempt", attempt),
				zap.Int("maxRetries", maxRetries),
				zap.String("devicePath", devicePath))
			time.Sleep(retryInterval)
		}
	}

	totalWaitTime := time.Duration(maxRetries) * retryInterval
	return fmt.Errorf("device path %s did not appear after %d attempts over %v",
		devicePath, maxRetries, totalWaitTime)
}

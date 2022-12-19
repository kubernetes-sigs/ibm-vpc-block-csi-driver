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
	"strconv"
	"strings"
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

// maxRetryAttempt ...
var maxRetryAttempt = 10

// maxRetryGap ...
var maxRetryGap = 60

// minVPCRetryGap ...
var minVPCRetryGap = 3

// minVPCRetryGapAttempt ...
var minVPCRetryGapAttempt = 3

// maxRetryAttempt ...
var maxVPCRetryAttempt = 46

//ConstantRetryGap ...
const (
	ConstantRetryGap        = 10 // seconds
	ConstMaxVPCRetryAttempt = 46
	ConstMinVPCRetryGap     = 3 //seconds
	SnapshotIDNotFound      = "snapshot_id_not_found"
	SnapshotNotFound        = "snapshots_not_found"
)

var volumeIDPartsCount = 5

var skipErrorCodes = map[string]bool{
	"validation_invalid_name":              true,
	"volume_capacity_max":                  true,
	"volume_id_invalid":                    true,
	"volume_profile_iops_invalid":          true,
	"volume_capacity_zero_or_negative":     true,
	"not_found":                            true,
	"volume_id_not_found":                  true,
	"volume_name_not_found":                true,
	"volume_profile_capacity_iops_invalid": true,
	"internal_error":                       false,
	"invalid_route":                        false,
	SnapshotNotFound:                       true,
	"snapshots_not_authorized":             true,
	SnapshotIDNotFound:                     true,

	// IKS ms error code for skip re-try
	"ST0008": true, //resources not found
	"ST0005": true, //worker node could not be found
	"ST0014": true, // Required parameter missing or invalid
	"ST0015": true, // Required parameter missing
	"ST0016": true, // Tagging failed .. Do not repeat
	"P4106":  true, // Instnace not found
	"P4107":  true, // Volume not found
	"P4109":  true, // Volume attachment not found
}

// retry ...
func retry(logger *zap.Logger, retryfunc func() error) error {
	var err error
	retryGap := 10
	for i := 0; i < maxRetryAttempt; i++ {
		if i > 0 {
			time.Sleep(time.Duration(retryGap) * time.Second)
		}
		err = retryfunc()
		if err != nil {
			logger.Info("err object is not nil", zap.Reflect("ERR", err))
			//Skip retry for the below type of Errors
			modelError, ok := err.(*models.Error)
			if !ok {
				continue
			}
			if skipRetry(modelError) {
				break
			}
			if i >= 1 {
				retryGap = 2 * retryGap
				if retryGap > maxRetryGap {
					retryGap = maxRetryGap
				}
			}
			if (i + 1) < maxRetryAttempt {
				logger.Info("Error while executing the function. Re-attempting execution ..", zap.Int("attempt..", i+2), zap.Int("retry-gap", retryGap), zap.Int("max-retry-Attempts", maxRetryAttempt), zap.Error(err))
			}
			continue
		}
		return err
	}
	return err
}

// skipRetry skip retry as per listed error codes
func skipRetry(err *models.Error) bool {
	for _, errorItem := range err.Errors {
		skipStatus, ok := skipErrorCodes[string(errorItem.Code)]
		if ok {
			return skipStatus
		}
	}
	return false
}

// SkipRetryForIKS skip retry as per listed error codes
func SkipRetryForIKS(err error) bool {
	iksError, iksok := err.(*models.IksError)
	if iksok {
		skipStatus, ok := skipErrorCodes[iksError.Code]
		if ok {
			return skipStatus
		}
	}
	return false
}

// skipRetryForObviousErrors skip retry as per listed error codes
func skipRetryForObviousErrors(err error, isIKS bool) bool {
	// Only for storage-api ms related calls error
	if isIKS {
		return SkipRetryForIKS(err)
	}

	// Only for RIaaS attachment related calls error
	riaasError, ok := err.(*models.Error)
	if ok {
		return skipRetry(riaasError)
	}
	return false
}

// FlexyRetry ...
type FlexyRetry struct {
	maxRetryAttempt       int
	maxRetryGap           int
	minVPCRetryGap        int
	minVPCRetryGapAttempt int
	maxVPCRetryAttempt    int
}

// NewFlexyRetryDefault ...
func NewFlexyRetryDefault() FlexyRetry {
	return FlexyRetry{
		// Default values as we configuration
		maxRetryAttempt:       maxRetryAttempt,
		maxRetryGap:           maxRetryGap,
		minVPCRetryGap:        minVPCRetryGap,
		minVPCRetryGapAttempt: minVPCRetryGapAttempt,
		maxVPCRetryAttempt:    maxVPCRetryAttempt,
	}
}

// NewFlexyRetry ...
func NewFlexyRetry(maxRtyAtmpt int, maxrRtyGap int) FlexyRetry {
	return FlexyRetry{
		maxRetryAttempt: maxRtyAtmpt,
		maxRetryGap:     maxrRtyGap,
	}
}

// FlexyRetry ...
func (fRetry *FlexyRetry) FlexyRetry(logger *zap.Logger, funcToRetry func() (error, bool)) error {
	var err error
	var stopRetry bool
	retryGap := 10
	for i := 0; i < fRetry.maxRetryAttempt; i++ {
		if i > 0 {
			time.Sleep(time.Duration(retryGap) * time.Second)
		}
		// Call function which required retry, retry is decided by function itself
		err, stopRetry = funcToRetry()
		if stopRetry {
			break
		}

		// Update retry gap as per exponentioal
		if i >= 1 {
			retryGap = 2 * retryGap
			if retryGap > fRetry.maxRetryGap {
				retryGap = fRetry.maxRetryGap
			}
		}
		if (i + 1) < fRetry.maxRetryAttempt {
			logger.Info("UNEXPECTED RESULT, Re-attempting execution ..", zap.Int("attempt..", i+2),
				zap.Int("retry-gap", retryGap), zap.Int("max-retry-Attempts", fRetry.maxRetryAttempt),
				zap.Bool("stopRetry", stopRetry), zap.Error(err))
		}
	}
	return err
}

// FlexyRetryWithConstGap ...
func (fRetry *FlexyRetry) FlexyRetryWithConstGap(logger *zap.Logger, funcToRetry func() (error, bool)) error {
	var err error
	var stopRetry bool
	// lets have more number of try for wait for attach and detach specially
	totalAttempt := fRetry.maxRetryAttempt * 4 // 40 time as per default values i.e 400 seconds
	for i := 0; i < totalAttempt; i++ {
		if i > 0 {
			time.Sleep(time.Duration(ConstantRetryGap) * time.Second)
		}
		// Call function which required retry, retry is decided by function itself
		err, stopRetry = funcToRetry()
		if stopRetry {
			break
		}

		if (i + 1) < totalAttempt {
			logger.Info("UNEXPECTED RESULT from FlexyRetryWithConstGap, Re-attempting execution ..", zap.Int("attempt..", i+2),
				zap.Int("retry-gap", ConstantRetryGap), zap.Int("max-retry-Attempts", totalAttempt),
				zap.Bool("stopRetry", stopRetry), zap.Error(err))
		}
	}
	return err
}

// FlexyRetryWithCustomGap ...
/*
Flow
1.) First attempt is immediately after attach was done.
2.) MinVPCRetryGapAttempts will be done with interval MinVPCRetryGap
3.) 2*MinVPCRetryGapAttempts will be done with interval of MinVPCRetryGap*2 ( it will default to 10 sec if it more than 10)
4.) Remaining attempts will be done with interval of 10 secs
*/
func (fRetry *FlexyRetry) FlexyRetryWithCustomGap(logger *zap.Logger, funcToRetry func() (error, bool)) error {
	var err error
	var stopRetry bool
	var interimRetryGap = (2 * fRetry.minVPCRetryGap)               //InteriRetryGap is always 2 * minVPCRetryGap
	maxInterimRetryGapAttempt := (3 * fRetry.minVPCRetryGapAttempt) //interimRetryGapAttempts will be 2 * minVPCRetryGapAttempt so the last attempt would be interimRetryGapAttempts + minVPCRetryGapAttempt
	retryGap := fRetry.minVPCRetryGap
	totalAttempt := fRetry.maxVPCRetryAttempt // Default = 46, as per default values 3 times 3 sec + 6 times 6 sec + 37 times 10 sec i.e 415 seconds

	//If interimRetryGap is resulting in value more than 10 secs lets default to 10 secs
	if interimRetryGap > ConstantRetryGap {
		interimRetryGap = ConstantRetryGap
	}

	for i := 0; i <= totalAttempt; i++ {
		if i > 0 {
			time.Sleep(time.Duration(retryGap) * time.Second)
		}

		// Call function which required retry, retry is decided by function itself
		err, stopRetry = funcToRetry()
		if stopRetry {
			break
		}

		//First fRetry.minVPCRetryGapAttempt attempts fRetry.minVPCRetryGap second and next (2 * fRetry.minVPCRetryGap)  attempts (2 * fRetry.minVPCRetryGap) second
		if i == fRetry.minVPCRetryGapAttempt {
			retryGap = interimRetryGap
		}

		//Remaining attempts 10 seconds
		if i == maxInterimRetryGapAttempt {
			retryGap = ConstantRetryGap // 10 seconds
		}

		if (i + 1) < totalAttempt {
			logger.Info("UNEXPECTED RESULT from FlexyRetryWithCustomGap, Re-attempting execution ..", zap.Int("attempt..", i+2),
				zap.Int("retry-gap", retryGap), zap.Int("max-retry-Attempts", totalAttempt),
				zap.Bool("stopRetry", stopRetry), zap.Error(err))
		}
	}
	return err
}

// ToInt ...
func ToInt(valueInInt string) int {
	value, err := strconv.Atoi(valueInInt)
	if err != nil {
		return 0
	}
	return value
}

// ToInt64 ...
func ToInt64(valueInInt string) int64 {
	value, err := strconv.ParseInt(valueInInt, 10, 64)
	if err != nil {
		return 0
	}
	return value
}

// FromProviderToLibVolume converting vpc provider volume type to generic lib volume type
func FromProviderToLibVolume(vpcVolume *models.Volume, logger *zap.Logger) (libVolume *provider.Volume) {
	logger.Debug("Entry of FromProviderToLibVolume method...")
	defer logger.Debug("Exit from FromProviderToLibVolume method...")

	if vpcVolume == nil {
		logger.Info("Volume details are empty")
		return
	}

	if vpcVolume.Zone == nil {
		logger.Info("Volume zone is empty")
		return
	}

	logger.Debug("Volume details of VPC client", zap.Reflect("models.Volume", vpcVolume))

	volumeCap := int(vpcVolume.Capacity)
	iops := strconv.Itoa(int(vpcVolume.Iops))
	var createdDate time.Time
	if vpcVolume.CreatedAt != nil {
		createdDate = *vpcVolume.CreatedAt
	}

	libVolume = &provider.Volume{
		VolumeID:     vpcVolume.ID,
		Provider:     VPC,
		Capacity:     &volumeCap,
		Iops:         &iops,
		VolumeType:   VolumeType,
		CreationTime: createdDate,
	}
	if vpcVolume.Zone != nil {
		libVolume.Az = vpcVolume.Zone.Name
	}
	if vpcVolume.SourceSnapshot != nil {
		libVolume.SnapshotID = vpcVolume.SourceSnapshot.ID
	}
	libVolume.CRN = vpcVolume.CRN
	libVolume.Tags = vpcVolume.UserTags
	return
}

// FromProviderToLibSnapshot converting vpc provider snapshot type to generic lib snapshot type
func FromProviderToLibSnapshot(vpcSnapshot *models.Snapshot, logger *zap.Logger) (libSnapshot *provider.Snapshot) {
	logger.Debug("Entry of FromProviderToLibSnapshot method...")
	defer logger.Debug("Exit from FromProviderToLibSnapshot method...")

	if vpcSnapshot == nil {
		logger.Info("Snapshot details are empty")
		return
	}

	logger.Debug("Snapshot details of VPC client", zap.Reflect("models.Snapshot", vpcSnapshot))

	var createdTime time.Time
	if vpcSnapshot.CreatedAt != nil {
		createdTime = *vpcSnapshot.CreatedAt
	}
	libSnapshot = &provider.Snapshot{
		VolumeID:             vpcSnapshot.SourceVolume.ID,
		SnapshotID:           vpcSnapshot.ID,
		SnapshotCreationTime: createdTime,
		SnapshotSize:         GiBToBytes(vpcSnapshot.MinimumCapacity),
		VPC:                  provider.VPC{Href: vpcSnapshot.Href},
	}
	if vpcSnapshot.LifecycleState == snapshotReadyState {
		libSnapshot.ReadyToUse = true
	} else {
		libSnapshot.ReadyToUse = false
	}
	return
}

// IsValidVolumeIDFormat validating(gc has 5 parts and NG has 6 parts)
func IsValidVolumeIDFormat(volID string) bool {
	parts := strings.Split(volID, "-")
	return len(parts) >= volumeIDPartsCount
}

// SetRetryParameters sets the retry logic parameters
func SetRetryParameters(maxAttempts int, maxGap int) {
	if maxAttempts > 0 {
		maxRetryAttempt = maxAttempts
	}

	if maxGap > 0 {
		maxRetryGap = maxGap
	}
}

func roundUpSize(volumeSizeBytes int64, allocationUnitBytes int64) int64 {
	return (volumeSizeBytes + allocationUnitBytes - 1) / allocationUnitBytes
}

// GiBToBytes converts GiB to Bytes
func GiBToBytes(volumeSizeGiB int64) int64 {
	return volumeSizeGiB * GiB
}

// isValidServiceSession check if Service Session is valid
func isValidServiceSession(vpcs *VPCSession) (err error) {
	//If VPC session contains valid SessionError then there is some session related issue
	if vpcs.SessionError != nil {
		vpcs.Logger.Warn("Provider session is not valid")
		return userError.GetUserError(string(userError.InvalidServiceSession), vpcs.SessionError)
	}

	return nil
}

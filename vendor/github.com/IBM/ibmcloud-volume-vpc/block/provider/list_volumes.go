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
	"fmt"
	"strings"
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/metrics"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	userError "github.com/IBM/ibmcloud-volume-vpc/common/messages"
	"github.com/IBM/ibmcloud-volume-vpc/common/vpcclient/models"
	"go.uber.org/zap"
)

const (
	maxLimit                 = 100
	startVolumeIDNotFoundMsg = "start parameter is not valid"
)

// ListVolumes list all volumes
func (vpcs *VPCSession) ListVolumes(limit int, start string, tags map[string]string) (*provider.VolumeList, error) {
	vpcs.Logger.Info("Entry ListVolumes", zap.Reflect("start", start), zap.Reflect("filters", tags))
	defer vpcs.Logger.Info("Exit ListVolumes", zap.Reflect("start", start), zap.Reflect("filters", tags))
	defer metrics.UpdateDurationFromStart(vpcs.Logger, "ListVolumes", time.Now())

	if limit < 0 {
		return nil, userError.GetUserError("InvalidListVolumesLimit", nil, limit)
	}

	if limit > maxLimit {
		vpcs.Logger.Warn(fmt.Sprintf("listVolumes requested max entries of %v, supports values <= %v so defaulting value back to %v", limit, maxLimit, maxLimit))
		limit = maxLimit
	}

	filters := &models.ListVolumeFilters{
		// Tag:          tags["tag"],
		ResourceGroupID: tags["resource_group.id"],
		ZoneName:        tags["zone.name"],
		VolumeName:      tags["name"],
	}

	vpcs.Logger.Info("Getting volumes list from VPC provider...", zap.Reflect("start", start), zap.Reflect("filters", filters))

	var volumes *models.VolumeList
	var err error
	err = retry(vpcs.Logger, func() error {
		volumes, err = vpcs.Apiclient.VolumeService().ListVolumes(limit, start, filters, vpcs.Logger)
		return err
	})

	if err != nil {
		if strings.Contains(err.Error(), startVolumeIDNotFoundMsg) {
			return nil, userError.GetUserError("StartVolumeIDNotFound", err, start)
		}
		return nil, userError.GetUserError("ListVolumesFailed", err)
	}

	vpcs.Logger.Info("Successfully retrieved volumes list from VPC backend", zap.Reflect("VolumesList", volumes))

	var respVolumesList = &provider.VolumeList{}
	if volumes != nil {
		if volumes.Next != nil {
			var next string
			// "Next":{"href":"https://eu-gb.iaas.cloud.ibm.com/v1/volumes?start=3e898aa7-ac71-4323-952d-a8d741c65a68\u0026limit=1\u0026zone.name=eu-gb-1"}
			if strings.Contains(volumes.Next.Href, "start=") {
				next = strings.Split(strings.Split(volumes.Next.Href, "start=")[1], "\u0026")[0]
			} else {
				vpcs.Logger.Warn("Volumes.Next.Href is not in expected format", zap.Reflect("volumes.Next.Href", volumes.Next.Href))
			}
			respVolumesList.Next = next
		}

		volumeslist := volumes.Volumes
		if len(volumeslist) > 0 {
			for _, volItem := range volumeslist {
				volumeResponse := FromProviderToLibVolume(volItem, vpcs.Logger)
				respVolumesList.Volumes = append(respVolumesList.Volumes, volumeResponse)
			}
		}
	}
	return respVolumesList, err
}

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

// Package watcher ...
package watcher

import (
	"flag"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"

	cloudprovider "github.com/IBM/ibm-csi-common/pkg/ibmcloudprovider"
	"github.com/IBM/ibm-csi-common/pkg/utils"
	"github.com/IBM/ibmcloud-volume-interface/config"
	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
)

// PVWatcher to watch  pv creation and add taggs
type PVWatcher struct {
	logger          *zap.Logger
	kclient         kubernetes.Interface
	config          *config.Config
	provisionerName string
	recorder        record.EventRecorder
	cloudProvider   cloudprovider.CloudProviderInterface
}

const (
	//IbmCloudGtAPIEndpoint ...
	IbmCloudGtAPIEndpoint = "IBMCLOUD_GT_API_ENDPOINT"
	//ReclaimPolicyTag ...
	ReclaimPolicyTag = "reclaimpolicy:"
	//NameSpaceTag ...
	NameSpaceTag = "namespace:"
	//StorageClassTag ...
	StorageClassTag = "storageclass:"
	//PVCNameTag ...
	PVCNameTag = "pvc:"
	//PVNameTag ...
	PVNameTag = "pv:"
	//VolumeCRN ...
	VolumeCRN = "volumeCRN"
	//ProvisionerTag ...
	ProvisionerTag = "provisioner:"

	//VolumeStatus ...
	VolumeStatus = "status"
	//VolumeStatusCreated ...
	VolumeStatusCreated = "created"
	//VolumeStatusDeleted ...
	VolumeStatusDeleted = "deleted"
	//VolumeUpdateEventReason ...
	VolumeUpdateEventReason = "VolumeMetaDataSaved"
	//VolumeUpdateEventSuccess ...
	VolumeUpdateEventSuccess = "Success"
)

// VolumeTypeMap ...
var VolumeTypeMap = map[string]string{}

var master = flag.String(
	"master",
	"",
	"Master URL to build a client config from. Either this or kubeconfig needs to be set if the provisioner is being run out of cluster.",
)
var kubeconfig = flag.String(
	"kubeconfig",
	"",
	"Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.",
)

// New creates the Watcher instance
func New(logger *zap.Logger, provisionerName string, volumeType string, cloudProvider cloudprovider.CloudProviderInterface) *PVWatcher {
	var restConfig *rest.Config
	var err error
	// Register provider
	VolumeTypeMap[provisionerName] = volumeType

	restConfig, err = clientcmd.BuildConfigFromFlags(*master, *kubeconfig)
	if err != nil {
		logger.Fatal("Failed to create config:", zap.Error(err))
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		logger.Fatal("Failed to create client:", zap.Error(err))
	}
	iksPodName := os.Getenv("POD_NAME")

	broadcaster := record.NewBroadcaster()
	broadcaster.StartLogging(glog.Infof)
	eventInterface := clientset.CoreV1().Events("")
	broadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: eventInterface})
	pvw := &PVWatcher{
		logger:          logger,
		config:          cloudProvider.GetConfig(),
		provisionerName: provisionerName,
		kclient:         clientset,
		cloudProvider:   cloudProvider,
		recorder:        broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: iksPodName}),
	}
	return pvw
}

// Start start pv watcher
func (pvw *PVWatcher) Start() {
	watchlist := cache.NewListWatchFromClient(pvw.kclient.CoreV1().RESTClient(), "persistentvolumes", "", fields.Everything())
	_, controller := cache.NewInformer(watchlist, &v1.PersistentVolume{}, time.Second*0,
		cache.FilteringResourceEventHandler{
			Handler: cache.ResourceEventHandlerFuncs{
				UpdateFunc: pvw.updateVolume,
			},
			FilterFunc: pvw.filter,
		},
	)
	pvw.logger.Info("PVWatcher starting")
	stopch := wait.NeverStop
	go controller.Run(stopch)
	pvw.logger.Info("PVWatcher started")
	<-stopch
}

func (pvw *PVWatcher) updateVolume(oldobj, obj interface{}) {
	// Run as non-blocking thread to allow parallel processing of volumes
	go func() {
		ctxLogger, requestID := utils.GetContextLogger(context.Background(), false)
		// panic-recovery function that avoid watcher thread to stop because of unexexpected error
		defer func() {
			if r := recover(); r != nil {
				ctxLogger.Error("Recovered from panic in pvwatcher", zap.Stack("stack"), zap.String("requestID", requestID))
			}
		}()

		ctxLogger.Info("Entry updateVolume()", zap.Reflect("obj", obj))
		pv, _ := obj.(*v1.PersistentVolume)
		session, err := pvw.cloudProvider.GetProviderSession(context.Background(), ctxLogger)
		if session != nil {
			volume := pvw.getVolume(pv, ctxLogger)
			ctxLogger.Info("volume to update ", zap.Reflect("volume", volume))
			err := session.UpdateVolume(volume)
			if err != nil {
				ctxLogger.Warn("Unable to update the volume", zap.Error(err))
				pvw.recorder.Event(pv, v1.EventTypeWarning, VolumeUpdateEventReason, err.Error())
			} else {
				pvw.recorder.Event(pv, v1.EventTypeNormal, VolumeUpdateEventReason, VolumeUpdateEventSuccess)
				ctxLogger.Warn("Volume Metadata saved successfully")
			}
		}
		ctxLogger.Info("Exit updateVolume()", zap.Error(err))
	}()
}

func (pvw *PVWatcher) getTags(pv *v1.PersistentVolume, ctxLogger *zap.Logger) (string, []string) {
	ctxLogger.Debug("Entry getTags()", zap.Reflect("pv", pv))
	volAttributes := pv.Spec.CSI.VolumeAttributes
	// Get user tag list
	tagstr := strings.TrimSpace(volAttributes["tags"])
	var tags []string
	if len(tagstr) > 0 {
		tags = strings.Split(tagstr, ",")
	}
	// append default tags to users tag list
	tags = append(tags, utils.ClusterIDLabel+":"+volAttributes[utils.ClusterIDLabel])
	tags = append(tags, ReclaimPolicyTag+string(pv.Spec.PersistentVolumeReclaimPolicy))
	tags = append(tags, StorageClassTag+pv.Spec.StorageClassName)
	tags = append(tags, NameSpaceTag+pv.Spec.ClaimRef.Namespace)
	tags = append(tags, PVCNameTag+pv.Spec.ClaimRef.Name)
	tags = append(tags, PVNameTag+pv.ObjectMeta.Name)
	tags = append(tags, ProvisionerTag+pvw.provisionerName)
	ctxLogger.Debug("Exit getTags()", zap.String("VolumeCRN", volAttributes[VolumeCRN]), zap.Reflect("tags", tags))
	return volAttributes[VolumeCRN], tags
}

func (pvw *PVWatcher) getVolume(pv *v1.PersistentVolume, ctxLogger *zap.Logger) provider.Volume {
	ctxLogger.Debug("Entry getVolume()", zap.Reflect("pv", pv))
	crn, tags := pvw.getTags(pv, ctxLogger)
	volume := provider.Volume{
		VolumeID:   pv.Spec.CSI.VolumeHandle,
		Provider:   provider.VolumeProvider(pvw.config.VPC.VPCBlockProviderType),
		VolumeType: provider.VolumeType(VolumeTypeMap[pv.Spec.CSI.Driver]),
	}
	volume.CRN = crn
	clusterID := pv.Spec.CSI.VolumeAttributes[utils.ClusterIDLabel]
	volume.Attributes = map[string]string{strings.ToLower(utils.ClusterIDLabel): clusterID}
	if pv.Status.Phase == v1.VolumeReleased {
		// Set only status in case of delete operation
		volume.Attributes[VolumeStatus] = VolumeStatusDeleted
	} else {
		volume.Tags = tags
		//Get Capacity and convert to GiB
		capacity := pv.Spec.Capacity[v1.ResourceStorage]
		capacityGiB := utils.BytesToGiB(capacity.Value())
		volume.Capacity = &capacityGiB
		iops := pv.Spec.CSI.VolumeAttributes[utils.IOPSLabel]
		volume.Iops = &iops
		volume.Attributes[VolumeStatus] = VolumeStatusCreated
	}
	ctxLogger.Debug("Exit getVolume()", zap.Reflect("volume", volume))
	return volume
}

func (pvw *PVWatcher) filter(obj interface{}) bool {
	pvw.logger.Debug("Entry filter()", zap.Reflect("obj", obj))
	pv, _ := obj.(*v1.PersistentVolume)
	var provisoinerMatch = false
	if pv != nil && pv.Spec.CSI != nil {
		provisoinerMatch = pv.Spec.CSI.Driver == pvw.provisionerName
	}
	pvw.logger.Debug("Exit filter()", zap.Bool("provisoinerMatch", provisoinerMatch))
	return provisoinerMatch
}

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

// Package main ...
package main

import (
	"flag"
	"strings"

	"math/rand"
	"net/http"
	"os"
	"time"

	libMetrics "github.com/IBM/ibmcloud-volume-interface/lib/metrics"
	k8sUtils "github.com/IBM/secret-utils-lib/pkg/k8s_utils"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	cloudProvider "github.com/IBM/ibm-csi-common/pkg/ibmcloudprovider"
	nodeInfoManager "github.com/IBM/ibm-csi-common/pkg/metadata"
	"github.com/IBM/ibm-csi-common/pkg/metrics"
	mountManager "github.com/IBM/ibm-csi-common/pkg/mountmanager"
	"github.com/IBM/ibm-csi-common/pkg/utils"
	"github.com/IBM/ibm-csi-common/pkg/watcher"
	csiConfig "github.com/kubernetes-sigs/ibm-vpc-block-csi-driver/config"
	driver "github.com/kubernetes-sigs/ibm-vpc-block-csi-driver/pkg/ibmcsidriver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	_ = flag.Set("logtostderr", "true") // #nosec G104: Attempt to set flags for logging to stderr only on best-effort basis.Error cannot be usefully handled.
	logger = setUpLogger()
	defer logger.Sync() //nolint: errcheck
}

var (
	endpoint             = flag.String("endpoint", "unix:/tmp/csi.sock", "CSI endpoint")
	metricsAddress       = flag.String("metrics-address", "0.0.0.0:9080", "Metrics address")
	extraVolumeLabelsStr = flag.String("extra-labels", "", "Extra labels to tag all volumes created by driver. It is a comma separated list of key value pairs like '<key1>:<value1>,<key2>:<value2>'.")
	vendorVersion        string
	logger               *zap.Logger
)

const (
	vpc = "vpc"
)

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	handle(logger)
	os.Exit(0)
}

func setUpLogger() *zap.Logger {
	// Prepare a new logger
	atom := zap.NewAtomicLevel()
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	), zap.AddCaller()).With(zap.String("name", csiConfig.CSIDriverGithubName)).With(zap.String("CSIDriverName", csiConfig.CSIDriverLogName))

	atom.SetLevel(zap.InfoLevel)
	return logger
}

func handle(logger *zap.Logger) {
	if vendorVersion == "" {
		logger.Fatal("IBM CSI driver vendorVersion must be set at compile time")
	}
	logger.Info("IBM CSI driver version", zap.Reflect("DriverVersion", vendorVersion))
	logger.Info("Controller Mutex Lock enabled", zap.Bool("LockEnabled", *utils.LockEnabled))
	// Setup Cloud Provider
	k8sClient, err := k8sUtils.Getk8sClientSet()
	if err != nil {
		logger.Fatal("Failed to instantiate IKS-Storage provider", zap.Error(err))
	}

	ibmcloudProvider, err := cloudProvider.NewIBMCloudStorageProvider(*extraVolumeLabelsStr, &k8sClient, logger)
	if err != nil {
		logger.Fatal("Failed to instantiate IKS-Storage provider", zap.Error(err))
	}

	// Setup CSI Driver
	ibmCSIDriver := driver.GetIBMCSIDriver()

	// Get new instance for the Mount Manager
	mounter := mountManager.NewNodeMounter()

	nodeName := os.Getenv("KUBE_NODE_NAME")

	nodeInfo := nodeInfoManager.NodeInfoManager{
		NodeName: nodeName,
	}

	statUtil := &(driver.VolumeStatUtils{})

	err = ibmCSIDriver.SetupIBMCSIDriver(ibmcloudProvider, mounter, statUtil, nil, &nodeInfo, logger, csiConfig.CSIDriverName, vendorVersion)
	if err != nil {
		logger.Fatal("Failed to initialize driver...", zap.Error(err))
	}

	logger.Info("Successfully initialized driver...")
	serveMetrics()
	// Start PV watcher if its controller POD
	if strings.Contains(os.Getenv("POD_NAME"), "csi-controller") && strings.Contains(os.Getenv("IKS_ENABLED"), "True") {
		pvwatcher := watcher.New(logger, csiConfig.CSIDriverName, csiConfig.CSIProviderVolumeType, ibmcloudProvider)
		go pvwatcher.Start()
	}

	ibmCSIDriver.Run(*endpoint)
}

func serveMetrics() {
	logger.Info("Starting metrics endpoint")
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		//http.Handle("/health-check", healthCheck)
		err := http.ListenAndServe(*metricsAddress, nil)
		logger.Error("Failed to start metrics service:", zap.Error(err))
	}()
	metrics.RegisterAll(csiConfig.CSIDriverGithubName)
	libMetrics.RegisterAll()
}

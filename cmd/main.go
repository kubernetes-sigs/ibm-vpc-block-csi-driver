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

//Package main ...
package main

import (
	"flag"
	"strings"

	"github.com/IBM/ibmcloud-volume-interface/config"
	libMetrics "github.com/IBM/ibmcloud-volume-interface/lib/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	cloudProvider "github.com/IBM/ibm-csi-common/pkg/ibmcloudprovider"
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
	_ = flag.Set("logtostderr", "true")
	logger = setUpLogger()
	defer logger.Sync() //nolint: errcheck
}

var (
	endpoint       = flag.String("endpoint", "unix:/tmp/csi.sock", "CSI endpoint")
	metricsAddress = flag.String("metrics-address", "0.0.0.0:9080", "Metrics address")
	vendorVersion  string
	logger         *zap.Logger
)

const (
	configFileName = "slclient.toml"
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
	configPath := filepath.Join(config.GetConfPathDir(), configFileName)
	ibmcloudProvider, err := cloudProvider.NewIBMCloudStorageProvider(configPath, logger)
	if err != nil {
		logger.Fatal("Failed to instantiate IKS-Storage provider", zap.Error(err))
	}

	// Setup CSI Driver
	ibmCSIDriver := driver.GetIBMCSIDriver()

	// Get new instance for the Mount Manager
	mounter := mountManager.NewNodeMounter()

	statUtil := &(driver.VolumeStatUtils{})

	err = ibmCSIDriver.SetupIBMCSIDriver(ibmcloudProvider, mounter, statUtil, nil, logger, csiConfig.CSIDriverName, vendorVersion)
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

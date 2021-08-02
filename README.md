# ibm-vpc-block-csi-driver

[![Build Status](https://app.travis-ci.com/kubernetes-sigs/ibm-vpc-block-csi-driver.svg?branch=master)](https://app.travis-ci.com/kubernetes-sigs/ibm-vpc-block-csi-driver)
[![code-coverage](https://github.com/kubernetes-sigs/ibm-vpc-block-csi-driver/blob/gh-pages/coverage/testbadge/badge.svg)](https://github.com/kubernetes-sigs/ibm-vpc-block-csi-driver/blob/gh-pages/coverage/testbadge/cover.html)

ibm-vpc-block-csi-driver is a CSI plugin for creating and mounting VPC block storage on IBM VPC infrastructure based openshift or kubernetes cluster

# Supported orchestration platforms

The following table details orchestration platforms suitable for deployment of the IBM VPC block storage CSI driver.

|Orchestration platform|Version|Architecture|
|----------------------|-------|------------|
|Kubernetes|1.21|x86|
|Kubernetes|1.20|x86|
|Kubernetes|1.19|x86|
|Red Hat® OpenShift®|4.7|x86|
|Red Hat OpenShift|4.6|x86|

# Prerequisites

To use the Block Storage for IBM VPC driver, complete the following tasks:

1. Create a cluster based on IBM VPC infrastructure

# Build the driver

For building the driver `docker` and `GO` should be installed

1. On your local machine, install [`docker`](https://docs.docker.com/install/) and [`Go`](https://golang.org/doc/install).
2. GO version should be >=1.16
3. Set the [`GOPATH` environment variable](https://github.com/golang/go/wiki/SettingGOPATH).
4. Build the driver image

   ## Clone the repo or your forked repo

   ```
   $ mkdir -p $GOPATH/src/github.com/IBM
   $ cd $GOPATH/src/github.com/IBM/
   $ git clone https://github.com/kubernetes-sigs/ibm-vpc-block-csi-driver.git
   $ cd ibm-vpc-block-csi-driver
   ```
   ## Build project and runs testcases

   ```
   $ make
   ```
   ## Build container image for the driver

   ```
   $ make buildimage
   ```

   ## Push image to registry

   Image should be pushed to any registry from which the worker nodes have access to pull

   You can push the driver image to [docker.io](https://hub.docker.com/)  registry or [IBM public registry](https://cloud.ibm.com/docs/Registry?topic=Registry-registry_overview#registry_regions_local) under your namespace.

   For pushing to IBM registry:

   Create an image pull secret in your cluster

   1. Review and retrieve the following values for your image pull secret.

      `<docker-username>` - Enter the string: `iamapikey`.

      `<docker-password>` - Enter your IAM API key. For more information about IAM API keys, see [ Understanding API keys ](https://cloud.ibm.com/docs/account?topic=account-manapikey).

      `<docker-email>` - Enter the string: iamapikey.

   2. Run the following command to create the image pull secret in your cluster. Note that your secret must be named icr-io-secret


      ```

       kubectl create secret docker-registry icr-io-secret --docker-server=icr.io --docker-username=iamapikey --docker-password=-<iam-api-key> --docker-email=iamapikey -n kube-system

      ```


# Deploy CSI driver on your cluster

- Install `kustomize` tool. The instructions are available [here](https://kubectl.docs.kubernetes.io/installation/kustomize/)
- Export cluster config
- Deploy CSI plugin on your cluster
  - You can use any overlays available under `deploy/kubernetes/driver/kubernetes/overlays/` and edit the image tag
  - Example using `stage` overlay to update the image tag
     - Change `iks-vpc-block-driver` image name in `deploy/kubernetes/driver/kubernetes/overlays/stage/controller-server-images.yaml`
     - Change `iks-vpc-block-driver` image name in `deploy/kubernetes/driver/kubernetes/overlays/stage/node-server-images.yaml`
  - Deploy plugin
    - `sh deploy/kubernetes/driver/kubernetes/deploy-vpc-block-driver.sh stage`

## Testing

- Create storage classes
  - `ls deploy/kubernetes/storageclass/ | xargs -I classfile kubectl apply -f deploy/kubernetes/storageclass/classfile`
- Create PVC
  - `kubectl create -f examples/kubernetes/validPVC.yaml`
- .Create POD with volume
  - `kubectl create -f examples/kubernetes/validPOD.yaml`

# E2E Tests

  Please refer [ this](https://github.com/IBM/ibm-csi-common/tree/master/tests/e2e) repository for e2e tests.

# How to contribute

If you have any questions or issues you can create a new issue [ here ](https://github.com/kubernetes-sigs/ibm-vpc-block-csi-driver/issues/new).

Pull requests are very welcome! Make sure your patches are well tested. Ideally create a topic branch for every separate change you make. For example:

1. Fork the repo

2. Create your feature branch (git checkout -b my-new-feature)

3. Commit your changes (git commit -am 'Added some feature')

4. Push to the branch (git push origin my-new-feature)

5. Create new Pull Request

6. Add the test results in the PR


# Licensing

Copyright 2020 IBM Corp.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

# ibm-vpc-block-csi-driver

[![Build Status](https://prow.k8s.io/badge.svg?jobs=pull-ibm-vpc-block-csi-driver-build)](https://prow.k8s.io)
[![Coverage Status](https://coveralls.io/repos/github/kubernetes-sigs/ibm-vpc-block-csi-driver/badge.svg?branch=master)](https://coveralls.io/github/kubernetes-sigs/ibm-vpc-block-csi-driver?branch=master)


[`IBM VPC Block`](https://cloud.ibm.com/docs/openshift?topic=openshift-vpc-block) Container Storage Interface (CSI) Driver provides a [`CSI`](https://github.com/container-storage-interface/spec/blob/master/spec.md) interface used by Container Orchestrators to manage the lifecycle of IBM VPC Block Data volumes.

# Supported orchestration platforms

The following are the supported orchestration platforms suitable for deployment for IBM VPC Block CSI Driver.

|Orchestration platform|Version|Architecture|
|----------------------|-------|------------|
|Red Hat® OpenShift®|4.7|x86|
|Red Hat® OpenShift®|4.8|x86|
|Red Hat® OpenShift®|4.9|x86|
|Kubernetes| 1.19|x86|
|Kubernetes| 1.20|x86|
|Kubernetes| 1.21|x86|

# Prerequisites

Following are the prerequisites to use the IBM VPC Block CSI Driver:

1. User should have either Red Hat® OpenShift® or kubernetes cluster on IBM VPC Gen 2 infrastructure.
2. Should have compatible orchestration platform.
3. Install and configure `ibmcloud is` CLI or get the required worker/node details by using [`IBM Cloud Console`](https://cloud.ibm.com)
4. Cluster's worker node should have following labels, if not please apply labels before deploying IBM VPC Block CSI Driver.
```
"ibm-cloud.kubernetes.io/worker-id"
"failure-domain.beta.kubernetes.io/region"
"failure-domain.beta.kubernetes.io/zone"
"topology.kubernetes.io/region"
"topology.kubernetes.io/zone"
```

## Apply worker labels

Please use [`apply-required-setup.sh`](https://github.com/kubernetes-sigs/ibm-vpc-block-csi-driver/blob/master/scripts/apply-required-setup.sh) script for all the nodes in the cluster which will need couple of inputs like 

`instanceID`:  That you can get from `ibmcloud is ins` 

`node-name`: this is as per node name in the kubernetes node check by using `kubectl get nodes`

`region-of-instanceID`:  region of the instanceID, this you can get the by using `ibmcloud is in <instanceID>`

`zone-of-instanceID`: Zone of the instanceID, this you can get the by using `ibmcloud is in <instanceID>`

Example :- ./apply-required-setup.sh <node-name> <instanceID> <region-of-instanceID> <zone-of-instanceID>

# Build the driver

For building the driver `docker` and `GO` should be installed on the system

1. On your local machine, install [`docker`](https://docs.docker.com/install/) and [`Go`](https://golang.org/doc/install).
2. GO version should be >=1.16
3. Set the [`GOPATH` environment variable](https://github.com/golang/go/wiki/SettingGOPATH).
4. Build the driver image

   ## Clone the repo or your forked repo

   ```
   $ mkdir -p $GOPATH/src/github.com/kubernetes-sigs
   $ cd $GOPATH/src/github.com/kubernetes-sigs/
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

   Image should be pushed to any registry from which cluster worker nodes have access to pull

   You can push the driver image to [docker.io](https://hub.docker.com/)  registry or [IBM public registry](https://cloud.ibm.com/docs/Registry?topic=Registry-registry_overview#registry_regions_local) under your namespace.

   For pushing to IBM registry:

   Create an image pull secret in your cluster

   1. ibmcloud login to the target region

   2. Run - ibmcloud cr region-set global

   3. Run - ibmcloud cr login

   4. Make sure kubectl is configured to use the cluster

   5. Review and retrieve the following values for your image pull secret.

      `<docker-username>` - Enter the string: `iamapikey`.

      `<docker-password>` - Enter your IAM API key. For more information about IAM API keys, see [ Understanding API keys ](https://cloud.ibm.com/docs/account?topic=account-manapikey).

      `<docker-email>` - Enter the string: iamapikey.

   6. Run the following command to create the image pull secret in your cluster. Note that your secret must be named icr-io-secret


      ```

       kubectl create secret docker-registry icr-io-secret --docker-server=icr.io --docker-username=iamapikey --docker-password=-<iam-api-key> --docker-email=iamapikey -n kube-system

      ```

# Deploy CSI driver on your cluster
- Edit [slclient_Gen2.toml](https://github.com/kubernetes-sigs/ibm-vpc-block-csi-driver/blob/master/deploy/kubernetes/driver/kubernetes/slclient_Gen2.toml) for your cluster.

IBM VPC endpoints which supports Gen2 is documented [here](https://cloud.ibm.com/docs/vpc?topic=vpc-service-endpoints-for-vpc)
- Install `kustomize` tool. The instructions are available [here](https://kubectl.docs.kubernetes.io/installation/kustomize/)
- Export cluster config i.e configuring kubectl command
- Deploy IBM VPC Block CSI Driver on your cluster
  - You can use any overlays available under `deploy/kubernetes/driver/kubernetes/overlays/` and edit the image tag if you want to use your own build image from this source code, although default overlays are already using released IBM VPC Block CSI Driver image 

  - `gcr.io/k8s-staging-cloud-provider-ibm/ibm-vpc-block-csi-driver:master` image is always the latest image build using `master` branch code.	
  - Example using `stage` overlay to update the image tag
     - Change `iks-vpc-block-driver` image name in `deploy/kubernetes/driver/kubernetes/overlays/stage/controller-server-images.yaml`
     - Change `iks-vpc-block-driver` image name in `deploy/kubernetes/driver/kubernetes/overlays/stage/node-server-images.yaml`
  - Deploy plugin
    - `bash deploy/kubernetes/driver/kubernetes/deploy-vpc-block-driver.sh stage`

## Testing

- Create storage classes
  - `ls deploy/kubernetes/storageclass/ | xargs -I classfile kubectl apply -f deploy/kubernetes/storageclass/classfile`
- Create PVC
  - `kubectl create -f examples/kubernetes/validPVC.yaml`
- Create POD with volume
  - `kubectl create -f examples/kubernetes/validPOD.yaml`

# Delete CSI driver from your cluster

  - Delete plugin
    - `bash deploy/kubernetes/driver/kubernetes/delete-vpc-csi-driver.sh stage`

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

## Vendor changes

For any changes to `go.mod` or `go.sum`, be sure to run `go mod vendor` to update dependencies in the `vendor/` directory. You can verify that the vendor directory is up-to-date before filing a PR by running `hack/verify-vendor.sh`.

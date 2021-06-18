# ibm-vpc-block-csi-driver
ibm-vpc-block-csi-driver is a CSI plugin for creating and mounting VPC block storage on IBM VPC infrastructure based openshift or kubernetes cluster

# Supported orchestration platforms

The following table details orchestration platforms suitable for deployment of the IBM® block storage CSI driver.

|Orchestration platform|Version|Architecture|
|----------------------|-------|------------|
|Kubernetes|1.20|x86|
|Kubernetes|1.21|x86|
|Kubernetes|1.19|x86|
|Red Hat® OpenShift®|4.7|x86|
|Red Hat OpenShift|4.6|x86|

# Build the driver

For building the driver `docker` and `GO` should be installed

1. On your local machine, install [`docker`](https://docs.docker.com/install/) and [`Go`](https://golang.org/doc/install).
2. Set the [`GOPATH` environment variable](https://github.com/golang/go/wiki/SettingGOPATH).
3. Build the driver image

   clone the repo or your forked repo
   ```
   $ mkdir -p $GOPATH/src/github.com/IBM
   $ mkdir -p $GOPATH/bin
   $ cd $GOPATH/src/github.com/IBM/
   $ git clone https://github.com/IBM/ibm-vpc-block-csi-driver.git
   $ cd ibm-vpc-block-csi-driver
   ```
   build project and runs testcases
   ```
   $ make
   ```
   build container image for the driver
   ```
   $ make buildimage
   ```

   Push image to registry

   Image should be pushed to any registry from which the worker nodes have access to pull

   1. You can push the driver image to [docker.io](https://hub.docker.com/)  registry or [IBM public registry](https://cloud.ibm.com/docs/Registry?topic=Registry-registry_overview#registry_regions_local) under your namespace.

# Deploy CSI driver on your cluster

- Export cluster config
- Deploy CSI plugin on your cluster
  - Update the image tag
     - Change `iks-vpc-block-driver` image name in `deploy/kubernetes/driver/kubernetes/overlays/stage/controller-server-images.yaml`
     - Change `iks-vpc-block-driver` image name in `deploy/kubernetes/driver/kubernetes/overlays/stage/node-server-images.yaml`
  - Install `kustomize` tool. The instructions are available [here](https://kubectl.docs.kubernetes.io/installation/kustomize/)
  - Deploy plugin
    - `sh deploy/kubernetes/driver/kubernetes/deploy-vpc-block-driver.sh stage`

## Testing
- Create storage classes
  - `ls deploy/kubernetes/storageclass/ | xargs -I classfile kubectl apply -f deploy/kubernetes/storageclass/classfile`
- Create PVC
  - `kubectl create -f examples/kubernetes/validPVC.yaml`
- .Create POD with volume
  - `kubectl create -f examples/kubernetes/validPOD.yaml`

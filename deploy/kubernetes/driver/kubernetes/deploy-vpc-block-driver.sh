#!/bin/bash

# Installing VPC block volume CSI Driver to the IKS cluster

set -o nounset
set -o errexit
#set -x

if [ $# != 2 ]; then
  echo "This will install 'stable' version of vpc csi driveri for gen2 cluster configuration!"
else
  readonly IKS_VPC_BLOCK_DRIVER_VERSION=$1
  readonly CLUSTER_TYPE=$2
  echo "This will install '${IKS_VPC_BLOCK_DRIVER_VERSION}' version of vpc csi driver for $CLUSTER_TYPE ! 
fi

readonly VERSION="${IKS_VPC_BLOCK_DRIVER_VERSION:-stable}"
readonly GENVERSION="{$CLUSTER_TYPE:-gen2}"
readonly PKG_DIR="${GOPATH}/src/github.com/kubernetes-sigs/ibm-vpc-block-csi-driver"
#source "${PKG_DIR}/deploy/kubernetes/driver/common.sh"

if [ $GENVERSION == "gen1" ]; then
	encodeVal=$(base64 -w 0 ${PKG_DIR}/deploy/kubernetes/driver/kubernetes/slclient_Gen1.toml)
else
	encodeVal=$(base64 -w 0 ${PKG_DIR}/deploy/kubernetes/driver/kubernetes/slclient_Gen2.toml)
fi

sed -i "s/REPLACE_ME/$encodeVal/g" ${PKG_DIR}/deploy/kubernetes/driver/kubernetes/overlays/${VERSION}/storage-secret-store.yaml

#ensure_kustomize

#${KUSTOMIZE_PATH}
kustomize build ${PKG_DIR}/deploy/kubernetes/driver/kubernetes/overlays/${VERSION} | kubectl apply -f -


#!/bin/bash

# Installing VPC block volume CSI Driver to the IKS cluster

set -o nounset
set -o errexit
# set -x

if [ $# != 1 ]; then
  echo "This will install 'stable' version of vpc csi driveri for gen2 cluster configuration!"
else
  readonly IKS_VPC_BLOCK_DRIVER_VERSION=$1
  echo "This will install '${IKS_VPC_BLOCK_DRIVER_VERSION}' version of vpc csi driver!" 
fi

readonly VERSION="${IKS_VPC_BLOCK_DRIVER_VERSION:-stable}"
readonly PKG_DIR="${GOPATH}/src/github.com/kubernetes-sigs/ibm-vpc-block-csi-driver"

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
	echo $OSTYPE	
	encodeVal=$(base64 -w 0 ${PKG_DIR}/deploy/kubernetes/driver/kubernetes/slclient_Gen2.toml)
        sed -i "s/REPLACE_ME/$encodeVal/g" ${PKG_DIR}/deploy/kubernetes/driver/kubernetes/manifests/storage-secret-store.yaml

elif [[ "$OSTYPE" == "darwin"* ]]; then

	encodeVal=$(base64 ${PKG_DIR}/deploy/kubernetes/driver/kubernetes/slclient_Gen2.toml)
        sed -i '.bak' "s/REPLACE_ME/$encodeVal/g" ${PKG_DIR}/deploy/kubernetes/driver/kubernetes/manifests/storage-secret-store.yaml
fi
# ensure_kustomize

kustomize build ${PKG_DIR}/deploy/kubernetes/driver/kubernetes/overlays/${VERSION} | kubectl apply -f -

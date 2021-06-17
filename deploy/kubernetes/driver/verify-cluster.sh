#!/bin/bash

set -o nounset
set -o errexit

IKS_CLUSTER_NAME=$1

KUBE_CONF=`ibmcloud ks cluster-config $IKS_CLUSTER_NAME | grep KUBECONFIG | awk '{print $2}'`
export $KUBE_CONF

kubectl get secrets -n kube-system | grep storage-secret-store  > /dev/null 2>&1
if [ $? != 0 ]; then
	echo "VPC Block volume driver can't be installed as you don't have storage secret in your cluster"
	exit
fi

echo "You can install VPC Block volume driver"

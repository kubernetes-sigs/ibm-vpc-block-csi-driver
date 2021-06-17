#
# Copyright 2020- IBM Inc. All rights reserved
# SPDX-License-Identifier: Apache2.0
#

## Raw Block Volume
These examples will show how to create a raw PVC and POD and then write data to that

[examples/kubernetes/raw-block-pvc.yaml](./raw-block-pvc.yaml)
(Make sure the `volumeMode` is `Block`.)

[examples/kubernetes/raw-block-pod.yaml](./raw-block-pod.yaml)
(Make sure the pod is consuming the PVC with the defined name and `volumeDevices` is used instead of `volumeMounts`.)

### Deploy the Application
```sh
kubectl apply -f examples/kubernetes/raw-block-pvc.yaml
kubectl apply -f examples/kubernetes/raw-block-pod.yaml
```

### Access Block Device
After the objects are created, verify that pod is running:

```sh
$ kubectl get pods
NAME   			READY   STATUS    RESTARTS   AGE
raw-block-pod    1/1     Running   0          16m
```
Verify the device node is mounted inside the container:

```sh
$ kubectl exec -it raw-block-pod -- ls -al /dev/xvda
brw-rw----    1 root     disk      202, 23296 Mar 12 04:23 /dev/xvda
```

Write to the device using:

```sh
$ kubectl exec -it raw-block-pod sh
$ dd if=/dev/zero of=/dev/xvda bs=1024k count=100
100+0 records in
100+0 records out
104857600 bytes (100.0MB) copied, 0.054862 seconds, 1.8GB/s
```

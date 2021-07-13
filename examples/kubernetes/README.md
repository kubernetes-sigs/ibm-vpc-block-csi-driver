# 
# Copyright 2021 The Kubernetes Authors.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

# #   http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
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

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

## StorageClass secret
We can use the storage class secret to overwrite the default values of storageClass parameters. The example below will show how to specify your PVC settings in a Kubernetes secret and reference this secret in a customized storage class. Then, use the customized storage class to create a PVC with the custom parameters that you set in your secret.

### Enabling every user to customize the default PVC settings

1. In your storage class YAML file [examples/kubernetes/my-storagesecretclass.yaml](./my-storagesecretclass.yaml), reference the Kubernetes secret in the `parameters` section as follows. Make sure to add the code as-is and not to change variables names.

```
csi.storage.k8s.io/provisioner-secret-name: ${pvc.name}
csi.storage.k8s.io/provisioner-secret-namespace: ${pvc.namespace}
```

Following parameters can be overwritten using the storageclass secret,

```
1. iops
2. zone
3. tags
4. encrypted
5. resourceGroup
6. encryptionKey
```

2. As the cluster user, create a Kubernetes secret like [examples/kubernetes/storageclass-secret.yaml](./storageclass-secret.yaml) which has all the possible parameters that can be overwritten.

3. Create your Kubernetes secret.

```
kubectl apply -f volume-secret.yaml
```

4. Create PVC like [examples/kubernetes/pvc-secret.yaml](./pvc-secret.yaml)


## Custom Filesystem Formatting Options (mkfsOptions)

You can specify custom mkfs options in your StorageClass to control how filesystems are formatted on new volumes. This is useful for optimizing volume formatting performance or customizing filesystem features.

### Using mkfsOptions in StorageClass

Add the `mkfsOptions` parameter to your StorageClass definition:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: ibmc-vpc-block-custom-mkfs
provisioner: vpc.block.csi.ibm.io
parameters:
  profile: "general-purpose"
  csi.storage.k8s.io/fstype: "ext4"
  mkfsOptions: "-E lazy_itable_init=0,lazy_journal_init=0 -m0"
  billingType: "hourly"
  encrypted: "false"
reclaimPolicy: "Delete"
allowVolumeExpansion: true
```

### Supported Options

The driver validates mkfs options to ensure security and compatibility:

**For ext4/ext3/ext2 filesystems:**
- `-b` (block size)
- `-E` (extended options)
- `-F` (force)
- `-i` (inode size)
- `-I` (inode ratio)
- `-J` (journal options)
- `-m` (reserved blocks percentage)
- `-N` (number of inodes)
- `-O` (filesystem features)
- `-T` (filesystem type)

**For xfs filesystems:**
- `-b` (block size)
- `-d` (data section options)
- `-f` (force)
- `-i` (inode options)
- `-l` (log section options)
- `-m` (metadata options)
- `-n` (naming options)
- `-r` (realtime section options)
- `-s` (sector size)

### Common Use Cases

**1. Faster ext4 formatting (skip lazy initialization):**
```yaml
mkfsOptions: "-E lazy_itable_init=0,lazy_journal_init=0"
```
This speeds up initial volume formatting but may slightly increase mount time.

**2. No reserved blocks for ext4 (maximize usable space):**
```yaml
mkfsOptions: "-m0"
```
By default, ext4 reserves 5% of space for root. This removes that reservation.

**3. Custom block size for XFS:**
```yaml
csi.storage.k8s.io/fstype: "xfs"
mkfsOptions: "-b size=4096 -f"
```

**4. Disable journaling for ext4 (use with caution):**
```yaml
mkfsOptions: "-O ^has_journal"
```
This can improve performance but reduces crash recovery capabilities.

### Example StorageClasses

See the following example StorageClass files:
- [deploy/kubernetes/storageclass/ibmc-vpc-block-custom-mkfs-StorageClass.yaml](../../deploy/kubernetes/storageclass/ibmc-vpc-block-custom-mkfs-StorageClass.yaml) - ext4 with optimized formatting
- [deploy/kubernetes/storageclass/ibmc-vpc-block-xfs-mkfs-StorageClass.yaml](../../deploy/kubernetes/storageclass/ibmc-vpc-block-xfs-mkfs-StorageClass.yaml) - XFS with custom block size

### Security Notes

- Options are validated to prevent command injection
- Only filesystem-specific options are allowed
- Dangerous characters and shell metacharacters are blocked
- Invalid options are silently ignored with a warning in the logs

### Troubleshooting

If your mkfsOptions are not being applied:

1. Check the controller logs for validation errors:
   ```sh
   kubectl logs -n kube-system -l app=ibm-vpc-block-csi-controller
   ```

2. Check the node logs during volume staging:
   ```sh
   kubectl logs -n kube-system -l app=ibm-vpc-block-csi-node
   ```

3. Verify your options are valid for the filesystem type specified in `csi.storage.k8s.io/fstype`

4. Ensure options don't contain forbidden characters (`;`, `|`, `&`, `$`, etc.)
Make sure to create the PVC with the same name as used for storageclass-secret. Using the same name for the secret and the PVC triggers the storage provider to apply the settings of the secret in your PVC.

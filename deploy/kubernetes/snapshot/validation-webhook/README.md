# Snapshot validation webhook
For a better understanding of webhooks and validation webhook in kubernetes, please refer  [this](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/).

In simple terms, the snapshot validation webhook when deployed, validates the input provided for creating/updating snapshot as described below:
1. Without the webhook, although the snapshot won't get created, no error is returned.
2. Kubernetes will return error when you try to update the snapshot with a different pvc name. Without the webhook, no error is returned, but snapshot config will be updated to new pvc given, but it will still be a snapshot of the previous pvc.

To understand more about snapshot validation web hook and it's benefits, refer [snapshot validation webhook]( https://github.com/kubernetes-csi/external-snapshotter/tree/v6.0.1#validating-webhook).

## Deploying snapshot validation webhook
**Note:** Out of the two ways defined below, deploying wihout cert manager works on Kubernetes clusters, but is not verified on openshift clusters. Deploying with cert manager works on both kubernetes and openshift clusters.

### Deploying webhook without cert manager
**Note:** This is not considered to be a production ready method to deploy the certificates. This is only one of many ways to deploy the certificates, it is your responsibility to ensure the security of your cluster. TLS certificates and private keys should be handled with care and you may not want to keep them in plain Kubernetes secrets.

The steps to deploy this webhook is already provided [here](https://github.com/kubernetes-csi/external-snapshotter/blob/v6.0.1/deploy/kubernetes/webhook-example/README.md#how-to-deploy-the-webhook), but the same might not work on your cluster, hence follow these steps (these are the same steps provided in this [READme](https://github.com/kubernetes-csi/external-snapshotter/blob/v6.0.1/deploy/kubernetes/webhook-example/README.md#how-to-deploy-the-webhook) but with some modifications and more details).
1. Clone [external-snapshotter](https://github.com/kubernetes-csi/external-snapshotter) repo. The following steps should be run from the same external-snapshotter folder.
2. Run the `create-cert.sh` script. More details about the script are mentioned below.
```
./deploy/kubernetes/webhook-example/create-cert.sh --service snapshot-validation-service --secret snapshot-validation-secret --namespace default
```
- The script takes 3 arguments - `service` which is name of the k8s service for the webhook, `secret` which is the name of the k8s secret which will hold RSA key and certificate, `namespace` which is the namespace where the pods, secret and service related to this webhook is deployed. If these arguments are not provided, the default values are `service - admission-webhook-example-svc`, `secret - admission-webhook-example-certs`, `namespace - default`.
- First the script creates a pem encoded private RSA key and certificate signing request using openssl, the same are named `server-key.pem` and `server.csr` files respectively.
- Next a [certificate signing request](https://kubernetes.io/docs/reference/access-authn-authz/certificate-signing-requests/) is created using server.csr content as input with kubernetes.io/kubelet-serving signer. To understand more about signers, please refer [this](https://kubernetes.io/docs/reference/access-authn-authz/certificate-signing-requests/#kubernetes-signers). 
- `kubectl approve` command is run to approve the CSR. This automatically generates a signed certificate in the Certificate Signed Request under `.status.certificate` field.
- A kubernetes secret is created with the secret name provided while runnning the script in the given namespace. The secret holds two keys in its data, `cert.pem` which holds the X.509 certificate and `key.pem` which holds the private RSA key. This secret will be used by snapshot validation webhook once deployed.
3. Fetch the certificate authority of the cluster where you are deploying the webhook. You will either find its value or the path leading to it in your KUBECONFIG file, under the name `certificate-authority` under your cluster. Fetch the content of the file (it begins and ends with `-----BEGIN CERTIFICATE-----` and `-----END CERTIFICATE-----` respectively). Encode the file `base64 -i <file>.pem`.
4. Add the base64 encoded value in `deploy/kubernetes/webhook-example/admission-configuration-template` file in front of `caBundle:` parameter and save the same to a file named `admission-configuration.yaml` in the same `webhook-example` directory.
5. Change the namespace to where you prefer to deploy the webhook in `admission-configuration.yaml, rbac-snapshot-webhook.yaml, webhook.yaml` files under `deploy/kubernetes/webhook-example` directory.
6. Run `kubectl apply -f deploy/kubernetes/webhook-example`.
7. Once done, you should be able to see 3 pods, 1 service and 1 secret in the provided namespace as shown below.
```
% kubectl get pods -n kube-system | grep snapshot-validation
snapshot-validation-deployment-6f88cc8b87-fdl4p       1/1     Running     0                6d16h
snapshot-validation-deployment-6f88cc8b87-jlc5g       1/1     Running     0                6d16h
snapshot-validation-deployment-6f88cc8b87-mmbd2       1/1     Running     0                6d16h

% kubectl get svc -n kube-system | grep snapshot-validation
snapshot-validation-service          ClusterIP      <CLUSTER-IP>   <none>                               443/TCP                                                       12d

% kubectl get secrets -n kube-system | grep snapshot-validation
snapshot-validation-secret                         Opaque                                2      6d16h
```
8. Check pod logs. 
- If you see the following, web hook is up and running.
    ```
    % kubectl logs -f snapshot-validation-deployment-6f88cc8b87-fdl4p -n kube-system
    I0921 18:15:14.545269    1 certwatcher.go:129] Updated current TLS certificate
    Starting webhook server
    I0921 18:15:14.645808    1 webhook.go:196] Starting certificate watcher
    ```
- If the following logs are observed, it indicates that the kubernetes secret used by webhook is not
populated with the cert and key value OR the certificate provided in the `caBundle:` may not be right. If neither is the case, reach out to the community - raise an issue [here](https://github.com/kubernetes-csi/external-snapshotter/issues).
    ```
    2022/09/22 18:38:49 http: TLS handshake error from <IP>:<PORT>: remote error: tls: bad certificate
    2022/09/22 18:38:50 http: TLS handshake error from <IP>:<PORT>: remote error: tls: bad certificate
    2022/09/22 18:38:51 http: TLS handshake error from <IP>:<PORT>: remote error: tls: bad certificate
    2022/09/22 18:38:52 http: TLS handshake error from <IP>:<PORT>: remote error: tls: bad certificate
    2022/09/22 18:38:53 http: TLS handshake error from <IP>:<PORT>: remote error: tls: bad certificate
    ```

### Deploying webhook with cert manager
The cert manager takes care of generating and managing the certificates required for the webhook which reduces the overhead of generating certs manually using openssl and csr. Please refer to the [cert manager](https://cert-manager.io/docs/) to learn more. Using cert manager for the snapshot validation webhook can be made more flexible. More information will be provided in the following steps. The [installation doc](https://cert-manager.io/docs/installation/kubectl/) for cert-manager has the steps to install and verify cert manager. But, please follow the steps given below which are specific to snapshot validation webhook.

1. Run `kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.yaml`.
    You should be able to see the following pods and service running in `cert-manager` namespace.
    ```
    % kubectl get pods -n cert-manager
    NAME                                       READY   STATUS    RESTARTS   AGE
    cert-manager-5dd59d9d9b-n8lfp              1/1     Running   0          6d14h
    cert-manager-cainjector-8696fc9f89-2mf7p   1/1     Running   0          6d14h
    cert-manager-webhook-7d4b5b8c56-wmdzx      1/1     Running   0          6d14h
    % kubectl get svc -n cert-manager
    NAME                   TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)    AGE
    cert-manager           ClusterIP   172.21.233.3     <none>        9402/TCP   6d14h
    cert-manager-webhook   ClusterIP   172.21.201.208   <none>        443/TCP    6d14h
    ```
    If any issues are observed, you may check in the [cert-manager community](https://github.com/cert-manager/cert-manager/issues).
2. Update the following yaml with the `namespace` (under `dnsNames`) where you want to deploy the snapshot validation webhook. `Issuer` and `Certificate` are CRDs which get created when the cert-manager is deployed. You can specify any namespace to deploy these resources (specified as <any-namespace> in this yaml file). `kubectl apply -f cert-issuer.yaml`. To know more about  `Issuers` refer [this](https://cert-manager.io/docs/concepts/issuer/).
    ```
    ## cert-issuer.yaml
    apiVersion: cert-manager.io/v1
    kind: Issuer
    metadata:
      name: selfsigned-issuer
      namespace: <any-namespace>
    spec:
      selfSigned: {}
    ---
    apiVersion: cert-manager.io/v1
    kind: Certificate
    metadata:
      name: selfsigned-cert
      namespace: <any-namespace>
    spec:
      dnsNames:
        - snapshot-validation-service.<namespace>.svc
        - snapshot-validation-service.<namespace>
        - snapshot-validation-service
      secretName: selfsigned-cert-tls
      issuerRef:
        name: selfsigned-issuer
    ```
3. After creating the above, you should be able see a k8s secret and 3 keys (base64 encoded) - `ca.crt, tls.crt, tls.key` in its data as shown below seen here.
    ```
    % kubectl get secret selfsigned-cert-tls  -n kube-system
    NAME                  TYPE                DATA   AGE
    selfsigned-cert-tls   kubernetes.io/tls   3      8s
    % kubectl get secret selfsigned-cert-tls  -n kube-system -oyaml
    apiVersion: v1
    data:
      ca.crt: <base64-encoded-value-of-certificate>
      tls.crt: <base64-encoded-value-of-certificate>
      tls.key: <base64-encoded-value-of-private-RSA-key>
    kind: Secret
    metadata:
      annotations:
        cert-manager.io/alt-names: snapshot-validation-service.kube-system.svc,snapshot-validation-service.kube-system,snapshot-validation-service
        cert-manager.io/certificate-name: selfsigned-cert
        cert-manager.io/common-name: ""
        cert-manager.io/ip-sans: ""
        cert-manager.io/issuer-group: ""
        cert-manager.io/issuer-kind: Issuer
        cert-manager.io/issuer-name: selfsigned-issuer
        cert-manager.io/uri-sans: ""
      creationTimestamp: "2022-09-29T06:56:44Z"
      name: selfsigned-cert-tls
      namespace: kube-system
      resourceVersion: "22902189"
      uid: <uid>
    type: kubernetes.io/tls
    ```
4. Create a k8s secret with CA cert and RSA key. Populate  `cert.pem` and `key.pem` with  `tls.crt` and `tls.key` value obtained from the secret created in the previous step. Substitute the namespace. `kubectl create -f snapshot-validation-secret.yaml`.
    ```
    ## snapshot-validation-secret.yaml
    apiVersion: v1
    data:
      cert.pem: <value-obtained-from-tls.crt>
      key.pem: <value-obtained-from-tls.key>
    kind: Secret
    metadata:
      name: snapshot-validation-secret
      namespace: <namespace-where-snapshot-validation-webhook-will-be-deployed>
    type: Opaque
    ```
5. Clone [external-snapshotter](https://github.com/kubernetes-csi/external-snapshotter) repo. The following steps should be run from the same external-snapshotter folder.
6. Add the `ca.crt` value obtained from the `selfsigned-cert-tls` secret in `deploy/kubernetes/webhook-example/admission-configuration-template` file in front of `caBundle:` parameter and save the same to a file named `admission-configuration.yaml` in the same `webhook-example` directory.
7. Change the namespace to where you prefer to deploy the webhook in `admission-configuration.yaml, rbac-snapshot-webhook.yaml, webhook.yaml` files under `deploy/kubernetes/webhook-example` directory.
8. Run `kubectl apply -f webhook-example`.
9. Once done, you should be able to see 3 pods, 1 service and 1 secret in the provided namespace as shown below.
    ```
    % kubectl get pods -n kube-system | grep snapshot-validation
    snapshot-validation-deployment-6f88cc8b87-fdl4p       1/1     Running     0                6d16h
    snapshot-validation-deployment-6f88cc8b87-jlc5g       1/1     Running     0                6d16h
    snapshot-validation-deployment-6f88cc8b87-mmbd2       1/1     Running     0                6d16h

    % kubectl get svc -n kube-system | grep snapshot-validation
    snapshot-validation-service          ClusterIP      <CLUSTER-IP>   <none>                               443/TCP                                                       12d

    % kubectl get secrets -n kube-system | grep snapshot-validation
    snapshot-validation-secret                         Opaque                                2      6d16h
    ```
10. Check pod logs.
- If you see the following, web hook is up and running.
    ```
    kubectl logs -f snapshot-validation-deployment-6f88cc8b87-fdl4p -n kube-system
    I0921 18:15:14.545269    1 certwatcher.go:129] Updated current TLS certificate
    Starting webhook server
    I0921 18:15:14.645808    1 webhook.go:196] Starting certificate watcher
    ```
**Note**: Few of the above steps can be automated by the means of init container or sidecars, which can automatically copy the certs. More about the same will be updated eventually.

### Testing webhook functionality
Provided that the snapshot validation webhook is deployed:
1. If you try to create a snapshot without providing the `volumeSnapshotClassName`, you will see the following error.
    ```
     % kubectl create -f volumeSnapshot.yaml
    Error from server: error when creating "volumeSnapshot.yaml": admission webhook "validation-webhook.snapshot.storage.k8s.io" denied the request: Spec.VolumeSnapshotClassName must not be the empty string
    ```
2. If you try to update a snapshot with a different pvc name using `kubectl edit`, you will see the following error.
    ```
    % kubectl edit volumesnapshot snapshot-0 -n default
    error: volumesnapshots.snapshot.storage.k8s.io "snapshot-0" could not be patched: admission webhook "validation-webhook.snapshot.storage.k8s.io" denied the request: Spec.Source.PersistentVolumeClaimName is immutable but was changed from pvc1 to pvc2
    ```
Without the webhook, these errors will not be seen.
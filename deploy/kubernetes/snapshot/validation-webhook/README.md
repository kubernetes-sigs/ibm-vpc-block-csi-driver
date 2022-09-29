# Snapshot validation webhook
For a better understanding of webhooks and validation webhook in kubernetes, please refer  [this](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/).

In simple terms, the snapshot validation webhook when deployed, validates the input provided for creating/updating snapshot as described below:
1. Without the webhook, although the snapshot won't get created, no error is returned.
2. Kubernetes will return error when you try to update the snapshot with a different pvc name. Without the webhook, no error is returned, but snapshot config will be updated to new pvc given, but it will still be a snapshot of the previous pvc.

To understand more about snapshot validation web hook and it's benefits, refer [snapshot validation webhook]( https://github.com/kubernetes-csi/external-snapshotter/tree/v6.0.1#validating-webhook)

## Deploying snapshot validation webhook
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
3. Fetch the certificate authority of the cluster where you are deploying the webhook. You will either find its value or tha path leading to it in your KUBECONFIG file, under the name `certificate-authority` under your cluster. Fetch the content of the file (it begins and ends with `-----BEGIN CERTIFICATE-----` and `-----END CERTIFICATE-----` respectively). Encode the file `base64 -i <file>.pem`.
4. Add the base64 encoded value in `deploy/kubernetes/webhook-example/admission-configuration-template` file in front of `caBundle:` parameter and save the same to a file named `admission-configuration.yaml` in the same `webhook-example` directory.
5. Change the namespace to where you prefer to deploy the webhook in `admission-configuration.yaml, rbac-snapshot-webhook.yaml, webhook.yaml` files under `deploy/kubernetes/webhook-example` directory
6. Run `kubectl apply -f webhook-example`
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
- If you see the following, web hook is up and running
    ```
    kubectl logs -f snapshot-validation-deployment-6f88cc8b87-fdl4p -n kube-system
    I0921 18:15:14.545269    1 certwatcher.go:129] Updated current TLS certificate
    Starting webhook server
    I0921 18:15:14.645808    1 webhook.go:196] Starting certificate watcher
    ```
- If the following logs are observed, 
    ```
    2022/09/22 18:38:49 http: TLS handshake error from <IP>:<PORT>: remote error: tls: bad certificate
    2022/09/22 18:38:50 http: TLS handshake error from <IP>:<PORT>: remote error: tls: bad certificate
    2022/09/22 18:38:51 http: TLS handshake error from <IP>:<PORT>: remote error: tls: bad certificate
    2022/09/22 18:38:52 http: TLS handshake error from <IP>:<PORT>: remote error: tls: bad certificate
    2022/09/22 18:38:53 http: TLS handshake error from <IP>:<PORT>: remote error: tls: bad certificate
    ```
    The k8s secret used by webhook is not populated with the cert and key value. 
    OR
    The certificate provided in the `caBundle:` may not be right. If neither is the case, reach out to the community - raise an issue [here](https://github.com/kubernetes-csi/external-snapshotter/issues).

### Deploying webhook with cert manager

# NodeSet Controller

## Building

Build the binary and upload:

```shell
$ make REPO=<ORG>/nodeset-controller
```

## Deployment

### Node

Create the ReplicaSet:

``` shell
$ kubectl create -f cmd/nodeset-controller/node-replicaset.yaml
```

### GKE

Create a service account in Google Cloud:

``` shell
$ gcloud iam service-accounts create nodeset
$ gcloud projects add-iam-policy-binding <PROJECT> --member serviceAccount:nodeset@<PROJECT>.iam.gserviceaccount.com --role roles/compute.instanceAdmin --role roles/container.viewer
```

Create the service account token as a secret:

``` shell
$ gcloud iam service-accounts keys create secret.json --iam-account=nodeset@<PROJECT>.iam.gserviceaccount.com
$ kubectl create secret generic nodeset-gcloud-service-account --from-file=SERVICE_ACCOUNT_FILE.json --namespace kube-system
```

Update the specific values in the `gke-replicaset.yaml`

* `gke-cluster-id`
* `gke-project-id`
* `gke-zone`

Create the ReplicaSet:

``` shell
$ kubectl create -f cmd/nodeset-controller/gke-replicaset.yaml
```

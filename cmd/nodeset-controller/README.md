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
$ jq -r ".private_key" secret.json > private_key
$ kubectl create secret generic nodeset-gcloud-service-account --from-file=private_key --namespace kube-system
```

Create the ReplicaSet:

``` shell
$ kubectl create -f cmd/nodeset-controller/gke-replicaset.yaml
```

# Nodeset
Generic nodeset for Kubernetes

## API Proposal
[here](https://github.com/kube-node/nodeset/blob/master/proposal.md)

## Deployment
[here](https://github.com/kube-node/nodeset/blob/master/cmd/nodeset-controller/README.md)

## Examples

### NodeSet

```yaml
apiVersion: nodeset.k8s.io/v1alpha1
kind: NodeSet
metadata:
  name: my-nodeset
spec:
  nodeClass: "nodeclass-do-2gb"
  nodeSetController: "default"
  replicas: 5
  maxUnavailable: 0
  maxSurge: 1
```

### NodeClass (For [Kube-Machine](https://github.com/kube-node/kube-machine))

```yaml
apiVersion: nodeset.k8s.io/v1alpha1
kind: NodeClass
metadata:
  name: nodeclass-do-2gb
nodeController: kube-machine
config:
  dockerMachineFlags:
    digitalocean-access-token: YOUR-DO-TOKEN
    digitalocean-image: coreos-stable
    digitalocean-private-networking: "true"
    digitalocean-region: fra1
    digitalocean-size: 2gb
    digitalocean-ssh-user: core
  provider: digitalocean
  provisioning:
    commands:
    - sudo chmod +x /opt/bin/bootstrap.sh && sudo /opt/bin/bootstrap.sh
    - sudo systemctl enable kubelet && sudo systemctl start kubelet
    files:
    - content: |-
        YOUR-BOOTSTRAP-KUBECONFIG
      owner: root
      path: /etc/kubernetes/bootstrap.kubeconfig
      permissions: "0640"
    - content: |-
        SOME-BOOTSTRAP-BASH-SCRIPT
      owner: root
      path: /opt/bin/bootstrap.sh
      permissions: "0750"
    - content: |-
        THE-KUBELET-SYSTEMD-UNIT-FILE
      owner: root
      path: /etc/systemd/system/kubelet.service
      permissions: "0640"
    users:
    - name: apiserver
      ssh_keys:
      - SOME-SSH-PUBLIC-KEY
      sudo: true
```

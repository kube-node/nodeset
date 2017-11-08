/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package node

import (
	"fmt"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
)

func (c *Controller) syncNodeSet(nodeset *v1alpha1.NodeSet) (changedNodeSet *v1alpha1.NodeSet, err error) {
	changedNodeSet, err = c.syncNodeSetVersion(nodeset)
	if err != nil {
		return nil, fmt.Errorf("failed to sync nodeset version: %v", err)
	}
	if changedNodeSet != nil {
		return changedNodeSet, nil
	}

	changedNodeSet, err = c.syncNodeSetStatusReplicas(nodeset)
	if err != nil {
		return nil, fmt.Errorf("failed to sync replicas: %v", err)
	}
	if changedNodeSet != nil {
		return changedNodeSet, nil
	}

	err = c.syncNodeSetReplicas(nodeset)
	if err != nil {
		return nil, fmt.Errorf("failed to sync replicas: %v", err)
	}

	return nil, nil
}

func (c *Controller) syncNodeSetVersion(nodeset *v1alpha1.NodeSet) (changedNodeSet *v1alpha1.NodeSet, err error) {
	nodeClass, err := c.nodeClassLister.Get(nodeset.Spec.NodeClass)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodeclass: %v", err)
	}

	mergedNodeClass, err := mergeNodeClass(nodeClass, nodeset.Spec.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to merge nodeclass: %v", err)
	}

	_, hash, err := convertNodeClassToString(mergedNodeClass)
	if err != nil {
		return nil, fmt.Errorf("failed to convert merged nodeclass to string: %v", err)
	}

	if nodeset.Annotations[v1alpha1.NodeClassContentHashAnnotationKey] != hash {
		nodeset.Annotations[v1alpha1.NodeClassContentHashAnnotationKey] = hash
		nodeset.Status.ObservedGeneration = nodeset.Status.ObservedGeneration + 1
		return nodeset, nil
	}
	return nil, nil
}

func (c *Controller) syncNodeSetReplicas(nodeset *v1alpha1.NodeSet) error {
	currentNodes, err := c.nodesetNodes(nodeset)
	if err != nil {
		return err
	}
	currentNodeCount := int32(len(currentNodes))

	if nodeset.Spec.Replicas > currentNodeCount {
		glog.Infof("Creating node ( spec.replicas(%d) > currentNodeCount(%d) )", nodeset.Spec.Replicas, currentNodeCount)
		node, err := c.createNode(nodeset)
		if err != nil {
			return err
		}
		_, err = c.kubeClient.CoreV1().Nodes().Create(node)
		if err != nil {
			return err
		}
		if err = waitForNodeInCache(node.Name, c.nodeLister); err != nil {
			return err
		}
	} else if nodeset.Spec.Replicas < currentNodeCount {
		glog.Infof("Deleting node ( spec.replicas(%d) < currentNodeCount(%d) )", nodeset.Spec.Replicas, currentNodeCount)
		err = c.kubeClient.CoreV1().Nodes().Delete(currentNodes[0].Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
		if err = waitForNodeDeletedInCache(currentNodes[0].Name, c.nodeLister); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) syncNodeSetStatusReplicas(nodeset *v1alpha1.NodeSet) (changedNodeSet *v1alpha1.NodeSet, err error) {
	nodes, err := c.nodesetNodes(nodeset)
	if err != nil {
		return nil, err
	}

	runningNodes := []*corev1.Node{}
	for _, n := range nodes {
		if nodeIsRunning(n) {
			runningNodes = append(runningNodes, n)
		}
	}
	if nodeset.Status.RunningReplicas != int32(len(runningNodes)) || nodeset.Status.Replicas != int32(len(nodes)) {
		nodeset.Status.RunningReplicas = int32(len(runningNodes))
		nodeset.Status.Replicas = int32(len(nodes))

		return nodeset, nil
	}

	return nil, nil
}

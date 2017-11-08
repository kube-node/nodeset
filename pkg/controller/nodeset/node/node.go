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
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
)

func (c *Controller) nodesetNodes(nodeset *v1alpha1.NodeSet) ([]*corev1.Node, error) {
	objs, err := c.nodeIndexer.ByIndex(OwnerUIDIndex, string(nodeset.GetUID()))
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %v", err)
	}

	nodes := make([]*corev1.Node, 0, len(objs))
	for _, n := range objs {
		nodes = append(nodes, n.(*corev1.Node))
	}
	return nodes, nil
}

func (c *Controller) createNode(nodeset *v1alpha1.NodeSet) (*corev1.Node, error) {
	nodeClass, err := c.nodeClassLister.Get(nodeset.Spec.NodeClass)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodeclass: %v", err)
	}
	nodeName := nodeset.Name + "-" + string(uuid.NewUUID())[:6]
	labels := map[string]string{}
	for k, v := range nodeClass.NodeLabels {
		labels[k] = v
	}
	labels[v1alpha1.NodeSetNameLabelKey] = nodeset.Name
	labels[v1alpha1.ControllerLabelKey] = nodeClass.NodeController
	labels[kubeHostnameLabelKey] = nodeName

	mergedNodeClass, err := mergeNodeClass(nodeClass, nodeset.Spec.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to merge nodeclass: %v", err)
	}

	content, _, err := convertNodeClassToString(mergedNodeClass)
	if err != nil {
		return nil, fmt.Errorf("failed to convert merged nodeclass to string: %v", err)
	}

	gv := v1alpha1.SchemeGroupVersion
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   nodeName,
			Labels: labels,
			Annotations: map[string]string{
				v1alpha1.NodeClassContentAnnotationKey:  content,
				v1alpha1.NodeSetGenerationAnnotationKey: strconv.Itoa(int(nodeset.Status.ObservedGeneration)),
			},
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(nodeset, gv.WithKind("NodeSet"))},
		},
	}
	return node, nil
}

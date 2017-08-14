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

package fake

import (
	v1alpha1 "github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeNodeSets implements NodeSetInterface
type FakeNodeSets struct {
	Fake *FakeNodesetV1alpha1
}

var nodesetsResource = schema.GroupVersionResource{Group: "nodeset", Version: "v1alpha1", Resource: "nodesets"}

var nodesetsKind = schema.GroupVersionKind{Group: "nodeset", Version: "v1alpha1", Kind: "NodeSet"}

// Get takes name of the nodeSet, and returns the corresponding nodeSet object, and an error if there is any.
func (c *FakeNodeSets) Get(name string, options v1.GetOptions) (result *v1alpha1.NodeSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(nodesetsResource, name), &v1alpha1.NodeSet{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeSet), err
}

// List takes label and field selectors, and returns the list of NodeSets that match those selectors.
func (c *FakeNodeSets) List(opts v1.ListOptions) (result *v1alpha1.NodeSetList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(nodesetsResource, nodesetsKind, opts), &v1alpha1.NodeSetList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.NodeSetList{}
	for _, item := range obj.(*v1alpha1.NodeSetList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested nodeSets.
func (c *FakeNodeSets) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(nodesetsResource, opts))
}

// Create takes the representation of a nodeSet and creates it.  Returns the server's representation of the nodeSet, and an error, if there is any.
func (c *FakeNodeSets) Create(nodeSet *v1alpha1.NodeSet) (result *v1alpha1.NodeSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(nodesetsResource, nodeSet), &v1alpha1.NodeSet{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeSet), err
}

// Update takes the representation of a nodeSet and updates it. Returns the server's representation of the nodeSet, and an error, if there is any.
func (c *FakeNodeSets) Update(nodeSet *v1alpha1.NodeSet) (result *v1alpha1.NodeSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(nodesetsResource, nodeSet), &v1alpha1.NodeSet{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeSet), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeNodeSets) UpdateStatus(nodeSet *v1alpha1.NodeSet) (*v1alpha1.NodeSet, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(nodesetsResource, "status", nodeSet), &v1alpha1.NodeSet{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeSet), err
}

// Delete takes name of the nodeSet and deletes it. Returns an error if one occurs.
func (c *FakeNodeSets) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(nodesetsResource, name), &v1alpha1.NodeSet{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeNodeSets) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(nodesetsResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.NodeSetList{})
	return err
}

// Patch applies the patch and returns the patched nodeSet.
func (c *FakeNodeSets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.NodeSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(nodesetsResource, name, data, subresources...), &v1alpha1.NodeSet{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeSet), err
}

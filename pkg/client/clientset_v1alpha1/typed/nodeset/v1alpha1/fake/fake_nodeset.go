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
	ns   string
}

var nodesetsResource = schema.GroupVersionResource{Group: "nodeset", Version: "v1alpha1", Resource: "nodesets"}

func (c *FakeNodeSets) Create(nodeSet *v1alpha1.NodeSet) (result *v1alpha1.NodeSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(nodesetsResource, c.ns, nodeSet), &v1alpha1.NodeSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeSet), err
}

func (c *FakeNodeSets) Update(nodeSet *v1alpha1.NodeSet) (result *v1alpha1.NodeSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(nodesetsResource, c.ns, nodeSet), &v1alpha1.NodeSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeSet), err
}

func (c *FakeNodeSets) UpdateStatus(nodeSet *v1alpha1.NodeSet) (*v1alpha1.NodeSet, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(nodesetsResource, "status", c.ns, nodeSet), &v1alpha1.NodeSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeSet), err
}

func (c *FakeNodeSets) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(nodesetsResource, c.ns, name), &v1alpha1.NodeSet{})

	return err
}

func (c *FakeNodeSets) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(nodesetsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.NodeSetList{})
	return err
}

func (c *FakeNodeSets) Get(name string, options v1.GetOptions) (result *v1alpha1.NodeSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(nodesetsResource, c.ns, name), &v1alpha1.NodeSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeSet), err
}

func (c *FakeNodeSets) List(opts v1.ListOptions) (result *v1alpha1.NodeSetList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(nodesetsResource, c.ns, opts), &v1alpha1.NodeSetList{})

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
		InvokesWatch(testing.NewWatchAction(nodesetsResource, c.ns, opts))

}

// Patch applies the patch and returns the patched nodeSet.
func (c *FakeNodeSets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.NodeSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(nodesetsResource, c.ns, name, data, subresources...), &v1alpha1.NodeSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeSet), err
}

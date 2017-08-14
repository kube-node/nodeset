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

// FakeNodeClasses implements NodeClassInterface
type FakeNodeClasses struct {
	Fake *FakeNodesetV1alpha1
}

var nodeclassesResource = schema.GroupVersionResource{Group: "nodeset", Version: "v1alpha1", Resource: "nodeclasses"}

var nodeclassesKind = schema.GroupVersionKind{Group: "nodeset", Version: "v1alpha1", Kind: "NodeClass"}

// Get takes name of the nodeClass, and returns the corresponding nodeClass object, and an error if there is any.
func (c *FakeNodeClasses) Get(name string, options v1.GetOptions) (result *v1alpha1.NodeClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(nodeclassesResource, name), &v1alpha1.NodeClass{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeClass), err
}

// List takes label and field selectors, and returns the list of NodeClasses that match those selectors.
func (c *FakeNodeClasses) List(opts v1.ListOptions) (result *v1alpha1.NodeClassList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(nodeclassesResource, nodeclassesKind, opts), &v1alpha1.NodeClassList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.NodeClassList{}
	for _, item := range obj.(*v1alpha1.NodeClassList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested nodeClasses.
func (c *FakeNodeClasses) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(nodeclassesResource, opts))
}

// Create takes the representation of a nodeClass and creates it.  Returns the server's representation of the nodeClass, and an error, if there is any.
func (c *FakeNodeClasses) Create(nodeClass *v1alpha1.NodeClass) (result *v1alpha1.NodeClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(nodeclassesResource, nodeClass), &v1alpha1.NodeClass{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeClass), err
}

// Update takes the representation of a nodeClass and updates it. Returns the server's representation of the nodeClass, and an error, if there is any.
func (c *FakeNodeClasses) Update(nodeClass *v1alpha1.NodeClass) (result *v1alpha1.NodeClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(nodeclassesResource, nodeClass), &v1alpha1.NodeClass{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeClass), err
}

// Delete takes name of the nodeClass and deletes it. Returns an error if one occurs.
func (c *FakeNodeClasses) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(nodeclassesResource, name), &v1alpha1.NodeClass{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeNodeClasses) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(nodeclassesResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.NodeClassList{})
	return err
}

// Patch applies the patch and returns the patched nodeClass.
func (c *FakeNodeClasses) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.NodeClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(nodeclassesResource, name, data, subresources...), &v1alpha1.NodeClass{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NodeClass), err
}

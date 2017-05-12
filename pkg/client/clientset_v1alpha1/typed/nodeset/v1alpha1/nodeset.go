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

package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	scheme "kube-node/nodeset/pkg/client/clientset_v1alpha1/scheme"
	v1alpha1 "kube-node/nodeset/pkg/nodeset/v1alpha1"
)

// NodeSetsGetter has a method to return a NodeSetInterface.
// A group's client should implement this interface.
type NodeSetsGetter interface {
	NodeSets(namespace string) NodeSetInterface
}

// NodeSetInterface has methods to work with NodeSet resources.
type NodeSetInterface interface {
	Create(*v1alpha1.NodeSet) (*v1alpha1.NodeSet, error)
	Update(*v1alpha1.NodeSet) (*v1alpha1.NodeSet, error)
	UpdateStatus(*v1alpha1.NodeSet) (*v1alpha1.NodeSet, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.NodeSet, error)
	List(opts v1.ListOptions) (*v1alpha1.NodeSetList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.NodeSet, err error)
	NodeSetExpansion
}

// nodeSets implements NodeSetInterface
type nodeSets struct {
	client rest.Interface
	ns     string
}

// newNodeSets returns a NodeSets
func newNodeSets(c *NodesetV1alpha1Client, namespace string) *nodeSets {
	return &nodeSets{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Create takes the representation of a nodeSet and creates it.  Returns the server's representation of the nodeSet, and an error, if there is any.
func (c *nodeSets) Create(nodeSet *v1alpha1.NodeSet) (result *v1alpha1.NodeSet, err error) {
	result = &v1alpha1.NodeSet{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("nodesets").
		Body(nodeSet).
		Do().
		Into(result)
	return
}

// Update takes the representation of a nodeSet and updates it. Returns the server's representation of the nodeSet, and an error, if there is any.
func (c *nodeSets) Update(nodeSet *v1alpha1.NodeSet) (result *v1alpha1.NodeSet, err error) {
	result = &v1alpha1.NodeSet{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("nodesets").
		Name(nodeSet.Name).
		Body(nodeSet).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclientstatus=false comment above the type to avoid generating UpdateStatus().

func (c *nodeSets) UpdateStatus(nodeSet *v1alpha1.NodeSet) (result *v1alpha1.NodeSet, err error) {
	result = &v1alpha1.NodeSet{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("nodesets").
		Name(nodeSet.Name).
		SubResource("status").
		Body(nodeSet).
		Do().
		Into(result)
	return
}

// Delete takes name of the nodeSet and deletes it. Returns an error if one occurs.
func (c *nodeSets) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("nodesets").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *nodeSets) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("nodesets").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the nodeSet, and returns the corresponding nodeSet object, and an error if there is any.
func (c *nodeSets) Get(name string, options v1.GetOptions) (result *v1alpha1.NodeSet, err error) {
	result = &v1alpha1.NodeSet{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("nodesets").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of NodeSets that match those selectors.
func (c *nodeSets) List(opts v1.ListOptions) (result *v1alpha1.NodeSetList, err error) {
	result = &v1alpha1.NodeSetList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("nodesets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested nodeSets.
func (c *nodeSets) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("nodesets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched nodeSet.
func (c *nodeSets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.NodeSet, err error) {
	result = &v1alpha1.NodeSet{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("nodesets").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}

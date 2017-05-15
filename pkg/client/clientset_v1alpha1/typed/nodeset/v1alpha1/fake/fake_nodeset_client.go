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
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
	v1alpha1 "kube-node/nodeset/pkg/client/clientset_v1alpha1/typed/nodeset/v1alpha1"
)

type FakeNodesetV1alpha1 struct {
	*testing.Fake
}

func (c *FakeNodesetV1alpha1) NodeClasses() v1alpha1.NodeClassInterface {
	return &FakeNodeClasses{c}
}

func (c *FakeNodesetV1alpha1) NodeSets(namespace string) v1alpha1.NodeSetInterface {
	return &FakeNodeSets{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeNodesetV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}

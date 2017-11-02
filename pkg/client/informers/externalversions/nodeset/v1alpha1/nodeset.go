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

// This file was automatically generated by informer-gen

package v1alpha1

import (
	versioned "github.com/kube-node/nodeset/pkg/client/clientset/versioned"
	internalinterfaces "github.com/kube-node/nodeset/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/kube-node/nodeset/pkg/client/listers/nodeset/v1alpha1"
	nodeset_v1alpha1 "github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	time "time"
)

// NodeSetInformer provides access to a shared informer and lister for
// NodeSets.
type NodeSetInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.NodeSetLister
}

type nodeSetInformer struct {
	factory internalinterfaces.SharedInformerFactory
}

// NewNodeSetInformer constructs a new informer for NodeSet type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewNodeSetInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return client.NodesetV1alpha1().NodeSets().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return client.NodesetV1alpha1().NodeSets().Watch(options)
			},
		},
		&nodeset_v1alpha1.NodeSet{},
		resyncPeriod,
		indexers,
	)
}

func defaultNodeSetInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewNodeSetInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
}

func (f *nodeSetInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&nodeset_v1alpha1.NodeSet{}, defaultNodeSetInformer)
}

func (f *nodeSetInformer) Lister() v1alpha1.NodeSetLister {
	return v1alpha1.NewNodeSetLister(f.Informer().GetIndexer())
}

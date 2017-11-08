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
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	nodesetclientset "github.com/kube-node/nodeset/pkg/client/clientset/versioned"
	nodesetinformers "github.com/kube-node/nodeset/pkg/client/informers/externalversions/nodeset/v1alpha1"
	nodesetlisters "github.com/kube-node/nodeset/pkg/client/listers/nodeset/v1alpha1"
	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
)

// Controller is a generic implementation of a nodeset controller which creates node resources
type Controller struct {
	kubeClient    kubernetes.Interface
	nodesetClient nodesetclientset.Interface

	nodesetLister  nodesetlisters.NodeSetLister
	nodesetIndexer cache.Indexer
	nodesetSynced  cache.InformerSynced

	nodeClassLister  nodesetlisters.NodeClassLister
	nodeClassIndexer cache.Indexer
	nodeClassSynced  cache.InformerSynced

	nodeLister  corelisters.NodeLister
	nodeIndexer cache.Indexer
	nodeSynced  cache.InformerSynced

	name string

	// queue is where incoming work is placed to de-dup and to allow "easy"
	// rate limited requeues on errors
	queue workqueue.RateLimitingInterface
}

// New returns a instance of the node nodeset controller
func New(name string, kubeClient kubernetes.Interface, nodesetClient nodesetclientset.Interface, nodesets nodesetinformers.NodeSetInformer, nodeclasses nodesetinformers.NodeClassInformer, nodes coreinformers.NodeInformer) (*Controller, error) {
	// index nodesets by uids
	// TODO: move outside of New
	err := nodesets.Informer().AddIndexers(map[string]cache.IndexFunc{
		UIDIndex: MetaUIDIndexFunc,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add uid indexer: %v", err)
	}

	// index nodes by owner uid
	err = nodes.Informer().AddIndexers(map[string]cache.IndexFunc{
		OwnerUIDIndex: MetaOwnerUIDIndexFunc,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add owner uid indexer: %v", err)
	}

	c := &Controller{
		kubeClient:    kubeClient,
		nodesetClient: nodesetClient,

		nodesetLister:  nodesets.Lister(),
		nodesetIndexer: nodesets.Informer().GetIndexer(),
		nodesetSynced:  nodesets.Informer().HasSynced,

		nodeClassLister:  nodeclasses.Lister(),
		nodeClassIndexer: nodeclasses.Informer().GetIndexer(),
		nodeClassSynced:  nodeclasses.Informer().HasSynced,

		nodeLister:  nodes.Lister(),
		nodeIndexer: nodes.Informer().GetIndexer(),
		nodeSynced:  nodes.Informer().HasSynced,

		name: name,

		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "nodeset"),
	}
	// register event handlers to fill the queue with nodeset creations, updates and deletions
	nodesets.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				c.queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta nodesetQueue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.queue.Add(key)
			}
		},
	})

	queueOwner := func(node *corev1.Node) {
		owner := metav1.GetControllerOf(node)
		if owner == nil {
			return
		}

		objs, err := c.nodesetIndexer.ByIndex(UIDIndex, string(owner.UID))
		if err != nil {
			return
		}

		for _, set := range objs {
			key, err := cache.MetaNamespaceKeyFunc(set)
			if err == nil {
				c.queue.Add(key)
			}
		}
	}
	nodes.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			node, ok := obj.(*corev1.Node)
			if !ok {
				return
			}
			queueOwner(node)
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			node, ok := new.(*corev1.Node)
			if !ok {
				return
			}
			queueOwner(node)
		},
		DeleteFunc: func(obj interface{}) {
			node, ok := obj.(*corev1.Node)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %+v", obj))
					return
				}
				node, ok = tombstone.Obj.(*corev1.Node)
				if !ok {
					utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a node %#v", obj))
					return
				}
			}
			queueOwner(node)
		},
	})

	return c, nil
}

// Run starts the control loop. This method is blocking.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) {
	// don't let panics crash the process
	defer utilruntime.HandleCrash()
	// make sure the work queue is shutdown which will trigger workers to end
	defer c.queue.ShutDown()

	glog.Infof("Starting controller")

	// wait for your secondary caches to fill before starting your work
	if !cache.WaitForCacheSync(stopCh, c.nodesetSynced, c.nodeSynced, c.nodeClassSynced) {
		return
	}

	// start up your worker threads based on threadiness.  Some controllers
	// have multiple kinds of workers
	for i := 0; i < threadiness; i++ {
		// runWorker will loop until "something bad" happens.  The .Until will
		// then rekick the worker after one second
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	// wait until we're told to stop
	<-stopCh
	glog.Infof("Shutting down <NAME> controller")
}

func (c *Controller) runWorker() {
	// hot loop until we're told to stop.  processNextWorkItem will
	// automatically wait until there's work available, so we don't worry
	// about secondary waits
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false
// when it's time to quit.
func (c *Controller) processNextWorkItem() bool {
	// pull the next work item from queue.  It should be a key we use to lookup
	// something in a cache
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// you always have to indicate to the queue that you've completed a piece of
	// work
	defer c.queue.Done(key)

	// do your work on the key.  This method will contains your "do stuff" logic
	err := c.syncHandler(key.(string))
	if err == nil {
		// if you had no error, tell the queue to stop tracking history for your
		// key. This will reset things like failure counts for per-item rate
		// limiting
		c.queue.Forget(key)
		return true
	}

	// there was a failure so be sure to report it.  This method allows for
	// pluggable error handling which can be used for things like
	// cluster-monitoring
	utilruntime.HandleError(fmt.Errorf("%v failed with : %v", key, err))

	// since we failed, we should requeue the item to work on later.  This
	// method will add a backoff to avoid hotlooping on particular items
	// (they're probably still not going to work right away) and overall
	// controller protection (everything I've done is broken, this controller
	// needs to calm down or it can starve other useful work) cases.
	c.queue.AddRateLimited(key)

	return true
}

func (c *Controller) syncHandler(key string) error {
	nodeset, err := c.nodesetLister.Get(key)
	if apierrors.IsNotFound(err) {
		glog.V(0).Infof("NodeSet %s was not found: %v", key, err)
		return nil
	}
	if err != nil {
		return err
	}

	if nodeset.Spec.NodeSetController != c.name {
		return nil
	}
	originalData, err := json.Marshal(nodeset)
	if err != nil {
		return fmt.Errorf("failed to marshal nodeset %s: %v", key, err)
	}

	changedNodeset, err := c.syncNodeSet(nodeset)
	if err != nil {
		return err
	}

	if changedNodeset != nil {
		if err = c.updateNodeSet(originalData, nodeset); err != nil {
			return err
		}
	}

	c.queue.AddAfter(nodeset.Name, 5*time.Second)
	return nil
}

func (c *Controller) updateNodeSet(originalData []byte, nodeset *v1alpha1.NodeSet) error {
	modifiedData, err := json.Marshal(nodeset)
	if err != nil {
		return err
	}

	patchData, err := jsonmergepatch.CreateThreeWayJSONMergePatch(nil, modifiedData, originalData)
	if err != nil {
		return err
	}
	//Avoid empty patch calls
	if string(patchData) == "{}" {
		return nil
	}

	_, err = c.nodesetClient.NodesetV1alpha1().NodeSets().Patch(nodeset.Name, types.MergePatchType, patchData)
	return err
}

const (
	kubeHostnameLabelKey    = "kubernetes.io/hostname"
	mergedNodeSetNamePrefix = "inherited-from-"
)

const (
	// UIDIndex is the name for the uid index function
	UIDIndex = "uid"
	// OwnerUIDIndex is the name for the owner uid index function
	OwnerUIDIndex = "owner-uid"
)

// MetaUIDIndexFunc indexes by uid.
func MetaUIDIndexFunc(obj interface{}) ([]string, error) {
	meta, err := meta.Accessor(obj)
	if err != nil {
		return []string{""}, fmt.Errorf("object has no meta: %v", err)
	}
	return []string{string(meta.GetUID())}, nil
}

// MetaOwnerUIDIndexFunc indexes by owner uid.
func MetaOwnerUIDIndexFunc(obj interface{}) ([]string, error) {
	node, ok := obj.(*corev1.Node)
	if !ok {
		return nil, nil
	}
	owner := metav1.GetControllerOf(node)
	if owner == nil {
		return nil, nil
	}
	return []string{string(owner.UID)}, nil
}

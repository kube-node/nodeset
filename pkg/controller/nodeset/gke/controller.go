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

package gke

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"

	gce "google.golang.org/api/compute/v1"
	gke "google.golang.org/api/container/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	nodesetclientset "github.com/kube-node/nodeset/pkg/client/clientset/versioned"
	nodesetinformers "github.com/kube-node/nodeset/pkg/client/informers/externalversions/nodeset/v1alpha1"
	nodesetlisters "github.com/kube-node/nodeset/pkg/client/listers/nodeset/v1alpha1"
	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	nodeAutoprovisioningPrefix    = "nodeautoprovisioning"
	gceURLSchema                  = "https"
	gceDomainSufix                = "googleapis.com/compute/v1/projects/"
	gkeControllerAnnotationPrefix = "gke.nodeset-controller.nodeset.k8s.io/"
	nodePoolAnnotationKey         = gkeControllerAnnotationPrefix + "node-pool"
	zoneAnnotationKey             = gkeControllerAnnotationPrefix + "zone"
	projectAnnotationKey          = gkeControllerAnnotationPrefix + "project"
	nodeSetFinalizer              = "gke.nodeset-controller.nodeset.k8s.io"
)

// Controller is a nodeset controller for GKE node pools
type Controller struct {
	nodesetClientset nodesetclientset.Interface
	nodesetLister    nodesetlisters.NodeSetLister
	nodesetIndexer   cache.Indexer
	nodesetsSynched  cache.InformerSynced

	name    string
	gke     *gke.Service
	gce     *gce.Service
	cluster Cluster

	// queue is where incoming work is placed to de-dup and to allow "easy"
	// rate limited requeues on errors
	queue workqueue.RateLimitingInterface
}

// New returns a instance of the GKE nodeset controller
func New(name string, clusterName string, client nodesetclientset.Interface, nodesets nodesetinformers.NodeSetInformer) (*Controller, error) {
	// index nodesets by uids
	// TODO: move outside of New
	err := nodesets.Informer().AddIndexers(map[string]cache.IndexFunc{
		UIDIndex: MetaUIDIndexFunc,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add uid indexer: %v", err)
	}

	c := &Controller{
		nodesetLister:   nodesets.Lister(),
		nodesetIndexer:  nodesets.Informer().GetIndexer(),
		nodesetsSynched: nodesets.Informer().HasSynced,

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

	// get GKE client
	c.gke, c.gce, c.cluster, err = NewGKEService(clusterName)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Run starts the control loop. This method is blocking.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) {
	// don't let panics crash the process
	defer utilruntime.HandleCrash()
	// make sure the work queue is shutdown which will trigger workers to end
	defer c.queue.ShutDown()

	glog.Infof("Starting <NAME> controller")

	// wait for your secondary caches to fill before starting your work
	if !cache.WaitForCacheSync(stopCh, c.nodesetsSynched) {
		return
	}

	// start up your worker threads based on threadiness.  Some controllers
	// have multiple kinds of workers
	for i := 0; i < threadiness; i++ {
		// runWorker will loop until "something bad" happens.  The .Until will
		// then rekick the worker after one second
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	go wait.Until(c.runReconciler, time.Minute, stopCh)

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
	// Cluster-monitoring
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
	ns, err := c.nodesetLister.Get(key)
	if apierrors.IsNotFound(err) {
		glog.V(0).Infof("Pod %s was not found: %v", key, err)
		return nil
	}
	if err != nil {
		return err
	}

	if ns.Spec.NodeSetController != c.name {
		return nil
	}

	glog.Infof("NodeSet %q seen in queue", ns.Name)

	// update instance group
	if ns.Spec.Replicas != ns.Status.Replicas {
		project := ns.Annotations[projectAnnotationKey]
		zone := ns.Annotations[zoneAnnotationKey]
		if project == "" || zone == "" {
			// skipping invalid NodeSet. Will be removed on next reconsilation.
			glog.Warningf("NodeSet %q cannot have empty project or zone.", ns.Name)
			return nil
		}
		comps := strings.SplitN(ns.Name, "-", 2)
		if comps[0] != zone {
			glog.Warningf("NodeSet %q has a zone mismatch, annotations say %q, but name say %q", ns.Name, project, comps[0])
			return nil
		}
		if len(comps) != 2 {
			glog.Warningf("Skipping invalid NodeSet %q.", ns.Name)
			return nil
		}
		name := comps[1]
		_, err := c.gce.InstanceGroupManagers.Resize(project, zone, name, int64(ns.Spec.Replicas)).Do()
		if err != nil {
			return err
		}
	}

	// update NodeSet
	newNS, err := c.updateNodeSetWithRetries(10, ns, func(old *v1alpha1.NodeSet) (*v1alpha1.NodeSet, error) {
		old.Status.Replicas = int32(ns.Spec.Replicas)
		old.Status.ObservedGeneration = ns.Generation
		return old, nil
	})
	if err != nil {
		return err
	}
	if ns.Spec.Replicas != newNS.Spec.Replicas {
		glog.Infof("NodeSet %q changed spec.replicas during controller run. Requeueing.", ns.Name)
		c.queue.Add(key)
	}

	return nil
}

func (c *Controller) runReconciler() {
	for c.reconsileNodeSets() {
		time.Sleep(5 * time.Minute)
	}
}

func (c *Controller) reconsileNodeSets() bool {
	// get all node sets
	nodeSets, err := c.nodesetLister.List(nil)
	if err != nil {
		glog.Warningf("Failed to list NodeSet: %v", err)
		return true
	}
	unseenNodeSets := sets.NewString()
	for _, ns := range nodeSets {
		unseenNodeSets.Insert(ns.Name)
	}

	resp, err := c.gke.Projects.Zones.Clusters.NodePools.List(c.cluster.Project, c.cluster.Zone, c.cluster.Name).Do()
	if err != nil {
		glog.Warningf("NodePool reconcile error: %v", err)
		return true
	}

	seenErrors := false
	for _, pool := range resp.NodePools {
		autoprovisioned := strings.Contains(pool.Name, nodeAutoprovisioningPrefix)
		if !autoprovisioned {
			continue
		}

		// format is
		// "https://www.googleapis.com/compute/v1/projects/mwielgus-proj/zones/europe-west1-b/instanceGroupManagers/gke-cluster-1-default-pool-ba78a787-grp"
		for _, url := range pool.InstanceGroupUrls {
			project, zone, name, err := parseGceURL(url, "instanceGroupManagers")
			if err != nil {
				glog.Warningf("parse error of %s/%s/%s InstanceGroupManager url %q", project, zone, name, url)
				return true
			}

			igm, err := c.gce.InstanceGroupManagers.Get(project, zone, name).Do()
			if err != nil {
				seenErrors = true
				glog.Warningf("Failed to get InstanceGroupManager %s/%s/%s", project, zone, name)
				continue
			}

			nsName := zone + "-" + name
			ns, err := c.nodesetLister.Get(nsName)
			if apierrors.IsNotFound(err) {
				one := intstr.FromInt(1)
				_, err = c.nodesetClientset.NodesetV1alpha1().NodeSets().Create(&v1alpha1.NodeSet{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							nodePoolAnnotationKey: pool.Name,
							projectAnnotationKey:  project,
							zoneAnnotationKey:     zone,
						},
						Name:       nsName,
						Finalizers: []string{nodeSetFinalizer},
					},
					Spec: v1alpha1.NodeSetSpec{
						NodeSetController: c.name,
						MaxSurge:          &one,
						Replicas:          int32(igm.TargetSize),
					},
					Status: v1alpha1.NodeSetStatus{
						Replicas: int32(igm.TargetSize),
						// TODO: set more of the fields
					},
				})
				continue
			} else if err != nil {
				glog.Warningf("Failed to get NodeSet %q: %v", name, err)
				continue
			}

			// we have seen this one
			unseenNodeSets.Delete(nsName)

			// check whether spec wasn't changed, if it was skip the reconsilation
			if ns.Spec.Replicas != ns.Spec.Replicas {
				continue
			}

			// update NodeSet
			_, err = c.updateNodeSetWithRetries(10, ns, func(old *v1alpha1.NodeSet) (*v1alpha1.NodeSet, error) {
				if old.Spec.Replicas == old.Status.Replicas {
					old.Spec.Replicas = int32(igm.TargetSize)
					old.Status.Replicas = int32(igm.TargetSize)
				}
				return old, nil
			})
			if err != nil {
				seenErrors = true
				glog.Warningf("Failed to update spec.replicas in NodeSet %q", nsName)
			}
		}
	}

	if seenErrors {
		return true
	}

	// delete unseen nodesets
	for _, nsName := range unseenNodeSets.List() {
		ns, err := c.nodesetLister.Get(nsName)
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			glog.Warningf("Failed to get NodeSet %q: %v", nsName, err)
			continue
		}

		// remove our finalizer. This also makes sure we will recreate the NodeSet if Google had a hickup
		// and didn't show us the InstanceGroupManager anymore. I.e. we will not try to delete the later.
		_, err = c.updateNodeSetWithRetries(10, ns, func(old *v1alpha1.NodeSet) (*v1alpha1.NodeSet, error) {
			if len(old.Finalizers) == 0 {
				return old, nil
			}
			newFinalizers := make([]string, len(old.Finalizers)-1)
			for _, f := range old.Finalizers {
				if f == nodeSetFinalizer {
					continue
				}
				newFinalizers = append(newFinalizers, f)
			}
			old.Finalizers = newFinalizers
			return old, nil
		})
		if err != nil {
			glog.Warningf("Failed to remove finalizer from NodeSet %q: %v", nsName, err)
			continue
		}

		if err = c.nodesetClientset.NodesetV1alpha1().NodeSets().Delete(nsName, nil); err != nil {
			glog.Warningf("Failed to delete NodeSet %q: %v", nsName, err)
		}
	}

	return true
}

func (c *Controller) updateNodeSetWithRetries(retries int, ns *v1alpha1.NodeSet, f func(*v1alpha1.NodeSet) (*v1alpha1.NodeSet, error)) (*v1alpha1.NodeSet, error) {
	nsName := ns.Name
	const initialRetries = 10
	for retries := initialRetries; retries > 0; retries = retries - 1 {
		var err error
		if retries < initialRetries {
			ns, err = c.nodesetLister.Get(ns.Name)
			if apierrors.IsNotFound(err) {
				return nil, err
			}
			if err != nil {
				glog.Warningf("Failed to get NodeSet %q, retrying up to %d more times: %v: %v", nsName, retries, err)
				continue
			}
		}

		ns, err = f(ns.DeepCopy())
		if err != nil {
			return nil, err
		}

		ns, err = c.nodesetClientset.NodesetV1alpha1().NodeSets().Update(ns)
		switch {
		case err == nil:
			return ns, nil
		case apierrors.IsNotFound(err):
			return nil, err
		default:
			glog.Warningf("Failed to update NodeSet %q, retrying up to %d more times: %v", nsName, retries, err)
		}
	}
	return nil, fmt.Errorf("giving up updating NodeSet %q after %d retries", nsName, initialRetries)
}

func parseGceURL(url, expectedResource string) (project string, zone string, name string, err error) {
	errMsg := fmt.Errorf("Wrong url: expected format https://content.googleapis.com/compute/v1/projects/<project-id>/zones/<zone>/%s/<name>, got %s", expectedResource, url)
	if !strings.Contains(url, gceDomainSufix) {
		return "", "", "", errMsg
	}
	if !strings.HasPrefix(url, gceURLSchema) {
		return "", "", "", errMsg
	}
	splitted := strings.Split(strings.Split(url, gceDomainSufix)[1], "/")
	if len(splitted) != 5 || splitted[1] != "zones" {
		return "", "", "", errMsg
	}
	if splitted[3] != expectedResource {
		return "", "", "", fmt.Errorf("Wrong resource in url: expected %s, got %s", expectedResource, splitted[3])
	}
	project = splitted[0]
	zone = splitted[2]
	name = splitted[4]
	return project, zone, name, nil
}

const (
	// UIDIndex is the name for the uid index function
	UIDIndex = "uid"
)

// MetaUIDIndexFunc indexes by uid.
func MetaUIDIndexFunc(obj interface{}) ([]string, error) {
	meta, err := meta.Accessor(obj)
	if err != nil {
		return []string{""}, fmt.Errorf("object has no meta: %v", err)
	}
	return []string{string(meta.GetUID())}, nil
}

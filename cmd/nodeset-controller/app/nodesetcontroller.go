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

package app

import (
	"math/rand"
	"time"

	"github.com/golang/glog"

	coreinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"

	"github.com/kube-node/nodeset/cmd/nodeset-controller/app/options"
	nodesetclientset "github.com/kube-node/nodeset/pkg/client/clientset/versioned"
	nodesetinformers "github.com/kube-node/nodeset/pkg/client/informers/externalversions"
	gkenodeset "github.com/kube-node/nodeset/pkg/controller/nodeset/gke"
	nodenodeset "github.com/kube-node/nodeset/pkg/controller/nodeset/node"
)

// ResyncPeriod returns a function which generates a duration each time it is
// invoked; this is so that multiple controllers don't get into lock-step and all
// hammer the apiserver with list requests simultaneously.
func ResyncPeriod(s *options.Options) func() time.Duration {
	return func() time.Duration {
		factor := rand.Float64() + 1
		return time.Duration(float64(s.MinResyncPeriod.Nanoseconds()) * factor)
	}
}

// Run runs the nodeset controller. It returns on error or when stopCh is closed.
func Run(s *options.Options, stopCh <-chan struct{}) error {
	if err := s.Validate(); err != nil {
		return err
	}

	kubeconfig, err := clientcmd.BuildConfigFromFlags("", s.Kubeconfig)
	if err != nil {
		return err
	}

	// Override kubeconfig qps/burst settings from flags
	kubeconfig.QPS = s.KubeAPIQPS
	kubeconfig.Burst = int(s.KubeAPIBurst)
	kubeClient, err := kubernetes.NewForConfig(restclient.AddUserAgent(kubeconfig, "nodeset-controller"))
	if err != nil {
		glog.Fatalf("Invalid API configuration: %v", err)
	}
	nodesetClient, err := nodesetclientset.NewForConfig(restclient.AddUserAgent(kubeconfig, "nodeset-controller"))
	if err != nil {
		glog.Fatalf("Invalid API configuration: %v", err)
	}

	/*
		eventBroadcaster := record.NewBroadcaster()
		eventBroadcaster.StartLogging(glog.Infof)
		eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(kubeClient.CoreV1().RESTClient()).Events("")})
		recorder := eventBroadcaster.NewRecorder(scheme.Scheme, clientv1.EventSource{Component: "controller-manager"})
	*/

	nodesetInformers := nodesetinformers.NewSharedInformerFactory(nodesetClient, ResyncPeriod(s)())
	nodesetInformers.Start(stopCh)

	glog.V(1).Infof("Starting NodeSet controller with %s backend", s.BackendName)

	switch s.BackendName {
	case "node":
		coreInformers := coreinformers.NewSharedInformerFactory(kubeClient, ResyncPeriod(s)())
		coreInformers.Start(stopCh)

		nodesetController := nodenodeset.New(s.ControllerName, nodesetInformers.Nodeset().V1alpha1().NodeSets(), nodesetInformers.Nodeset().V1alpha1().NodeClasses(), coreInformers.Core().V1().Nodes())
		nodesetController.Run(2, stopCh)
	case "gke":
		nodesetController, err := gkenodeset.New(s.ControllerName, s.GKEClusterName, nodesetClient, nodesetInformers.Nodeset().V1alpha1().NodeSets())
		if err != nil {
			return err
		}
		nodesetController.Run(2, stopCh)
	}

	return nil
}

type ControllerContext struct {
	// InformerFactory gives access to informers for the controller.
	InformerFactory nodesetinformers.SharedInformerFactory

	// corev1 event recorder
	eventRecorder record.EventRecorder

	// Options provides access to init options for a given controller
	Options options.Options

	// Stop channel
	StopCh <-chan struct{}
}

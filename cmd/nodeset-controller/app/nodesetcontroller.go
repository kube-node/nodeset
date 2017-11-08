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

	nodesetInformers := nodesetinformers.NewSharedInformerFactory(nodesetClient, ResyncPeriod(s)())
	coreInformers := coreinformers.NewSharedInformerFactory(kubeClient, ResyncPeriod(s)())
	glog.V(1).Infof("Starting NodeSet controller with %s backend", s.BackendName)
	switch s.BackendName {
	case "node":
		nodesetController, err := nodenodeset.New(s.ControllerName, kubeClient, nodesetClient, nodesetInformers.Nodeset().V1alpha1().NodeSets(), nodesetInformers.Nodeset().V1alpha1().NodeClasses(), coreInformers.Core().V1().Nodes())
		if err != nil {
			return err
		}
		go nodesetController.Run(2, stopCh)
	case "gke":
		nodesetController, err := gkenodeset.New(s.ControllerName, s.GKEClusterID, s.GKEClusterZone, s.GKEProjectID, nodesetClient, nodesetInformers.Nodeset().V1alpha1().NodeSets())
		if err != nil {
			return err
		}
		go nodesetController.Run(2, stopCh)
	}

	go nodesetInformers.Start(stopCh)
	go coreInformers.Start(stopCh)
	<-stopCh
	return nil
}
